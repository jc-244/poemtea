// Package scene draws the picture.
//
// One rule binds every scene: it must be an ambient loop with no narrative and
// no beginning or end. You have to be able to look away at any instant without
// having missed anything. That is the whole reason this can sit in front of you
// while you wait without becoming another thing demanding to be finished.
package scene

import "github.com/jc-244/poemtea/internal/render"

// Scene draws one frame. t is seconds since the picture went up and dt the
// time since the last frame, so a scene can animate either by absolute phase
// or by integration.
type Scene interface {
	Draw(c *render.Canvas, t, dt float64)
}

// All returns one of each picture. They take turns rather than being chosen at
// random: with two of them, random means seeing the same one twice about half
// the time, which reads as the other one being broken.
func All() []Scene { return []Scene{NewTea(), NewRain()} }

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
