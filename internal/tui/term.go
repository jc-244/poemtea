// Package tui owns the raw terminal: alt screen, raw mode, resize, and — most
// importantly — putting all of it back. This program draws over the screen of a
// tool the user is actually working in, so a crash that leaves the terminal in
// raw mode with a hidden cursor is worse than the program never having run.
package tui

import (
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/term"
)

const (
	enterAlt  = "\x1b[?1049h"
	leaveAlt  = "\x1b[?1049l"
	hideCur   = "\x1b[?25l"
	showCur   = "\x1b[?25h"
	resetAttr = "\x1b[0m"
	clearAll  = "\x1b[2J\x1b[H"
)

type Session struct {
	out      *os.File
	oldState *term.State
	restored bool
	Resize   chan struct{}
	stopSig  chan os.Signal
}

// Open takes over the terminal. Restore is registered against SIGINT/SIGTERM as
// well as the normal defer path, because the common way to quit anything in a
// terminal is Ctrl-C.
func Open() (*Session, error) {
	s := &Session{out: os.Stdout, Resize: make(chan struct{}, 1)}

	st, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return nil, err
	}
	s.oldState = st

	s.out.WriteString(enterAlt + hideCur + clearAll)

	winch := make(chan os.Signal, 1)
	signal.Notify(winch, syscall.SIGWINCH)
	go func() {
		for range winch {
			select {
			case s.Resize <- struct{}{}:
			default:
			}
		}
	}()

	s.stopSig = make(chan os.Signal, 1)
	signal.Notify(s.stopSig, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-s.stopSig
		s.Restore()
		os.Exit(0)
	}()

	return s, nil
}

func (s *Session) Size() (w, h int) {
	w, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w <= 0 || h <= 0 {
		return 80, 24
	}
	return w, h
}

func (s *Session) Write(frame string) { s.out.WriteString(frame) }

func (s *Session) Restore() {
	if s.restored {
		return
	}
	s.restored = true
	s.out.WriteString(resetAttr + showCur + leaveAlt)
	if s.oldState != nil {
		term.Restore(int(os.Stdin.Fd()), s.oldState)
	}
}
