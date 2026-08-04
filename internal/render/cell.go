package render

import "strings"

// Everything here draws only our own picture. It never has to describe,
// reproduce or repaint the host's screen — the terminal keeps that on the other
// buffer — so a cell is a character and two colours, and nothing else.
//
// It was not always this small. An earlier version modelled the host's screen
// so its text could be seen dissolving into the picture, and paid for it: cells
// had to carry palette identity, terminal defaults, seven style attributes,
// five underline styles, double-width state and grapheme clusters. Every one of
// those was discovered by shipping a bug — the terminal turned black, a themed
// palette was flattened to RGB, Chinese lines shifted a column, a faint
// placeholder came back looking like text the user had typed.
//
// All of it existed to serve a two-second animation. None of it is needed to
// put a picture on the screen and take it off again.
type RGB struct{ R, G, B uint8 }

func (c RGB) Blend(o RGB, t float64) RGB {
	if t <= 0 {
		return c
	}
	if t >= 1 {
		return o
	}
	return RGB{
		R: uint8(float64(c.R) + (float64(o.R)-float64(c.R))*t),
		G: uint8(float64(c.G) + (float64(o.G)-float64(c.G))*t),
		B: uint8(float64(c.B) + (float64(o.B)-float64(c.B))*t),
	}
}

func (c RGB) Scale(f float64) RGB {
	clamp := func(v float64) uint8 {
		if v < 0 {
			return 0
		}
		if v > 255 {
			return 255
		}
		return uint8(v)
	}
	return RGB{clamp(float64(c.R) * f), clamp(float64(c.G) * f), clamp(float64(c.B) * f)}
}

// Cell is one character position.
//
// The scene emits '▀' with fg as the upper pixel and bg as the lower one; the
// poem is written as real runes into the same grid. Compositing at cell level
// rather than pixel level is what keeps text legible over a 1x2-subpixel image.
type Cell struct {
	Ch rune
	Fg RGB
	Bg RGB
}

type Grid struct {
	W, H    int
	Cells   []Cell
	prev    []Cell
	hasPrev bool
}

func NewGrid(w, h int) *Grid {
	g := &Grid{W: w, H: h, Cells: make([]Cell, w*h), prev: make([]Cell, w*h)}
	for i := range g.Cells {
		g.Cells[i] = Cell{Ch: ' '}
	}
	return g
}

// Invalidate forces the next Render to redraw every cell.
func (g *Grid) Invalidate() { g.hasPrev = false }

func (g *Grid) At(x, y int) *Cell {
	if x < 0 || y < 0 || x >= g.W || y >= g.H {
		return nil
	}
	return &g.Cells[y*g.W+x]
}

func (g *Grid) Set(x, y int, c Cell) {
	if p := g.At(x, y); p != nil {
		*p = c
	}
}

// Render serializes the grid, emitting only what changed since the last call.
//
// Three things keep the byte count survivable:
//   - unchanged cells are skipped. This is the big one: most of an ambient
//     frame is identical to the last, and only rain and water actually move.
//   - colours are re-emitted only when they differ from the current pen
//   - cursor moves are emitted only when a run breaks, since consecutive
//     changed cells advance the cursor by themselves
//
// The frame is wrapped in synchronized-output markers (?2026) so terminals that
// understand them swap it in one go instead of tearing; the rest ignore it.
func (g *Grid) Render() string {
	var b strings.Builder
	b.Grow(g.W * g.H * 4)
	b.WriteString("\x1b[?2026h")

	var curFg, curBg RGB
	var havePen bool
	cx, cy := -1, -1 // where the terminal cursor actually is

	for y := 0; y < g.H; y++ {
		for x := 0; x < g.W; x++ {
			i := y*g.W + x
			c := g.Cells[i]
			if g.hasPrev && indistinguishable(c, g.prev[i]) {
				continue
			}
			if cy != y || cx != x {
				b.WriteString("\x1b[")
				b.WriteString(itoa(y + 1))
				b.WriteByte(';')
				b.WriteString(itoa(x + 1))
				b.WriteByte('H')
				cy = y
			}
			if !havePen || c.Fg != curFg {
				writeColor(&b, 38, c.Fg)
				curFg = c.Fg
			}
			if !havePen || c.Bg != curBg {
				writeColor(&b, 48, c.Bg)
				curBg = c.Bg
			}
			havePen = true
			if c.Ch == 0 {
				b.WriteRune(' ')
			} else {
				b.WriteRune(c.Ch)
			}
			cx = x + 1
			// prev records what was actually sent, so the tolerance below never
			// accumulates: a cell drifting slowly eventually crosses it and is
			// corrected.
			g.prev[i] = c
		}
	}

	b.WriteString("\x1b[0m\x1b[?2026l")
	g.hasPrev = true
	return b.String()
}

func writeColor(b *strings.Builder, base int, c RGB) {
	b.WriteString("\x1b[")
	b.WriteString(itoa(base))
	b.WriteString(";2;")
	b.WriteString(itoa(int(c.R)))
	b.WriteByte(';')
	b.WriteString(itoa(int(c.G)))
	b.WriteByte(';')
	b.WriteString(itoa(int(c.B)))
	b.WriteByte('m')
}

// tolerance is how far a channel may drift before a cell is worth redrawing.
// The scenes animate the whole sky by fractions of a level per frame, which
// under exact comparison dirties nearly every cell and defeats the diff. Two
// parts in 255 is not visible on any display.
const tolerance = 2

func indistinguishable(a, b Cell) bool {
	return a.Ch == b.Ch && closeRGB(a.Fg, b.Fg) && closeRGB(a.Bg, b.Bg)
}

func closeRGB(a, b RGB) bool {
	return near(a.R, b.R) && near(a.G, b.G) && near(a.B, b.B)
}

func near(a, b uint8) bool {
	if a > b {
		return a-b <= tolerance
	}
	return b-a <= tolerance
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [8]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
