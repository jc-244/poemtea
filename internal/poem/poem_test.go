package poem

import (
	"testing"

	"github.com/jc-244/poemtea/internal/render"
)

// Wider than this and a line cannot be centred on an eighty-column terminal.
// A 七言 couplet set two 句 to a line is thirty-two, so the ceiling is nowhere
// near — it is here to catch the day somebody adds a line that is.
const maxWidth = 76

func label(p Poem) string {
	if p.Source != "" {
		return p.Source
	}
	return p.Author
}

func TestCorpusFitsANarrowTerminal(t *testing.T) {
	for _, p := range Corpus {
		for _, ln := range p.Lines {
			if w := render.StringWidth(ln); w > maxWidth {
				t.Errorf("%s: %d columns wide, over %d — %s", label(p), w, maxWidth, ln)
			}
		}
	}
}

func TestCorpusKeepsToThreeLines(t *testing.T) {
	for _, p := range Corpus {
		if len(p.Lines) > 3 {
			t.Errorf("%s has %d lines; three is the ceiling", label(p), len(p.Lines))
		}
	}
}
