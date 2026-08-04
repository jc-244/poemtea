package scene

import (
	"math"

	"github.com/jc-244/poemtea/internal/render"
)

// Rain: night sky, two hill silhouettes, one lit window, slanted rain, and a
// water band that reflects the window. Everything is procedural — no assets,
// no licensing, infinite and never exactly repeating.
type Rain struct {
	w, h  int
	drops []drop
	rings []ring
	rnd   *rng
	winX  float64 // window position, in pixels
	winY  float64
}

type drop struct {
	x, y   float64
	vy     float64
	length float64
	bright float64
}

type ring struct {
	x, y float64
	r    float64
	life float64
}

var (
	skyTop  = render.RGB{R: 5, G: 7, B: 16}
	skyLow  = render.RGB{R: 30, G: 36, B: 62}
	hillFar = render.RGB{R: 20, G: 25, B: 46}
	hillNr  = render.RGB{R: 5, G: 6, B: 14}
	cabin   = render.RGB{R: 3, G: 4, B: 9}
	rainCol = render.RGB{R: 150, G: 175, B: 210}
	warm    = render.RGB{R: 255, G: 186, B: 104}
)

func NewRain() *Rain { return &Rain{rnd: newRNG(0xC0FFEE11)} }

func (r *Rain) resize(w, h int) {
	r.w, r.h = w, h
	n := (w * h) / 60
	if n < 60 {
		n = 60
	}
	r.drops = make([]drop, n)
	for i := range r.drops {
		r.drops[i] = r.spawn(true)
	}
	r.rings = r.rings[:0]
	// The cabin sits off-center. Dead center reads as a diagram; off-center
	// reads as a place.
	r.winX = float64(w) * 0.30
	r.winY = r.nearTop(r.winX) - float64(h)*0.035
}

func (r *Rain) horizon() float64 { return float64(r.h) * 0.68 }

// nearTop is the skyline of the near hill — the ground everything stands on.
func (r *Rain) nearTop(x float64) float64 {
	fx := x / float64(r.w)
	return r.horizon() - float64(r.h)*(0.055+0.045*math.Sin(fx*3.3+2.2)+0.020*math.Sin(fx*7.9+0.4))
}

func (r *Rain) farTop(x float64) float64 {
	fx := x / float64(r.w)
	return r.horizon() - float64(r.h)*(0.130+0.085*math.Sin(fx*4.7+0.8)+0.035*math.Sin(fx*11.3))
}

func (r *Rain) spawn(anywhere bool) drop {
	y := -r.rnd.rangeF(0, float64(r.h)*0.4)
	if anywhere {
		y = r.rnd.rangeF(0, r.horizon())
	}
	return drop{
		x:      r.rnd.rangeF(-float64(r.h)*0.3, float64(r.w)),
		y:      y,
		vy:     r.rnd.rangeF(30, 64),
		length: r.rnd.rangeF(2, 5),
		bright: r.rnd.rangeF(0.07, 0.30),
	}
}

// slant is how far a drop drifts horizontally per unit of fall. Rain that
// falls straight down looks like static; a slight lean reads as weather.
const slant = 0.34

func (r *Rain) Draw(c *render.Canvas, t, dt float64) {
	if c.W != r.w || c.H != r.h {
		r.resize(c.W, c.H)
	}
	hz := r.horizon()

	r.drawSky(c, t, hz)
	r.drawHills(c, hz)
	flicker := r.drawWindow(c, t)
	r.drawWater(c, t, hz, flicker)
	r.drawRain(c, dt, hz)
	r.drawRings(c, dt, hz)
	r.vignette(c)
}

func (r *Rain) drawSky(c *render.Canvas, t, hz float64) {
	for y := 0; y < c.H; y++ {
		f := float64(y) / hz
		if f > 1 {
			f = 1
		}
		// eased so most of the gradient lives near the horizon
		col := skyTop.Blend(skyLow, f*f)
		for x := 0; x < c.W; x++ {
			c.Set(x, y, col)
		}
	}
	// slow cloud bands, barely visible — they exist so the sky is never
	// perfectly flat, not so you notice them
	for y := 0; y < int(hz); y++ {
		fy := float64(y)
		band := math.Sin(fy*0.09+t*0.07) * math.Sin(fy*0.021-t*0.04)
		if band <= 0 {
			continue
		}
		a := band * 0.05
		for x := 0; x < c.W; x++ {
			c.Add(x, y, render.RGB{R: 40, G: 45, B: 70}, a)
		}
	}
}

func (r *Rain) drawHills(c *render.Canvas, hz float64) {
	for x := 0; x < c.W; x++ {
		fx := float64(x)
		// The far ridge is lighter than the sky it sits against, the near one
		// almost black. Aerial perspective is the only depth cue available at
		// this resolution — there is no room for detail to do the work.
		for y := int(r.farTop(fx)); y < int(hz); y++ {
			c.Set(x, y, hillFar)
		}
		for y := int(r.nearTop(fx)); y < int(hz); y++ {
			c.Set(x, y, hillNr)
		}
	}
	r.drawCabin(c)
}

// drawCabin gives the light something to come out of. Without it the window is
// a glowing rectangle floating over a hill, which reads as an error.
func (r *Rain) drawCabin(c *render.Canvas) {
	ww, wh := 3, 2
	if c.W > 90 {
		ww, wh = 4, 3
	}
	bw := ww * 3          // body width
	bh := wh*2 + wh/2 + 2 // body height
	bx := int(r.winX) - (bw-ww)/2
	by := int(r.winY) - wh/2 - 1

	for y := by; y < by+bh; y++ {
		for x := bx; x < bx+bw; x++ {
			c.Set(x, y, cabin)
		}
	}
	// pitched roof, one pixel step per column from each edge
	rh := bw / 3
	for i := 0; i < rh; i++ {
		for x := bx + i; x < bx+bw-i; x++ {
			c.Set(x, by-1-i, cabin)
		}
	}
}

// drawWindow returns the current flicker factor so the water reflection can
// breathe in sync with it.
func (r *Rain) drawWindow(c *render.Canvas, t float64) float64 {
	flicker := 0.86 + 0.14*math.Sin(t*2.3) + 0.05*math.Sin(t*7.1+1.3)
	col := warm.Scale(flicker)

	wx, wy := int(r.winX), int(r.winY)
	ww, wh := 3, 2
	if c.W > 90 {
		ww, wh = 4, 3
	}
	for y := wy; y < wy+wh; y++ {
		for x := wx; x < wx+ww; x++ {
			c.Set(x, y, col)
		}
	}
	// halo — cheap radial falloff, but it is what sells the window as a light
	// source instead of an orange rectangle
	rad := float64(wh) * 5
	for y := wy - int(rad); y <= wy+wh+int(rad); y++ {
		for x := wx - int(rad); x <= wx+ww+int(rad); x++ {
			dx := float64(x) - (r.winX + float64(ww)/2)
			dy := (float64(y) - (r.winY + float64(wh)/2)) * 2.0
			d := math.Hypot(dx, dy)
			if d > rad || d < 0.001 {
				continue
			}
			a := (1 - d/rad)
			c.Add(x, y, warm, a*a*a*0.55*flicker)
		}
	}
	return flicker
}

func (r *Rain) drawWater(c *render.Canvas, t, hz, flicker float64) {
	depth := float64(c.H) - hz
	if depth <= 0 {
		return
	}
	for y := int(hz); y < c.H; y++ {
		d := (float64(y) - hz) / depth // 0 at shore, 1 at bottom
		// water is the sky, darkened and pulled toward black with depth
		base := skyLow.Blend(render.RGB{R: 3, G: 4, B: 10}, d*0.85).Scale(0.75)
		for x := 0; x < c.W; x++ {
			c.Set(x, y, base)
		}
	}
	// the window's reflection: a vertical smear that wobbles, widening as it
	// comes toward you
	for y := int(hz); y < c.H; y++ {
		d := (float64(y) - hz) / depth
		wob := math.Sin(float64(y)*0.55-t*2.1)*1.6 + math.Sin(float64(y)*0.23+t*1.1)*2.4
		cx := r.winX + 1.5 + wob*d*2.2
		width := 1.2 + d*4.5
		a := (1 - d) * 0.5 * flicker
		for x := int(cx - width); x <= int(cx+width); x++ {
			f := 1 - math.Abs(float64(x)-cx)/width
			if f <= 0 {
				continue
			}
			c.Add(x, y, warm, a*f*f)
		}
	}
	// horizontal ripple lines drifting toward the viewer
	for y := int(hz); y < c.H; y++ {
		d := (float64(y) - hz) / depth
		phase := float64(y)*0.9 - t*1.4
		v := math.Sin(phase)
		if v <= 0.55 {
			continue
		}
		a := (v - 0.55) / 0.45 * 0.10 * (0.3 + d)
		for x := 0; x < c.W; x++ {
			c.Add(x, y, render.RGB{R: 120, G: 140, B: 180}, a)
		}
	}
}

func (r *Rain) drawRain(c *render.Canvas, dt, hz float64) {
	for i := range r.drops {
		d := &r.drops[i]
		d.y += d.vy * dt
		d.x += d.vy * dt * slant

		if d.y > hz {
			// a drop that reaches the water leaves a ring behind it
			if r.rnd.f() < 0.16 && len(r.rings) < 18 {
				r.rings = append(r.rings, ring{x: d.x, y: hz + r.rnd.rangeF(0, float64(c.H)-hz), r: 0, life: 1})
			}
			*d = r.spawn(false)
			continue
		}
		if d.x > float64(c.W) {
			*d = r.spawn(false)
			continue
		}
		for k := 0.0; k < d.length; k++ {
			y := int(d.y - k)
			x := int(d.x - k*slant)
			// the head of the streak is brightest; the tail fades out
			a := d.bright * (1 - k/d.length)
			c.Add(x, y, rainCol, a)
		}
	}
}

func (r *Rain) drawRings(c *render.Canvas, dt, hz float64) {
	out := r.rings[:0]
	for _, rg := range r.rings {
		rg.r += dt * 7
		rg.life -= dt * 1.5
		if rg.life <= 0 {
			continue
		}
		// Drawn as a very flat ellipse: water is seen at a glancing angle, so a
		// ring is nearly a horizontal line. An ellipse tall enough to read as a
		// ring reads instead as a dead pixel cluster — which is exactly what the
		// first version looked like.
		for a := 0.0; a < math.Pi*2; a += 0.16 {
			x := int(rg.x + math.Cos(a)*rg.r)
			y := int(rg.y + math.Sin(a)*rg.r*0.13)
			if float64(y) < hz {
				continue
			}
			c.Add(x, y, render.RGB{R: 140, G: 160, B: 195}, rg.life*rg.life*0.055)
		}
		out = append(out, rg)
	}
	r.rings = out
}

// vignette darkens the edges. In a terminal this matters more than it does on
// a screen: it stops the art from looking like it was clipped by the window.
func (r *Rain) vignette(c *render.Canvas) {
	cx, cy := float64(c.W)/2, float64(c.H)/2
	maxD := math.Hypot(cx, cy*0.75)
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			d := math.Hypot(float64(x)-cx, (float64(y)-cy)*0.75) / maxD
			if d < 0.55 {
				continue
			}
			f := (d - 0.55) / 0.45
			i := y*c.W + x
			c.Px[i] = c.Px[i].Scale(1 - f*f*0.75)
		}
	}
}
