package render

import (
	"strings"
	"testing"
)

func TestRenderWritesAWideGlyphOnceAndSkipsItsSecondColumn(t *testing.T) {
	g := NewGrid(4, 1)
	g.Set(0, 0, Cell{Ch: '雨'})
	g.Set(1, 0, Cell{Ch: contd})
	g.Set(2, 0, Cell{Ch: 'a'})

	out := g.Render()

	if n := strings.Count(out, "雨"); n != 1 {
		t.Errorf("wide glyph written %d times, want 1", n)
	}
	if strings.Contains(out, "\x1b[1;2H") {
		t.Error("the second column of the wide glyph was addressed; the terminal owns it")
	}
	// The glyph left the cursor two columns along, so the character after it
	// needs no reposition. A move here means the width was counted as one.
	if strings.Contains(out, "\x1b[1;3H") {
		t.Error("cursor was repositioned after a wide glyph; its width was counted as one column")
	}
}

// The regression this file exists for. A cell covered by a wide glyph sends
// nothing, so the diff has to be told about it anyway — otherwise the frame
// that puts a narrow character back finds the cell unchanged, skips it, and
// leaves the right half of the old glyph on screen.
func TestRenderRepaintsTheColumnAWideGlyphUsedToCover(t *testing.T) {
	g := NewGrid(4, 1)

	g.Set(0, 0, Cell{Ch: 'a'})
	g.Set(1, 0, Cell{Ch: 'b'})
	g.Render()

	g.Set(0, 0, Cell{Ch: '雨'})
	g.Set(1, 0, Cell{Ch: contd})
	g.Render()

	g.Set(0, 0, Cell{Ch: 'a'})
	g.Set(1, 0, Cell{Ch: 'b'})
	out := g.Render()

	if !strings.Contains(out, "b") {
		t.Error("the column the wide glyph covered was left unpainted")
	}
}

func TestRenderStillSkipsUnchangedCells(t *testing.T) {
	g := NewGrid(4, 1)
	g.Set(0, 0, Cell{Ch: '雨'})
	g.Set(1, 0, Cell{Ch: contd})
	g.Render()

	if out := g.Render(); strings.Contains(out, "雨") {
		t.Error("an unchanged wide glyph was redrawn")
	}
}
