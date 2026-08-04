// Package wrap runs another terminal program inside a pty we own, so we can
// take the screen away from it for a while and give it back intact.
//
// The rule the whole package is built around: we are never a second writer to
// the terminal. Either the host's bytes go through, or ours do — never both.
// That is what makes this safe to put in front of a tool people are actually
// working in, and it is why fighting the host TUI (the obvious approach) was
// the wrong one.
package wrap

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/jc-244/poemtea/internal/dbg"
	"github.com/jc-244/poemtea/internal/state"
	"golang.org/x/term"
)

type Wrapper struct {
	cmd  *exec.Cmd
	ptmx *os.File
	out  *os.File

	mu     sync.Mutex // serializes every write to the terminal
	muted  bool
	filter Filter

	// replay holds the drawing the host produced while the layer was up. It is
	// written back verbatim when the layer comes down, so what the user is left
	// with never depends on how completely we modelled the host.
	replay   []byte
	overflow bool

	// capturing records the host's output as well as forwarding it. Used once,
	// right after the layer goes up, to catch the host repainting itself into
	// the layer — those raw bytes become the backdrop every animation frame is
	// drawn on top of, so what is underneath the pixels is always the host's
	// own drawing rather than our idea of it.

	oldState *term.State
	restored bool

	Resized chan struct{}
	Exited  chan struct{}
	// Input fires when the user touches the keyboard. It is the escape hatch:
	// whatever is on the screen, typing must bring the host back. Without it a
	// stuck busy flag — a hook that never fired, an agent that crashed — leaves
	// someone staring at a picture they cannot dismiss, in the terminal they
	// were trying to work in.
	Input   chan struct{}
	keys    keyDetect
	waitErr error

	// Owner identifies the agent this wrapper launched. See internal/state.
	Owner string

	// lastOut is when the host last wrote anything, so an idle screen can be
	// told apart from one that is quietly filling with an answer.
	lastOut atomic.Int64
}

func newOwnerID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Any distinct string will do; uniqueness only has to hold among the
		// wrappers running right now.
		return "pid" + itoa(os.Getpid())
	}
	return hex.EncodeToString(b[:])
}

func Start(argv []string) (*Wrapper, error) {
	if len(argv) == 0 {
		return nil, errors.New("nothing to run")
	}

	w := &Wrapper{
		out:     os.Stdout,
		Resized: make(chan struct{}, 1),
		Exited:  make(chan struct{}),
		Input:   make(chan struct{}, 1),
	}
	w.filter.CursorVisible = true

	cols, rows, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || cols <= 0 || rows <= 0 {
		cols, rows = 80, 24
	}
	// The owner id is handed to the agent through the environment, where its
	// hooks inherit it and stamp it onto every state record they write. It is
	// what lets this wrapper answer only for the agent it launched, and what
	// replaces guessing at whether a hook's long-dead parent shell is alive.
	// Honour an id already in the environment so a session can be driven by
	// hand: POEMTEA_OWNER=test poemtea run -- claude, then POEMTEA_OWNER=test
	// poemtea busy from anywhere.
	w.Owner = os.Getenv(state.OwnerEnv)
	if w.Owner == "" {
		w.Owner = newOwnerID()
	}

	w.cmd = exec.Command(argv[0], argv[1:]...)
	w.cmd.Env = append(os.Environ(), state.OwnerEnv+"="+w.Owner)

	ptmx, err := pty.StartWithSize(w.cmd, &pty.Winsize{Rows: uint16(rows), Cols: uint16(cols)})
	if err != nil {
		return nil, err
	}
	w.ptmx = ptmx

	// The host expects a real terminal, so our stdin has to be raw and every
	// byte has to reach it untouched. Input is never filtered — the user must
	// always be able to interrupt whatever is happening.
	if st, err := term.MakeRaw(int(os.Stdin.Fd())); err == nil {
		w.oldState = st
	}

	w.lastOut.Store(time.Now().UnixNano())
	dbg.Printf("start %v pid=%d size=%dx%d owner=%s", argv, w.cmd.Process.Pid, cols, rows, w.Owner)

	go w.pumpInput()
	go w.pumpOutput()
	go w.watchResize()
	go w.watchSignals()
	go func() {
		w.waitErr = w.cmd.Wait()
		dbg.Printf("child exited: %v", w.waitErr)
		close(w.Exited)
	}()

	return w, nil
}

// pumpInput forwards the keyboard to the host, unchanged and unconditionally —
// input is never filtered, never buffered, never swallowed, whatever else this
// program is doing. It also reports that a key was pressed, which is the only
// signal that the user has come back.
func (w *Wrapper) pumpInput() {
	buf := make([]byte, 4096)
	for {
		n, err := os.Stdin.Read(buf)
		if n > 0 {
			// Forwarded first and unconditionally: whatever it is, the host
			// gets it untouched and immediately.
			_, _ = w.ptmx.Write(buf[:n])
			if w.keys.sawKey(buf[:n]) {
				select {
				case w.Input <- struct{}{}:
				default:
				}
			}
		}
		if err != nil {
			return
		}
	}
}

// pumpOutput is the one place the host's bytes are handled. They always reach
// the screen model; whether they reach the terminal depends on the mute state.
func (w *Wrapper) pumpOutput() {
	buf := make([]byte, 32*1024)
	fwd := make([]byte, 0, 4096)
	for {
		n, err := w.ptmx.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			w.lastOut.Store(time.Now().UnixNano())
			w.mu.Lock()
			fwd, w.replay = w.filter.Split(fwd[:0], w.replay, chunk, w.muted)
			if len(fwd) > 0 {
				_, _ = w.out.Write(fwd)
			}
			if len(w.replay) > replayCap {
				w.overflow = true
				w.replay = w.replay[:0]
			}
			w.mu.Unlock()
		}
		if err != nil {
			dbg.Printf("output pump ended: %v", err)
			return
		}
	}
}

// watchSignals is the difference between a terminal that survives being killed
// and one that does not. Ctrl-C never reaches us — raw mode turns it into a
// byte for the agent — but a kill, or closing the window, otherwise leaves the
// terminal in raw mode, on the alternate buffer, with no cursor.
func (w *Wrapper) watchSignals() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGHUP, os.Interrupt)
	sig := <-ch
	dbg.Printf("signal %v: restoring terminal", sig)
	w.Restore()
	if w.cmd.Process != nil {
		_ = w.cmd.Process.Signal(sig)
	}
	os.Exit(1)
}

func (w *Wrapper) watchResize() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	for range ch {
		cols, rows, err := term.GetSize(int(os.Stdout.Fd()))
		if err != nil || cols <= 0 || rows <= 0 {
			continue
		}
		_ = pty.Setsize(w.ptmx, &pty.Winsize{Rows: uint16(rows), Cols: uint16(cols)})
		select {
		case w.Resized <- struct{}{}:
		default:
		}
	}
}

// LastOutput reports when the host last drew anything.
func (w *Wrapper) LastOutput() time.Time { return time.Unix(0, w.lastOut.Load()) }

func (w *Wrapper) Size() (int, int) {
	cols, rows, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || cols <= 0 || rows <= 0 {
		return 80, 24
	}
	return cols, rows
}

// Mute takes the screen, painting firstFrame in the same breath.
//
// The frame is not optional and the caller must pass the host's current screen:
// if the host is not on the alternate buffer we push our own, and a fresh
// alternate buffer is blank. Switch first and paint on the next tick and the
// user gets a flash of empty terminal — which is exactly the kind of violence
// this program exists not to do. Switching and painting in one write makes the
// swap invisible, because both buffers show the same thing at the moment of the
// swap.
//
// Pushing our own buffer at all is about scrollback: an ambient animation
// scrolled into someone's transcript would be unforgivable.
// Mute raises the layer.
//
// This is the whole of it, and it is worth being clear about why it is so
// short. A terminal is one grid of cells: no z-order, no transparency, no way
// to read back what a cell holds, no undo. The single layer primitive the
// protocol has is the alternate buffer — a second whole grid you switch to and
// back — and it is all-or-nothing. Used as intended, showing and hiding the
// picture is two escape sequences and cannot lose anything, because nothing
// ever writes to the screen underneath.
func (w *Wrapper) Mute() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.muted {
		return
	}
	w.muted = true
	w.replay, w.overflow = w.replay[:0], false
	_, _ = w.out.WriteString("\x1b[?1049h\x1b[?25l")
}

// Unmute hands the screen back, painting finalFrame after the buffer is
// restored rather than before.
//
// The order matters and the other way round is wrong. While muted, the host's
// output was withheld, so if we pushed our own buffer the one underneath still
// holds the screen as it was when we took over. Painting into our buffer and
// then popping it reveals that stale screen. Popping first and then painting
// leaves the host's *current* screen, which is the only correct thing to hand
// back. Callers must Invalidate before rendering finalFrame — after a buffer
// swap, nothing about the previous frame is still on screen.
func (w *Wrapper) Unmute() {
	w.mu.Lock()
	if !w.muted {
		w.mu.Unlock()
		return
	}

	// Taking the layer away is the entire restore: the terminal puts the host's
	// screen back exactly as it was, because nothing ever wrote to it.
	var b strings.Builder
	b.WriteString("\x1b[?1049l")

	// Then the host's own bytes, uninterpreted, covering what it drew while it
	// was hidden. Nothing here is understood, so nothing can be lost.
	if !w.overflow {
		b.Write(w.replay)
	}
	if w.filter.CursorVisible {
		b.WriteString("\x1b[?25h")
	}
	_, _ = w.out.WriteString(b.String())

	dbg.Printf("unmute (replayed %d bytes, overflow=%v)", len(w.replay), w.overflow)
	overflowed := w.overflow
	w.replay, w.muted, w.overflow = w.replay[:0], false, false
	w.mu.Unlock()

	if overflowed {
		// The buffer was abandoned, so the screen the terminal just restored is
		// as stale as the moment we took it. Nobody can fix that but the host,
		// and a resize is the one way to ask: change the size and change it
		// back, and any well-behaved TUI repaints itself.
		//
		// This is a recovery path, not a mechanism. Relying on it normally
		// would mean a visible reflow every time; here it only runs when the
		// alternative is leaving the user with a screen that is minutes old.
		cols, rows := w.Size()
		_ = pty.Setsize(w.ptmx, &pty.Winsize{Rows: uint16(rows - 1), Cols: uint16(cols)})
		_ = pty.Setsize(w.ptmx, &pty.Winsize{Rows: uint16(rows), Cols: uint16(cols)})
	}
}

// replayCap bounds the held output. A whole Claude Code startup is three
// kilobytes and a long turn a few hundred; this exists only so a pathological
// host cannot exhaust memory.
const replayCap = 8 << 20

// Paint writes one of our frames. It takes the same lock as the output pump so
// a frame can never be interleaved with a forwarded query.
func (w *Wrapper) Paint(frame string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.muted {
		return
	}
	_, _ = w.out.WriteString(frame)
}

// Restore is the panic button. Anything that goes wrong anywhere should end up
// here: give the terminal back, whatever state we were in.
func (w *Wrapper) Restore() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.restored {
		return
	}
	w.restored = true
	state.ClearOwner(w.Owner)
	if w.muted {
		// The layer is always ours, so it always comes off. Getting this wrong
		// is the one failure the user cannot undo without `reset`.
		_, _ = w.out.WriteString("\x1b[?1049l")
		_, _ = w.out.WriteString("\x1b[0m\x1b[?25h")
		w.muted = false
	}
	if w.oldState != nil {
		_ = term.Restore(int(os.Stdin.Fd()), w.oldState)
	}
	_ = w.ptmx.Close()
}

func (w *Wrapper) ExitCode() int {
	var ee *exec.ExitError
	if errors.As(w.waitErr, &ee) {
		return ee.ExitCode()
	}
	if w.waitErr != nil {
		return 1
	}
	return 0
}

func itoa(n int) string {
	if n <= 0 {
		return "1"
	}
	var b [8]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
