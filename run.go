package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/jc-244/poemtea/internal/dbg"
	"github.com/jc-244/poemtea/internal/poem"
	"github.com/jc-244/poemtea/internal/render"
	"github.com/jc-244/poemtea/internal/scene"
	"github.com/jc-244/poemtea/internal/state"
	"github.com/jc-244/poemtea/internal/wrap"
)

const (
	pollEvery = 250 * time.Millisecond
	revealIn  = 1.6 // seconds for the picture to assemble
	revealOut = 1.0 // and to come apart; shorter, because leaving should be eager

	// repaintEvery forces a full redraw now and then.
	//
	// Drawing only what changed assumes every byte we sent arrived and was
	// applied. On a terminal without synchronized output — macOS Terminal.app,
	// for one — a twenty-kilobyte frame lands in pieces, and anything that goes
	// astray is stale forever, because the diff believes those cells are already
	// correct. A periodic full frame costs one redraw every couple of seconds
	// and bounds how long any such damage can last.
	repaintEvery = 2 * time.Second
)

// runWrap runs the host program and puts the picture in front of it while it
// works.
//
// Two things happen at different levels, and keeping them separate is the whole
// design. Taking the screen is instantaneous and total: the alternate screen
// buffer is the only layer a terminal has, it is all-or-nothing, and used that
// way the host's screen cannot be damaged because nothing is ever written to
// it. The dissolve is not a transition between the host and the picture at all
// — it happens inside the picture, which assembles out of a flat field made of
// our own pixels.
//
// The obvious version, the host's text seen melting into the picture, needs to
// know what every cell of its screen holds, and a terminal will not tell you.
// Reconstructing it meant carrying palette identity, terminal defaults, style
// attributes, double-width state and grapheme clusters, and every one of them
// was found by shipping a bug onto a real screen. Moving the same effect into
// our own canvas costs forty lines and cannot be wrong.
func runWrap(args []string) error {
	after := 8.0  // how long it must be busy before the picture appears
	linger := 1.5 // how long it must be quiet before it goes away
	idle := 30.0  // how long everything must be still before it appears anyway

	var argv []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-after":
			i++
			after, _ = strconv.ParseFloat(args[i], 64)
		case "-linger":
			i++
			linger, _ = strconv.ParseFloat(args[i], 64)
		case "-idle":
			i++
			idle, _ = strconv.ParseFloat(args[i], 64)
		case "--":
			argv = args[i+1:]
			i = len(args)
		default:
			if argv == nil {
				argv = args[i:]
				i = len(args)
			}
		}
	}
	if len(argv) == 0 {
		return fmt.Errorf("usage: poemtea run [-after 8] [-linger 1.5] [-idle 30] -- <command>")
	}

	w, err := wrap.Start(argv)
	if err != nil {
		return err
	}
	defer w.Restore()

	seed := uint64(time.Now().UnixNano())
	cols, rows := w.Size()
	canvas := render.NewCanvas(cols, rows)
	grid := render.NewGrid(cols, rows)

	sc := scene.NewRain()
	rev := scene.NewReveal(seed)
	deck := poem.NewDeck(seed)
	cur := deck.Next()

	tick := time.NewTicker(time.Second / fps)
	defer tick.Stop()

	var (
		showing   bool
		leaving   bool
		reveal    float64 // 0 flat field, 1 full picture
		busySince time.Time
		idleSince time.Time
		showAt    float64 // seconds the picture has been up; drives the poem's fade
		lastPoll  time.Time
		lastFull  time.Time
		busy      bool
		start     = time.Now()
		last      = start
		lastKey   = start
	)

	// leave starts the picture coming apart. Both ways out use it — the agent
	// finishing and you touching the keyboard — because there is no reason for
	// them to look different. Your keystroke reached the agent the instant you
	// typed it; the only thing taking a second is the picture getting out of the
	// way, and it is visibly doing that the whole time.
	leave := func(why string) {
		if !showing || leaving {
			return
		}
		dbg.Printf("layer leaving: %s", why)
		leaving = true
	}

	// down is the moment the layer actually comes off.
	down := func() {
		w.Unmute()
		showing, leaving = false, false
		// Restart the clock: having just interrupted the picture, you get the
		// full quiet period again before it can return.
		busySince = time.Now()
	}

	for {
		select {
		case <-w.Exited:
			w.Restore()
			os.Exit(w.ExitCode())

		case <-w.Input:
			// A keystroke means you are back, so the picture starts leaving.
			// This is also the escape hatch: if the busy flag ever sticks — a
			// hook that never fired, an agent that crashed — any key gets your
			// terminal back.
			lastKey = time.Now()
			leave("you typed")

		case <-w.Resized:
			cols, rows = w.Size()
			canvas = render.NewCanvas(cols, rows)
			grid = render.NewGrid(cols, rows)

		case now := <-tick.C:
			dt := now.Sub(last).Seconds()
			last = now

			if now.Sub(lastPoll) >= pollEvery {
				lastPoll = now
				wasBusy := busy
				busy = state.AnyBusy(w.Owner)
				if busy && !wasBusy {
					busySince = now
				}
				if !busy && wasBusy {
					idleSince = now
				}
			}

			if !showing {
				// The busy signal spans a whole turn — from the prompt being
				// submitted to the turn ending — so the think/tool/think
				// oscillation inside it never surfaces here. What this threshold
				// decides is which turns are long enough to be worth covering
				// the screen for; a three-second answer never is.
				why := ""
				switch {
				case busy && !busySince.IsZero() && now.Sub(busySince).Seconds() >= after:
					why = "agent working"
				// And the plain screensaver case: nothing has moved at all for a
				// while. Requiring the host to be quiet too means this can never
				// cover an answer while it is still being written — only one you
				// have already stopped reading.
				case idle > 0 && now.Sub(lastKey).Seconds() >= idle &&
					now.Sub(w.LastOutput()).Seconds() >= idle:
					why = "nothing has moved"
				}
				if why != "" {
					dbg.Printf("layer up: %s", why)
					cur = deck.Next()
					showAt, reveal = 0, 0
					grid.Invalidate()
					w.Mute()
					showing, leaving = true, false
				}
				continue
			}

			showAt += dt
			if leaving {
				reveal -= dt / revealOut
				if reveal <= 0 {
					down()
					continue
				}
			} else if reveal < 1 {
				reveal += dt / revealIn
			}

			if now.Sub(lastFull) >= repaintEvery {
				lastFull = now
				grid.Invalidate()
			}

			sc.Draw(canvas, now.Sub(start).Seconds(), dt)
			// The dissolve lives here, in the picture's own pixels — never in
			// any idea of what the host had on screen.
			rev.Apply(canvas, reveal, scene.Base)
			canvas.ToGrid(grid)
			if showAt > holdFor+fadeOut+restFor {
				cur = deck.Next()
				showAt = 0
			}
			render.DrawPoem(grid, cur.Lines, attribution(cur), poemAlpha(showAt)*clamp01(reveal))
			w.Paint(grid.Render())

			if !busy && !idleSince.IsZero() && now.Sub(idleSince).Seconds() >= linger {
				leave("agent stopped working")
			}
		}
	}
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
