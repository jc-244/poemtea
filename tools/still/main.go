// Command still renders one frame of a scene to a PNG.
//
//	go run ./tools/still -out docs/screenshot.png
//
// It exists so the picture in the README is made by the code that draws the
// real thing. A screenshot pasted in once drifts away from the art the first
// time anybody changes a colour, and nobody notices for a year.
//
// The pixels are the scene's own. The font is the one thing it cannot have: a
// terminal draws the glyph, so -poem fills each character cell with a bar of
// its ink colour instead. That is useful for checking the poem's contrast
// against the picture and useless as a picture of a poem, which is why it is
// off by default — the README shows the verse as text, where it is set by the
// reader's own font, as it is on your terminal.
package main

import (
	"flag"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"

	"github.com/jc-244/poemtea/internal/poem"
	"github.com/jc-244/poemtea/internal/render"
	"github.com/jc-244/poemtea/internal/scene"
)

func main() {
	var (
		cols  = flag.Int("cols", 120, "terminal columns")
		rows  = flag.Int("rows", 32, "terminal rows")
		zoom  = flag.Int("zoom", 8, "pixels per canvas pixel")
		which = flag.String("scene", "tea", "tea or rain")
		index = flag.Int("poem", -1, "index into the corpus, -1 for none")
		warm  = flag.Int("warm", 240, "frames to run first, so anything animated has settled")
		out   = flag.String("out", "still.png", "file to write")
	)
	flag.Parse()

	var sc scene.Scene = scene.NewTea()
	if *which == "rain" {
		sc = scene.NewRain()
	}

	canvas := render.NewCanvas(*cols, *rows)
	grid := render.NewGrid(*cols, *rows)
	for i := 0; i < *warm; i++ {
		sc.Draw(canvas, float64(i)*0.05, 0.05)
	}
	canvas.ToGrid(grid)

	if *index >= 0 {
		p := poem.Corpus[*index%len(poem.Corpus)]
		render.DrawPoem(grid, p.Lines, "— "+p.Author, 1)
	}

	img := image.NewRGBA(image.Rect(0, 0, *cols**zoom, *rows*2**zoom))
	block := func(cx, cy int, c render.RGB) {
		for dy := 0; dy < *zoom; dy++ {
			for dx := 0; dx < *zoom; dx++ {
				img.Set(cx**zoom+dx, cy**zoom+dy, color.RGBA{c.R, c.G, c.B, 255})
			}
		}
	}
	for y := 0; y < *rows; y++ {
		for x := 0; x < *cols; x++ {
			cell := grid.At(x, y)
			if cell.Ch == '▀' { // the scene: two pixels stacked in one cell
				block(x, y*2, cell.Fg)
				block(x, y*2+1, cell.Bg)
				continue
			}
			block(x, y*2, cell.Bg)
			block(x, y*2+1, cell.Bg)
			if cell.Ch != ' ' && cell.Ch != 0 {
				for dy := *zoom / 4; dy < *zoom*2-*zoom/4; dy++ {
					for dx := *zoom / 6; dx < *zoom-*zoom/6; dx++ {
						img.Set(x**zoom+dx, y*2**zoom+dy,
							color.RGBA{cell.Fg.R, cell.Fg.G, cell.Fg.B, 255})
					}
				}
			}
		}
	}

	f, err := os.Create(*out)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		log.Fatal(err)
	}
	log.Printf("wrote %s (%dx%d)", *out, img.Bounds().Dx(), img.Bounds().Dy())
}
