// Package scene draws the picture.
//
// One rule binds every scene: it must be an ambient loop with no narrative and
// no beginning or end. You have to be able to look away at any instant without
// having missed anything. That is the whole reason this can sit in front of you
// while you wait without becoming another thing demanding to be finished.
package scene

import (
	"math"

	"github.com/jc-244/poemtea/internal/render"
)

// Scene draws one frame. t is seconds since the picture went up and dt the
// time since the last frame, so a scene can animate either by absolute phase
// or by integration.
//
// Tea is the one that runs. Rain is still here and still built; swapping which
// one appears is the single call in runWrap and in the demo.
type Scene interface {
	Draw(c *render.Canvas, t, dt float64)
}

// Both pictures satisfy it. Rain is built but not drawn, so nothing would
// notice if it stopped satisfying it — these two lines are what keep swapping
// back a one-word change rather than a repair.
var (
	_ Scene = (*Tea)(nil)
	_ Scene = (*Rain)(nil)
)

// vignette darkens the edges. In a terminal this matters more than it does on
// a screen: it stops the art from looking like it was clipped by the window.
//
// from is how far out the darkening starts, as a fraction of the way to the
// corner, and depth is how much is taken at the very edge. Both scenes want
// the same falloff and different amounts of it.
func vignette(c *render.Canvas, from, depth float64) {
	cx, cy := float64(c.W)/2, float64(c.H)/2
	maxD := math.Hypot(cx, cy*0.75)
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			d := math.Hypot(float64(x)-cx, (float64(y)-cy)*0.75) / maxD
			if d < from {
				continue
			}
			f := (d - from) / (1 - from)
			i := y*c.W + x
			c.Px[i] = c.Px[i].Scale(1 - f*f*depth)
		}
	}
}

// rng is a small xorshift, so scenes stay self-contained and cheap. Nothing
// here needs cryptographic or even statistical quality.
type rng struct{ s uint64 }

func newRNG(seed uint64) *rng {
	if seed == 0 {
		seed = 0x9E3779B97F4A7C15
	}
	return &rng{s: seed}
}

func (r *rng) next() uint64 {
	r.s ^= r.s << 13
	r.s ^= r.s >> 7
	r.s ^= r.s << 17
	return r.s
}

// f returns a float in [0,1).
func (r *rng) f() float64 { return float64(r.next()>>11) / float64(1<<53) }

// rangeF returns a float in [lo,hi).
func (r *rng) rangeF(lo, hi float64) float64 { return lo + r.f()*(hi-lo) }
