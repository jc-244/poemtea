package render

import "github.com/mattn/go-runewidth"

// How many cells a character takes.
//
// The layout of a poem must depend on the poem and nothing else. The library's
// package-level default is derived from the locale — it will quietly answer
// differently on a machine whose LANG is Japanese — so we carry our own
// condition and set every field of it ourselves. NewCondition copies the
// locale-derived values, which is exactly what the two lines below undo.
//
// EastAsianWidth false means an *ambiguous* character is one cell: the em dash
// in an attribution, the diaeresis in Brontë. The English corpus has assumed
// that since the first line was drawn, and it is the assumption a terminal not
// running in a CJK locale makes too.
var widths = narrowAmbiguous()

func narrowAmbiguous() *runewidth.Condition {
	c := runewidth.NewCondition()
	c.EastAsianWidth = false
	c.StrictEmojiNeutral = true
	return c
}

// RuneWidth is 2 for a double-width character and 1 for the rest.
func RuneWidth(r rune) int { return widths.RuneWidth(r) }

// StringWidth is how many cells a string occupies once drawn.
func StringWidth(s string) int { return widths.StringWidth(s) }
