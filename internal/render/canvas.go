package render

// Canvas is a pixel buffer whose height is always twice the terminal row count.
// One character cell holds two vertically stacked pixels via the upper-half
// block: fg paints the top pixel, bg paints the bottom one.
//
// So a 100x30 terminal is a 100x60 canvas. That is roughly a Game Boy sprite
// scene — small, but pixel art was born at this size.
type Canvas struct {
	W, H int
	Px   []RGB
}

const halfBlock = '▀'

func NewCanvas(w, rows int) *Canvas {
	return &Canvas{W: w, H: rows * 2, Px: make([]RGB, w*rows*2)}
}

func (c *Canvas) Set(x, y int, col RGB) {
	if x < 0 || y < 0 || x >= c.W || y >= c.H {
		return
	}
	c.Px[y*c.W+x] = col
}

// Fill paints one whole row. Skies, water and table tops are all built a row
// at a time.
func (c *Canvas) Fill(y int, col RGB) {
	if y < 0 || y >= c.H {
		return
	}
	row := c.Px[y*c.W : (y+1)*c.W]
	for i := range row {
		row[i] = col
	}
}

// Add blends col over the existing pixel with weight a. Rain, glow and ripples
// all draw this way so they layer without hard edges.
func (c *Canvas) Add(x, y int, col RGB, a float64) {
	if x < 0 || y < 0 || x >= c.W || y >= c.H || a <= 0 {
		return
	}
	i := y*c.W + x
	c.Px[i] = c.Px[i].Blend(col, a)
}

// ToGrid folds the pixel buffer into character cells.
func (c *Canvas) ToGrid(g *Grid) {
	rows := c.H / 2
	for y := 0; y < rows && y < g.H; y++ {
		for x := 0; x < c.W && x < g.W; x++ {
			g.Set(x, y, Cell{
				Ch: halfBlock,
				Fg: c.Px[(y*2)*c.W+x],
				Bg: c.Px[(y*2+1)*c.W+x],
			})
		}
	}
}
