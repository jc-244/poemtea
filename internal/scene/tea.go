package scene

import (
	"math"

	"github.com/jc-244/poemtea/internal/render"
)

// Tea: a 紫砂 pot on a wooden table, its lid off and leaning against it, and
// the mountains a long way past the edge of the table. Two lacquered pillars
// stand at the frame's edges — that, and the beam across the top, is all that
// says you are under a pavilion roof rather than out in the open.
//
// The composition follows the rain's: the pot sits a quarter of the way in,
// not in the middle. The poem is drawn centred, so the middle of the frame is
// spoken for, and a thing in the exact centre reads as a diagram anyway.
//
// At this size a shape is read from its silhouette and almost nothing else, so
// the pot is drawn as an ovoid with a spout and a handle rather than as a
// cylinder with shading. A cylinder reads as a tin. The pillars are vermilion
// for the same reason: two near-black bands at the edges read as a letterbox
// crop, not as columns.
//
// Everything moves except the furniture: steam off the mouth, mist along the
// ridge line, the tea breathing where the light lands on it. Nothing begins or
// ends, so there is nothing to have missed.
type Tea struct {
	w, h  int
	rnd   *rng
	puffs []puff

	drops  []streak
	tableY float64 // the table's far edge
	potX   float64 // centre of the pot
	rimY   float64 // the plane of the mouth
	potW   float64
	potH   float64
	pillar float64
}

// streak is one thread of the drizzle outside. It falls only in the view
// beyond the table, never on the table or the pot: the line where it stops is
// the eave, and that line is what puts you under the roof.
type streak struct {
	x, y   float64
	vy     float64
	length float64
	bright float64
}

// puff is one wisp of steam. It rises, widens and comes apart; there is never
// a last one.
type puff struct {
	x, y   float64
	vy     float64
	drift  float64
	age    float64
	life   float64
	radius float64
}

var (
	duskTop = render.RGB{R: 24, G: 30, B: 46}
	duskLow = render.RGB{R: 176, G: 152, B: 124} // the glow the ridges stand against
	ridge1  = render.RGB{R: 112, G: 116, B: 132} // farthest, palest
	ridge2  = render.RGB{R: 70, G: 74, B: 92}
	ridge3  = render.RGB{R: 42, G: 45, B: 58}
	mistCol = render.RGB{R: 168, G: 164, B: 158}

	woodTop  = render.RGB{R: 104, G: 68, B: 43}
	woodNear = render.RGB{R: 72, G: 45, B: 28}
	woodEdge = render.RGB{R: 40, G: 25, B: 16}
	grainCol = render.RGB{R: 132, G: 90, B: 58}
	shadow   = render.RGB{R: 28, G: 17, B: 11}

	// 朱漆: the columns of a pavilion are lacquered, not left dark.
	lacquer    = render.RGB{R: 126, G: 46, B: 36}
	lacquerLit = render.RGB{R: 178, G: 76, B: 56}
	lacquerDim = render.RGB{R: 62, G: 22, B: 18}

	// 紫砂: a matte red-brown clay, warmer in the light than it looks.
	clay     = render.RGB{R: 118, G: 62, B: 45}
	clayLit  = render.RGB{R: 178, G: 108, B: 76}
	clayDark = render.RGB{R: 48, G: 24, B: 18}

	tea        = render.RGB{R: 112, G: 138, B: 66}
	teaLit     = render.RGB{R: 168, G: 194, B: 104}
	steamCol   = render.RGB{R: 214, G: 218, B: 214}
	drizzleCol = render.RGB{R: 186, G: 196, B: 210}
)

func NewTea() *Tea { return &Tea{rnd: newRNG(0x7EA0C0FFEE)} }

func (s *Tea) resize(w, h int) {
	s.w, s.h = w, h
	fh, fw := float64(h), float64(w)

	// The table is the near quarter. It was half, and half of the frame given
	// to an empty plane reads as ground, not as a table you are sitting at.
	s.tableY = fh * 0.74
	s.potW = math.Max(15, fw*0.16)
	s.potH = s.potW * 0.72 // a 紫砂壶 is wider than it is tall
	s.potX = fw * 0.24
	// The pot stands on the table, so its foot is below the table's far edge
	// and its body well above it. That overlap is the only depth cue there is
	// room for at this size.
	s.rimY = s.tableY + s.potH*0.34 - s.potH
	s.pillar = math.Max(4, fw*0.062)
	s.puffs = s.puffs[:0]

	// 细雨 — many threads, each almost nothing. A few heavy ones read as a
	// downpour; the whole effect here is that you have to look to be sure.
	n := (w * int(s.tableY)) / 55
	if n < 40 {
		n = 40
	}
	s.drops = make([]streak, n)
	for i := range s.drops {
		s.drops[i] = s.spawnStreak(true)
	}
}

func (s *Tea) Draw(c *render.Canvas, t, dt float64) {
	if c.W != s.w || c.H != s.h {
		s.resize(c.W, c.H)
	}
	s.sky(c)
	s.ridges(c)
	s.mist(c, t)
	s.drizzle(c, dt)
	s.table(c)
	s.pot(c, t)
	s.steam(c, dt)
	s.pavilion(c)
	s.vignette(c)
}

func (s *Tea) sky(c *render.Canvas) {
	for y := 0; y < int(s.tableY) && y < c.H; y++ {
		f := float64(y) / s.tableY
		c.Fill(y, duskTop.Blend(duskLow, f*f))
	}
}

// ridge is one skyline.
//
// The waveform is triangular — asin(sin) — because a sine gives rolling hills
// and what is wanted is peaks. Softer octaves on top keep the slopes from
// reading as a sawtooth.
func (s *Tea) ridge(x, base, amp, freq, phase float64) float64 {
	fx := x / float64(s.w) * freq
	tri := math.Asin(math.Sin(fx+phase)) * (2 / math.Pi)
	v := 0.62*tri + 0.26*math.Sin(fx*2.3+phase*1.6) + 0.12*math.Sin(fx*5.1+phase*2.9)
	return base - amp*(0.45+0.55*v)
}

// ridges: three of them, each darker and lower than the last. Aerial
// perspective is the only depth cue available at this size.
func (s *Tea) ridges(c *render.Canvas) {
	hz, fh := s.tableY, float64(s.h)
	for x := 0; x < c.W; x++ {
		fx := float64(x)
		for _, r := range []struct {
			amp, freq, phase float64
			col              render.RGB
		}{
			{fh * 0.46, 3.4, 0.6, ridge1},
			{fh * 0.32, 5.2, 2.4, ridge2},
			{fh * 0.19, 8.1, 5.1, ridge3},
		} {
			for y := int(s.ridge(fx, hz, r.amp, r.freq, r.phase)); y < int(hz); y++ {
				c.Set(x, y, r.col)
			}
		}
	}
}

// mist lies along the foot of the ridges and drifts. It is what makes the
// mountains read as far away rather than as a low wall.
func (s *Tea) mist(c *render.Canvas, t float64) {
	top := s.tableY - float64(s.h)*0.22
	for y := int(top); y < int(s.tableY); y++ {
		if y < 0 {
			continue
		}
		d := (float64(y) - top) / (s.tableY - top)
		for x := 0; x < c.W; x++ {
			fx := float64(x) / float64(s.w)
			band := 0.5 + 0.5*math.Sin(fx*5.5-t*0.06)*math.Sin(fx*2.1+t*0.037)
			c.Add(x, y, mistCol, d*d*0.55*band)
		}
	}
}

// drizzleSlant is how far a thread drifts sideways per unit of fall. Rain that
// falls dead straight reads as static.
const drizzleSlant = 0.22

func (s *Tea) drizzle(c *render.Canvas, dt float64) {
	for i := range s.drops {
		d := &s.drops[i]
		d.y += d.vy * dt
		d.x += d.vy * dt * drizzleSlant
		if d.y > s.tableY || d.x > float64(s.w) {
			*d = s.spawnStreak(false)
			continue
		}
		for k := 0.0; k < d.length; k++ {
			y := d.y - k
			if y < 0 || y > s.tableY {
				continue
			}
			c.Add(int(d.x-k*drizzleSlant), int(y), drizzleCol, d.bright*(1-k/d.length))
		}
	}
}

// table. What makes a plane read as a table rather than as ground is its edge:
// a dark band of thickness where the top stops, with the view continuing above
// it. The grain then says wood, and the shadow under the pot says the pot is
// standing on it rather than floating in front of it.
func (s *Tea) table(c *render.Canvas) {
	depth := float64(c.H) - s.tableY
	if depth <= 0 {
		return
	}
	edge := math.Max(2, depth*0.10)

	for y := int(s.tableY); y < c.H; y++ {
		fy := float64(y) - s.tableY
		if fy < edge {
			c.Fill(y, woodEdge) // the thickness of the top, seen end-on
			continue
		}
		d := (fy - edge) / (depth - edge)
		c.Fill(y, woodTop.Blend(woodNear, d*d))
	}

	// Grain: long shallow arcs. Straight lines read as a ruled page.
	for y := int(s.tableY + edge); y < c.H; y++ {
		fy := float64(y)
		d := (fy - s.tableY) / depth
		for x := 0; x < c.W; x++ {
			fx := float64(x) / float64(s.w)
			v := math.Sin(fy*1.7 + math.Sin(fx*2.4)*2.2)
			if v <= 0.55 {
				continue
			}
			c.Add(x, y, grainCol, (v-0.55)/0.45*0.30*(1-d*0.5))
		}
	}
}

// ellipse fills an axis-aligned ellipse, calling paint with how far in each
// pixel is (1 at the centre, 0 at the edge).
func ellipse(c *render.Canvas, cx, cy, rx, ry float64, paint func(x, y int, inward float64)) {
	for y := int(cy - ry - 1); y <= int(cy+ry+1); y++ {
		for x := int(cx - rx - 1); x <= int(cx+rx+1); x++ {
			dx, dy := (float64(x)-cx)/rx, (float64(y)-cy)/ry
			if r := dx*dx + dy*dy; r <= 1 {
				paint(x, y, 1-math.Sqrt(r))
			}
		}
	}
}

func (s *Tea) pot(c *render.Canvas, t float64) {
	half := s.potW / 2
	footY := s.rimY + s.potH

	// Its shadow first, so everything else sits on top of it.
	ellipse(c, s.potX+half*0.25, footY-1, half*1.9, s.potH*0.16, func(x, y int, in float64) {
		c.Add(x, y, shadow, 0.55*in)
	})

	// The belly: an ovoid, widest a little above halfway. A cylinder with
	// shading reads as a tin however it is lit; the silhouette has to do it.
	belly := func(f float64) float64 {
		v := 1 - math.Pow((f-0.42)/0.70, 2)
		if v <= 0 {
			return 0
		}
		return half * math.Sqrt(v)
	}

	s.spout(c, half, footY)
	s.handle(c, half)

	for y := int(s.rimY); y <= int(footY); y++ {
		f := (float64(y) - s.rimY) / s.potH
		w := belly(f)
		if w <= 0 {
			continue
		}
		for x := int(s.potX - w); x <= int(s.potX+w); x++ {
			off := (float64(x) - s.potX) / w
			// Lit from the upper left, where the sky is open.
			lit := math.Max(0, 1-math.Abs(off+0.45)*1.5)
			col := clay.Blend(clayLit, lit*0.75).Blend(clayDark, f*f*0.55)
			c.Set(x, y, col)
		}
	}

	// Foot ring: one darker row where the clay meets the wood.
	for x := int(s.potX - belly(1)); x <= int(s.potX+belly(1)); x++ {
		c.Set(x, int(footY), clayDark)
	}

	// The mouth, seen from just above, with the tea in it.
	rx, ry := half*0.52, s.potH*0.15
	ellipse(c, s.potX, s.rimY, rx, ry, func(x, y int, in float64) {
		if in < 0.22 {
			c.Set(x, y, clayLit) // the lip
			return
		}
		dx := (float64(x) - s.potX) / rx
		lift := 0.5 + 0.5*math.Sin(t*0.9+dx*1.7+float64(y)*0.4)
		c.Set(x, y, tea.Blend(teaLit, in*0.7*lift))
	})

	// On the table beside the pot, tipped against it. Sitting it on the belly
	// made it read as part of the pot; what separates them is the dark edge
	// drawn under it, not the position.
	s.lid(c, s.potX+half*1.30, footY-s.potW*0.09)
}

// spout, on the left, drawn as a tapering run of dots along a curve. A spout
// is what tells a pot from a jar, and at this size it is three pixels wide at
// the root and one at the tip.
func (s *Tea) spout(c *render.Canvas, half, footY float64) {
	x0, y0 := s.potX-half*0.75, s.rimY+s.potH*0.50
	x1, y1 := s.potX-half*1.85, s.rimY+s.potH*0.02
	for u := 0.0; u <= 1; u += 0.02 {
		// One control point above the line bows it upward.
		bx := x0 + (x1-x0)*u
		by := y0 + (y1-y0)*u - math.Sin(u*math.Pi)*s.potH*0.10
		w := (1-u)*half*0.17 + 0.55
		for dy := -w; dy <= w; dy++ {
			for dx := -w * 0.7; dx <= w*0.7; dx++ {
				col := clay
				if dy < 0 {
					col = clay.Blend(clayLit, 0.5)
				}
				c.Set(int(bx+dx), int(by+dy), col)
			}
		}
	}
}

// handle, on the right: an arc standing clear of the belly, which is the other
// half of reading as a pot.
func (s *Tea) handle(c *render.Canvas, half float64) {
	cx := s.potX + half*1.05
	cy := s.rimY + s.potH*0.48
	rx, ry := half*0.62, s.potH*0.40
	for a := -1.45; a <= 1.45; a += 0.02 {
		x := cx + math.Cos(a)*rx
		y := cy + math.Sin(a)*ry
		for dy := -0.8; dy <= 0.8; dy++ {
			for dx := -0.8; dx <= 0.8; dx++ {
				c.Set(int(x+dx), int(y+dy), clay.Blend(clayLit, 0.35))
			}
		}
	}
}

// lid: off the pot and leaning against it, its foot on the table and its top
// against the belly. Seen from the front it is a disc turned almost edge-on,
// so it is drawn as a tilted ellipse with a thickened lower rim and the knob
// on the face turned toward us.
func (s *Tea) lid(c *render.Canvas, cx, cy float64) {
	// Flat: a disc tipped toward us is much shorter than it is wide, and that
	// ratio is the whole of what says "lid" rather than "ball".
	rx, ry := s.potW*0.30, s.potW*0.17
	const tilt = -0.62
	sin, cos := math.Sin(tilt), math.Cos(tilt)

	// An outline first. The lid and the pot are the same clay, so without one
	// they merge into a single blob and the lid stops existing.
	ellipse(c, cx, cy, rx*1.28, ry*1.55, func(x, y int, _ float64) {
		c.Set(x, y, clayDark)
	})
	for y := int(cy - ry - 2); y <= int(cy+ry+2); y++ {
		for x := int(cx - ry - 2); x <= int(cx+ry+2); x++ {
			ox, oy := float64(x)-cx, float64(y)-cy
			u := (ox*cos + oy*sin) / rx
			v := (-ox*sin + oy*cos) / ry
			r := u*u + v*v
			if r > 1 {
				continue
			}
			switch {
			case r > 0.80 && v > 0:
				c.Set(x, y, clayDark) // the rim's thickness, along the low edge
			case r > 0.80:
				c.Set(x, y, clayLit) // and its catch-light along the high edge
			default:
				c.Set(x, y, clay.Blend(clayLit, math.Max(0, 1-r-u)*0.45))
			}
		}
	}
	// The knob, standing off the face turned toward us: without it the disc is
	// just a coin.
	kx, ky := cx+ry*0.9*sin, cy-ry*0.9*cos
	for y := int(ky - 1); y <= int(ky+1); y++ {
		for x := int(kx - 1); x <= int(kx+1); x++ {
			c.Set(x, y, clayLit)
		}
	}
}

func (s *Tea) steam(c *render.Canvas, dt float64) {
	for len(s.puffs) < 20 {
		s.puffs = append(s.puffs, s.spawnPuff())
	}
	out := s.puffs[:0]
	for _, p := range s.puffs {
		p.age += dt
		p.y -= p.vy * dt
		if p.age >= p.life || p.y < 0 {
			out = append(out, s.spawnPuff())
			continue
		}
		f := p.age / p.life
		// Rising steam widens and thins; it does not fade evenly, it comes
		// apart.
		radius := p.radius * (0.6 + 2.2*f)
		alpha := 0.5 * (1 - f) * (1 - f)
		x := p.x + math.Sin(p.drift+f*3.4)*radius*1.3
		for dy := -radius; dy <= radius; dy++ {
			for dx := -radius; dx <= radius; dx++ {
				d := math.Hypot(dx, dy*1.6) / radius
				if d > 1 {
					continue
				}
				c.Add(int(x+dx), int(p.y+dy), steamCol, alpha*(1-d)*(1-d))
			}
		}
		out = append(out, p)
	}
	s.puffs = out
}

func (s *Tea) spawnStreak(anywhere bool) streak {
	y := -s.rnd.rangeF(0, s.tableY*0.4)
	if anywhere {
		y = s.rnd.rangeF(0, s.tableY)
	}
	return streak{
		x:      s.rnd.rangeF(-s.tableY*drizzleSlant, float64(s.w)),
		y:      y,
		vy:     s.rnd.rangeF(16, 30),
		length: s.rnd.rangeF(2.0, 4.5),
		bright: s.rnd.rangeF(0.05, 0.16),
	}
}

func (s *Tea) spawnPuff() puff {
	return puff{
		x:      s.potX + s.rnd.rangeF(-s.potW*0.16, s.potW*0.16),
		y:      s.rimY - s.rnd.rangeF(0, 2),
		vy:     s.rnd.rangeF(4.0, 8.5),
		drift:  s.rnd.rangeF(0, math.Pi*2),
		life:   s.rnd.rangeF(2.2, 4.0),
		radius: s.rnd.rangeF(1.4, 2.6),
	}
}

// pavilion is drawn last because it is the nearest thing in the frame: two
// lacquered pillars and the beam they carry.
//
// They were near-black bands to begin with, which reads as a letterbox crop
// rather than as architecture. Vermilion, a highlight down the round of each
// shaft, a plinth at the foot and a bracket at the head are what make them
// columns.
func (s *Tea) pavilion(c *render.Canvas) {
	beam := math.Max(2, float64(c.H)*0.05)
	for _, left := range []bool{true, false} {
		for i := 0; i < int(s.pillar); i++ {
			x := i
			f := float64(i) / s.pillar // 0 at the frame's edge
			if !left {
				x = c.W - 1 - i
			}
			// A band of light down the shaft, nearer the inner face.
			col := lacquer.
				Blend(lacquerLit, math.Max(0, 1-math.Abs(f-0.7)*3.2)*0.9).
				Blend(lacquerDim, math.Max(0, f-0.85)*4)
			for y := int(beam); y < c.H; y++ {
				c.Set(x, y, col)
			}
		}
		// Plinth: a wider, darker block where the shaft meets the floor.
		base := int(s.pillar * 1.4)
		for i := 0; i < base; i++ {
			x := i
			if !left {
				x = c.W - 1 - i
			}
			for y := c.H - int(math.Max(2, float64(c.H)*0.035)); y < c.H; y++ {
				c.Set(x, y, lacquerDim)
			}
		}
		// Bracket: the head of the column, spreading to meet the beam.
		for i := 0; i < int(s.pillar*1.7); i++ {
			x := i
			if !left {
				x = c.W - 1 - i
			}
			for y := int(beam); y < int(beam*1.8); y++ {
				c.Set(x, y, lacquerDim)
			}
		}
	}
	// The beam across the top: the roof, seen as the underside of its edge.
	for y := 0; y < int(beam); y++ {
		c.Fill(y, lacquerDim.Blend(render.RGB{}, float64(y)/beam*0.5))
	}
}

func (s *Tea) vignette(c *render.Canvas) {
	cx, cy := float64(c.W)/2, float64(c.H)/2
	maxD := math.Hypot(cx, cy*0.75)
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			d := math.Hypot(float64(x)-cx, (float64(y)-cy)*0.75) / maxD
			if d < 0.62 {
				continue
			}
			f := (d - 0.62) / 0.38
			i := y*c.W + x
			c.Px[i] = c.Px[i].Scale(1 - f*f*0.55)
		}
	}
}
