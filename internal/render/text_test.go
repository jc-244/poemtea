package render

import "testing"

// An ambiguous-width character is one cell. The em dash that opens every
// attribution is one of them, and the library will answer differently if it is
// ever allowed to consult the locale — so the answer is pinned here, where a
// change shows up as a failing test rather than as a line off centre.
func TestAmbiguousCharactersAreOneCell(t *testing.T) {
	for _, tc := range []struct {
		s    string
		want int
	}{
		{"— Ezra Pound", 12},
		{"Emily Brontë", 12},
		{"Whirl up, sea—", 14},
	} {
		if got := StringWidth(tc.s); got != tc.want {
			t.Errorf("StringWidth(%q) = %d, want %d", tc.s, got, tc.want)
		}
	}
}

func TestChineseIsTwoCellsPerCharacter(t *testing.T) {
	if got := StringWidth("山行"); got != 4 {
		t.Errorf("StringWidth(山行) = %d, want 4", got)
	}
	// A 七言 line as this corpus sets it: seven characters, a comma, seven
	// more, a full stop — sixteen characters, thirty-two columns.
	if got := StringWidth("远上寒山石径斜，白云生处有人家。"); got != 32 {
		t.Errorf("width of a 七言 line = %d, want 32", got)
	}
}

func TestPutLineCentresByDisplayWidth(t *testing.T) {
	g := NewGrid(40, 1)
	putLine(g, "山行", 0, Ink, 1)

	// Four columns in forty start at 18. Counting characters instead would
	// start at 19 and leave the line a column off centre.
	for _, want := range []struct {
		x  int
		ch rune
	}{{18, '山'}, {19, contd}, {20, '行'}, {21, contd}} {
		if got := g.At(want.x, 0).Ch; got != want.ch {
			t.Errorf("cell %d holds %q, want %q", want.x, got, want.ch)
		}
	}
}

func TestPutLineNeverSplitsAWideGlyph(t *testing.T) {
	g := NewGrid(3, 1)
	putLine(g, "山山", 0, Ink, 1)

	if got := g.At(0, 0).Ch; got != '山' {
		t.Errorf("cell 0 holds %q, want 山", got)
	}
	if got := g.At(1, 0).Ch; got != contd {
		t.Errorf("cell 1 holds %q, want the continuation mark", got)
	}
	// The second 山 needs two columns and only one is left, so it is dropped
	// whole rather than written as a half.
	if got := g.At(2, 0).Ch; got != ' ' {
		t.Errorf("cell 2 holds %q, want it untouched", got)
	}
}
