package poem

import (
	"strings"
	"testing"

	"github.com/jc-244/poemtea/internal/render"
)

// Wider than this and a line cannot be centred on an eighty-column terminal.
// A 七言 couplet set two 句 to a line is thirty-two columns, so the ceiling is
// nowhere near — it is here to catch the day something is generated that is.
const maxWidth = 76

func TestCorpusFitsANarrowTerminal(t *testing.T) {
	for _, p := range Corpus {
		for _, ln := range p.Lines {
			if w := render.StringWidth(ln); w > maxWidth {
				t.Errorf("%s: %d columns wide, over %d — %s", p.Source, w, maxWidth, ln)
			}
		}
	}
}

// Every poem is a 绝句 set two 句 to a line, so every poem is two lines of two
// 句. A generator that starts emitting anything else — half a 律诗, a stray
// couplet — shows up here rather than on somebody's screen.
func TestCorpusIsQuatrainsTwoJuToALine(t *testing.T) {
	for _, p := range Corpus {
		if len(p.Lines) != 2 {
			t.Errorf("%s has %d lines, want 2", p.Source, len(p.Lines))
			continue
		}
		width := 0
		for _, ln := range p.Lines {
			ju := strings.FieldsFunc(strings.TrimSuffix(ln, "。"), func(r rune) bool {
				return r == '，'
			})
			if len(ju) != 2 {
				t.Errorf("%s: line %q is not two 句", p.Source, ln)
				continue
			}
			for _, j := range ju {
				n := len([]rune(j))
				if n != 5 && n != 7 {
					t.Errorf("%s: 句 %q is %d characters, want 5 or 7", p.Source, j, n)
				}
				if width == 0 {
					width = n
				} else if n != width {
					t.Errorf("%s: 句 lengths differ (%d and %d)", p.Source, width, n)
				}
			}
		}
	}
}

func TestCorpusIsAllDuMu(t *testing.T) {
	for _, p := range Corpus {
		if p.Author != "杜牧" {
			t.Errorf("%s: author %q", p.Source, p.Author)
		}
		if p.Source == "" {
			t.Errorf("a poem has no title: %v", p.Lines)
		}
	}
}
