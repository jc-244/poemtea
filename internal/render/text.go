package render

// Text is drawn *after* the canvas has been folded into cells, never into the
// pixel buffer. At 1x2 subpixels a letter is illegible; the image and the words
// have to live at different resolutions in the same grid.

var (
	Ink   = RGB{R: 235, G: 229, B: 214}
	Faint = RGB{R: 138, G: 134, B: 126}
)

// DrawPoem places the lines low in the frame with the attribution beneath, and
// darkens a band behind them so the words stay readable over rain or water.
// alpha drives the fade; at 0 nothing is drawn at all.
func DrawPoem(g *Grid, lines []string, attribution string, alpha float64) {
	if alpha <= 0.001 || len(lines) == 0 {
		return
	}
	if alpha > 1 {
		alpha = 1
	}

	block := len(lines)
	if attribution != "" {
		block += 2
	}
	top := int(float64(g.H)*0.62) - block/2
	if top < 1 {
		top = 1
	}
	if top+block >= g.H {
		top = g.H - block - 1
	}

	// The scrim fades in slightly ahead of the text, so the image dims first
	// and the words arrive into a space already made for them.
	scrim := alpha * 1.15
	if scrim > 1 {
		scrim = 1
	}
	dim(g, top-2, top+block+2, scrim*0.72)

	for i, ln := range lines {
		putLine(g, ln, top+i, Ink, alpha)
	}
	if attribution != "" {
		putLine(g, attribution, top+len(lines)+1, Faint, alpha*0.8)
	}
}

// dim darkens every cell in a row range, with a soft falloff at the two edge
// rows so the band does not read as a rectangle pasted over the art.
func dim(g *Grid, y0, y1 int, amount float64) {
	for y := y0; y < y1; y++ {
		if y < 0 || y >= g.H {
			continue
		}
		f := amount
		switch {
		case y == y0 || y == y1-1:
			f *= 0.35
		case y == y0+1 || y == y1-2:
			f *= 0.7
		}
		for x := 0; x < g.W; x++ {
			c := g.At(x, y)
			c.Fg = c.Fg.Scale(1 - f*0.8)
			c.Bg = c.Bg.Scale(1 - f*0.8)
		}
	}
}

// putLine centres a line and writes it into the grid.
//
// Centring is by display width, not by count of characters: a line of Chinese
// is half as many characters as it is columns wide, and counting characters
// would push it a quarter of the screen off to the right.
func putLine(g *Grid, s string, y int, col RGB, alpha float64) {
	x := (g.W - StringWidth(s)) / 2
	if x < 0 {
		x = 0
	}
	for _, ch := range s {
		w := RuneWidth(ch)
		// A double-width glyph is never split across the right edge: half a
		// character is worse than no character.
		if x+w > g.W {
			break
		}
		c := g.At(x, y)
		if c == nil {
			break
		}
		// The cell's two half-pixels average into a single background, then the
		// glyph is blended up out of it — so the text emerges from the image
		// rather than being stamped on top.
		bg := c.Fg.Blend(c.Bg, 0.5)
		if ch == ' ' {
			c.Ch, c.Fg, c.Bg = ' ', bg, bg
		} else {
			c.Ch = ch
			c.Bg = bg
			c.Fg = bg.Blend(col, alpha)
		}
		// The second column of a wide glyph is the terminal's to paint, and it
		// takes the background of the cell the glyph was written into — so a
		// Chinese character sits on one flat colour two columns wide rather
		// than on two samples of the image.
		if w == 2 {
			if n := g.At(x+1, y); n != nil {
				n.Ch = contd
			}
		}
		x += w
	}
}
