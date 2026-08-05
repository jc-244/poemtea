// Package poem holds the corpus: 杜牧's 绝句, and nothing else.
//
// The poems live in corpus.go, which is generated — run corpus.py to rebuild
// it from 全唐詩 on wikisource. Do not add a poem by hand; add it to the source
// the generator reads, or teach the generator to find it.
package poem

type Poem struct {
	Lines  []string
	Author string
	Source string // the poem's title
}

// Deck hands out poems in a shuffled order and only reshuffles once the whole
// corpus has been seen. Sampling at random would let the same line come up
// twice in five minutes, which reads as broken rather than as repetition.
type Deck struct {
	order []int
	i     int
	seed  uint64
}

func NewDeck(seed uint64) *Deck {
	d := &Deck{seed: seed}
	d.reshuffle()
	return d
}

func (d *Deck) reshuffle() {
	d.order = make([]int, len(Corpus))
	for i := range d.order {
		d.order[i] = i
	}
	for i := len(d.order) - 1; i > 0; i-- {
		d.seed ^= d.seed << 13
		d.seed ^= d.seed >> 7
		d.seed ^= d.seed << 17
		j := int(d.seed % uint64(i+1))
		d.order[i], d.order[j] = d.order[j], d.order[i]
	}
	d.i = 0
}

func (d *Deck) Next() Poem {
	if d.i >= len(d.order) {
		d.reshuffle()
	}
	p := Corpus[d.order[d.i]]
	d.i++
	return p
}
