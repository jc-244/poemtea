package wrap

import "testing"

func TestOnlyRealKeysCount(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want bool
		name string
	}{
		{"a", true, "a letter"},
		{"\x1b[A", true, "up arrow"},
		{"\x1bOP", true, "F1"},
		{"\x1b", true, "the escape key alone"},
		{"\x1b[15~", true, "F5"},
		{"\x1b[200~pasted\x1b[201~", true, "a paste"},
		{"\x1ba", true, "alt+a"},
		{"\r", true, "return"},

		{"\x1b[<0;10;5M", true, "left button down"},
		{"\x1b[<2;10;5M", true, "right button down"},
		{"\x1b[<64;10;5M", true, "wheel up"},
		{"\x1b[<65;10;5M", true, "wheel down"},
		{"\x1b[M`  ", true, "wheel up, original encoding"},
		{"\x1b[M   ", true, "button down, original encoding"},

		{"\x1b[<35;80;12M", false, "mouse moving"},
		{"\x1b[<0;10;5m", false, "mouse release"},
		{"\x1b[<32;10;5M", false, "dragging with a button held"},
		{"\x1b[<66;10;5M", false, "wheel left"},
		{"\x1b[<67;10;5M", false, "wheel right"},
		{"\x1b[M@  ", false, "moving, original encoding"},
		{"\x1b[M#  ", false, "button released, original encoding"},
		{"\x1b[I", false, "window focused"},
		{"\x1b[O", false, "window unfocused"},
		{"\x1b[12;40R", false, "cursor position report"},
		{"\x1b[?62;c", false, "device attributes reply"},
		{"\x1b]11;rgb:1c/1c/1c\x07", false, "background colour reply"},
		{"\x1b[<35;1;1M\x1b[<35;2;1M\x1b[<35;3;1M", false, "a mouse crossing the window"},
	} {
		var k keyDetect
		if got := k.sawKey([]byte(tc.in)); got != tc.want {
			t.Errorf("%s (%q): got %v, want %v", tc.name, tc.in, got, tc.want)
		}
	}
}

func TestMouseFollowedByTypingStillCounts(t *testing.T) {
	var k keyDetect
	if !k.sawKey([]byte("\x1b[<35;5;5Mx")) {
		t.Error("a key press after a mouse report was missed")
	}
}

func TestSequenceSplitAcrossReads(t *testing.T) {
	var k keyDetect
	if k.sawKey([]byte("\x1b[<35;80")) {
		t.Error("half a mouse report was taken for a key press")
	}
	if k.sawKey([]byte(";12M")) {
		t.Error("the rest of the mouse report was taken for a key press")
	}
}
