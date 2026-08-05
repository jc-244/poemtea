package main

import (
	"fmt"
	"os"
	"time"

	"github.com/jc-244/poemtea/internal/poem"
	"github.com/jc-244/poemtea/internal/render"
	"github.com/jc-244/poemtea/internal/scene"
	"github.com/jc-244/poemtea/internal/tui"
)

const usage = `poemtea — your agent is running, and there is a poem on the screen

  poemtea run -- claude    run an agent; the screen turns into a picture while
                           it works, and turns back when it answers
  poemtea install-hooks    show the hook configuration needed for the above

  poemtea demo             play the scene full-screen (for working on the art)

  poemtea busy | idle      mark a session working or done (called by hooks)

While running, any keystroke gives the screen straight back to the host — the
picture never stands between you and the tool. Quit by quitting the host.

run flags:  -after 16     seconds of agent work before the picture appears
            -linger 1.5   seconds of quiet before it goes away
            -idle 30      seconds of nothing at all before it appears anyway
                          (0 turns the plain screensaver off)
demo keys:  n  next poem      q / esc / ctrl-c  quit
`

func main() {
	cmd := "demo"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	switch cmd {
	case "demo":
		if err := runDemo(); err != nil {
			fmt.Fprintln(os.Stderr, "poemtea:", err)
			os.Exit(1)
		}
	case "run":
		if err := runWrap(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "poemtea:", err)
			os.Exit(1)
		}
	case "busy", "idle":
		if err := runMark(cmd == "busy"); err != nil {
			// A hook must never break the tool it is attached to. Report and
			// succeed: a missing poem is not worth failing a turn over.
			fmt.Fprintln(os.Stderr, "poemtea:", err)
		}
	case "install-hooks":
		if err := runInstallHooks(); err != nil {
			fmt.Fprintln(os.Stderr, "poemtea:", err)
			os.Exit(1)
		}
	case "-h", "--help", "help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "poemtea: unknown command %q\n\n%s", cmd, usage)
		os.Exit(2)
	}
}

const (
	fps     = 20
	fadeIn  = 1.6 // seconds for the poem to arrive
	fadeOut = 1.2 // and to leave
	restFor = 3.0 // bare picture between poems, so the words are an event

	// cycle is one poem a minute, fade and gap included. It is the number
	// anyone actually means, so it is the one written down; how long the words
	// stand still follows from it.
	cycle   = 60.0
	holdFor = cycle - fadeOut - restFor
)

func runDemo() error {
	s, err := tui.Open()
	if err != nil {
		return err
	}
	defer s.Restore()

	keys := make(chan byte, 8)
	go func() {
		buf := make([]byte, 16)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil {
				return
			}
			for _, b := range buf[:n] {
				select {
				case keys <- b:
				default:
				}
			}
		}
	}()

	var sc scene.Scene = scene.NewTea()
	deck := poem.NewDeck(uint64(time.Now().UnixNano()))
	cur := deck.Next()

	w, h := s.Size()
	canvas := render.NewCanvas(w, h)
	grid := render.NewGrid(w, h)

	tick := time.NewTicker(time.Second / fps)
	defer tick.Stop()

	start := time.Now()
	last := start
	phase := 0.0 // seconds into the current poem's cycle

	for {
		select {
		case b := <-keys:
			switch b {
			case 'q', 3, 27: // q, ctrl-c, esc
				return nil
			case 'n':
				cur = deck.Next()
				phase = 0
			}
		case now := <-tick.C:
			dt := now.Sub(last).Seconds()
			last = now
			phase += dt

			select {
			case <-s.Resize:
				w, h = s.Size()
				canvas = render.NewCanvas(w, h)
				grid = render.NewGrid(w, h)
			default:
			}

			if nw, nh := s.Size(); nw != w || nh != h {
				w, h = nw, nh
				canvas = render.NewCanvas(w, h)
				grid = render.NewGrid(w, h)
			}

			sc.Draw(canvas, now.Sub(start).Seconds(), dt)
			canvas.ToGrid(grid)

			alpha := poemAlpha(phase)
			if phase > cycle {
				cur = deck.Next()
				phase = 0
			}
			render.DrawPoem(grid, cur.Lines, attribution(cur), alpha)

			s.Write(grid.Render())
		}
	}
}

// poemAlpha is a trapezoid: fade in, hold, fade out, then a gap of bare picture
// before the next one. The gap is the point — a poem that is always on screen
// stops being a poem and becomes wallpaper.
func poemAlpha(t float64) float64 {
	switch {
	case t < fadeIn:
		return ease(t / fadeIn)
	case t < holdFor:
		return 1
	case t < holdFor+fadeOut:
		return ease(1 - (t-holdFor)/fadeOut)
	default:
		return 0
	}
}

func ease(t float64) float64 {
	if t < 0 {
		return 0
	}
	if t > 1 {
		return 1
	}
	return t * t * (3 - 2*t)
}

func attribution(p poem.Poem) string {
	if p.Author == "" {
		return ""
	}
	return "— " + p.Author
}
