"""Draw the wordmark. Writes docs/logo-light.svg and docs/logo-dark.svg.

    python tools/logo.py

It is drawn from rectangles rather than set in a typeface, so it cannot render
differently on a machine that lacks the font. That is the only thing it has in
common with a terminal's block characters — the shapes are a logotype, not a
5x7 bitmap font, and the difference is in three places:

  - the blocks touch. A gap between them reads as an LED matrix.
  - a stroke is one unit and a counter is two, so the letters are chunky. One
    and three is the proportion of a screen font and looks like one.
  - the counter is not left empty. Its lower part is filled in a lighter tone,
    which is what stops the mark reading as a grid of squares — it is the whole
    of the effect.

An earlier version raised the M a row for an ascender, which moved the whole
letter instead of making it taller and lifted it off the baseline. Real
ascenders need glyphs of different heights; a mark that sits straight beats one
with a broken one.

Two files, because the light one inverts for a dark page: on a dark background
the letter is pale and the counter is dark. A single mark recoloured, rather
than inverted, always ends up dim on one of the two.
"""

import os

HERE = os.path.dirname(os.path.abspath(__file__))
DOCS = os.path.join(os.path.dirname(HERE), "docs")

UNIT = 12   # px per cell
TOP = 1     # rows of headroom, so an ascender has somewhere to go

# '#' the letter, 'o' the lit part of a counter, '.' nothing.
GLYPHS = {
    "P": ["####",
          "#oo#",
          "####",
          "#...",
          "#..."],
    "O": ["####",
          "#..#",
          "#oo#",
          "#oo#",
          "####"],
    "E": ["####",
          "#...",
          "###.",
          "#oo.",
          "####"],
    "M": ["#...#",
          "##.##",
          "#.#.#",
          "#ooo#",
          "#...#"],
    # Four wide, this reads as an 8. The fifth column is the tail, and the tail
    # is the only thing that makes an ampersand one.
    "&": [".##..",
          "#..#.",
          ".##..",
          "#.#.#",
          ".##.#"],
    "T": ["#####",
          "..#..",
          "..#..",
          "..#..",
          "..#.."],
    "A": ["####",
          "#oo#",
          "####",
          "#..#",
          "#..#"],
    " ": ["..", "..", "..", "..", ".."],
}

WORD = "POEM & TEA"
# Which run of letters takes which of the two tones.
WOOD, AMP, TEA = range(3)
TONE = [WOOD]*4 + [AMP]*3 + [TEA]*3     # POEM | " & " | TEA

# 木 for POEM and 茶 for TEA, both taken from the scene: the table and what is
# in the pot. Light page first, then the inversion.
THEMES = {
    "light": {"body": ["#8A5426", "#8C7C6E", "#5F7A2E"],
              "lit":  ["#E0BE97", "#D6CFC7", "#C3D9A0"]},
    "dark":  {"body": ["#D69C63", "#B7B1B1", "#AECC77"],
              "lit":  ["#5A3A1C", "#4B4646", "#3C5320"]},
}


def lay_out():
    """(cell x, cell y, kind, tone) for every filled cell, and the size."""
    cells, x = [], 0
    for i, ch in enumerate(WORD):
        rows = GLYPHS[ch]
        drop = 0
        for y, row in enumerate(rows):
            for dx, c in enumerate(row):
                if c != ".":
                    cells.append((x + dx, TOP + y + drop, c, TONE[i]))
        x += len(rows[0]) + (1 if ch != " " else 0)
    w = max(c[0] for c in cells) + 1
    h = max(c[1] for c in cells) + 1
    return cells, w, h


def write(theme, path):
    cells, w, h = lay_out()
    pad = UNIT
    W, H = w*UNIT + pad*2, h*UNIT + pad*2
    out = ['<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" '
           'viewBox="0 0 %d %d" role="img" aria-label="POEM &amp; TEA">' % (W, H, W, H),
           '<title>POEM &amp; TEA</title>']
    # One group per fill, so recolouring is six lines rather than three hundred.
    for kind in ("lit", "#"):
        for tone in range(3):
            colour = THEMES[theme]["lit" if kind == "lit" else "body"][tone]
            rects = ['  <rect x="%d" y="%d" width="%d" height="%d"/>'
                     % (pad + cx*UNIT, pad + cy*UNIT, UNIT, UNIT)
                     for cx, cy, k, t in cells
                     if t == tone and (k == "o") == (kind == "lit")]
            if rects:
                out.append('<g fill="%s">' % colour)
                out += rects
                out.append('</g>')
    out.append('</svg>')
    with open(path, "w") as f:
        f.write("\n".join(out) + "\n")
    return W, H


for theme in THEMES:
    size = write(theme, os.path.join(DOCS, "logo-%s.svg" % theme))
print("wrote docs/logo-light.svg and docs/logo-dark.svg, %dx%d" % size)
