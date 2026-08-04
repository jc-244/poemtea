package scene

import "github.com/jc-244/poemtea/internal/render"

// Reveal assembles the picture out of a flat field, and takes it apart again.
//
// This is the dissolve, and the whole point is where it happens: on our own
// canvas, in our own buffer, made of pixels we computed. An earlier version
// dissolved the host's text into the picture instead, which meant knowing what
// every cell of the host's screen held — and paying for that knowledge with an
// emulated terminal, six dependencies, and a bug for every attribute the model
// failed to carry. The effect was never worth that. Applied to the rain rather
// than to somebody else's screen, it costs forty lines and cannot be wrong.
type Reveal struct {
	w, h  int
	order []float64 // per pixel, the point in [0,1] at which it arrives
	rnd   *rng
}

func NewReveal(seed uint64) *Reveal { return &Reveal{rnd: newRNG(seed)} }

func (r *Reveal) resize(w, h int) {
	r.w, r.h = w, h
	r.order = make([]float64, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			// Pure noise reads as television static. Mixing in a top-down
			// gradient makes it read as weather moving in rather than a signal
			// breaking up.
			sweep := float64(y) / float64(max(h-1, 1))
			r.order[y*w+x] = 0.45*sweep + 0.55*r.rnd.f()
		}
	}
}

// Apply holds back the pixels that have not arrived yet at progress p, leaving
// them at base. At p=0 the frame is a flat field; at p=1 it is the picture.
func (r *Reveal) Apply(c *render.Canvas, p float64, base render.RGB) {
	if c.W != r.w || c.H != r.h {
		r.resize(c.W, c.H)
	}
	if p >= 1 {
		return
	}
	for i := range c.Px {
		if at := r.order[i]; p < at {
			c.Px[i] = base
		} else if d := (p - at) / 0.12; d < 1 {
			// A short ramp so a pixel arrives rather than appearing.
			c.Px[i] = base.Blend(c.Px[i], d)
		}
	}
}

// Base is the colour the picture assembles out of: the darkest part of the sky,
// so the field the pixels arrive into already belongs to the scene.
var Base = render.RGB{R: 5, G: 7, B: 16}
