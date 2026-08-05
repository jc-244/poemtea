package scene

import (
	"testing"

	"github.com/jc-244/poemtea/internal/render"
)

const testW, testRows = 120, 30

func differing(a, b *render.Canvas) int {
	n := 0
	for i := range a.Px {
		if a.Px[i] != b.Px[i] {
			n++
		}
	}
	return n
}

// The two moving parts of the tea scene are tested one at a time, on a canvas
// with nothing else drawn on it. Testing the whole frame proves only that
// something moved — the mist and the tea's surface both drift with t, so the
// steam and the drizzle could both have been frozen and it would still pass.
func TestSteamAndDrizzleEachMove(t *testing.T) {
	for _, tc := range []struct {
		name string
		step func(s *Tea, c *render.Canvas)
	}{
		{"steam", func(s *Tea, c *render.Canvas) { s.steam(c, 0.05) }},
		{"drizzle", func(s *Tea, c *render.Canvas) { s.drizzle(c, 0.05) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := NewTea()
			s.resize(testW, testRows*2)
			// A fresh canvas each frame: Add accumulates, so reusing one would
			// show a difference even if nothing had moved.
			a := render.NewCanvas(testW, testRows)
			tc.step(s, a)
			b := render.NewCanvas(testW, testRows)
			tc.step(s, b)
			if n := differing(a, b); n < 10 {
				t.Errorf("only %d pixels differ between frames; the %s is not moving", n, tc.name)
			}
		})
	}
}

// The drizzle falls outside the pavilion and stops at the table's edge. That
// line is the eave, and it is the only thing putting the viewer under a roof.
//
// The drizzle is drawn alone here, on a canvas with nothing else on it. Two
// separate things keep the rain off the table — this clamp, and the fact that
// the table is painted afterwards and would cover it anyway — so a whole-frame
// test cannot fail unless both are broken at once, which makes it no test at
// all. Alone, the rule is falsifiable.
func TestDrizzleStopsAtTheTablesEdge(t *testing.T) {
	s := NewTea()
	s.resize(testW, testRows*2)
	painted := 0
	for i := 0; i < 400; i++ {
		c := render.NewCanvas(testW, testRows)
		s.drizzle(c, 0.05)
		for y := int(s.tableY) + 1; y < c.H; y++ {
			for x := 0; x < c.W; x++ {
				if c.Px[y*c.W+x] != (render.RGB{}) {
					t.Fatalf("rain reached (%d,%d), below the table's edge at %.0f", x, y, s.tableY)
				}
			}
		}
		for _, px := range c.Px {
			if px != (render.RGB{}) {
				painted++
			}
		}
	}
	// And it did fall somewhere, or the check above passed for the wrong reason.
	if painted == 0 {
		t.Error("no rain was drawn at all")
	}
}
