# Design notes

Why poemtea is built the way it is. None of this is needed to use it — see the
[README](../README.md) for that. It is here because most of these decisions were
paid for with a bug on somebody's real screen, and the reasoning is worth more
than the code it produced.

## The layer

```
poemtea run -- claude
  ├── pty ─────────────────── claude runs inside; it believes it owns a terminal
  ├── the alternate buffer ── taken from claude, used as the layer
  ├── replay buffer ───────── what claude drew while it was covered
  └── ~/.poemtea/sessions/*.json ── the busy/idle protocol
```

**The picture is a layer, and the terminal is what remembers what is under it.**
A terminal is one grid of character cells: no z-order, no transparency, no way
to read back what a cell holds, no undo. The only layer primitive the protocol
has is the alternate screen buffer — a whole second grid you switch to and back
from. Used as intended, showing and hiding the picture is two escape sequences,
and the host's screen cannot be damaged because nothing is ever written to it.

The host's own request for that buffer is withheld at startup, so the buffer
stays ours and the host lives on the primary one.

**What the host drew while covered is replayed, not reconstructed.** Its
withheld bytes are buffered and written back verbatim when the layer comes down.
Nothing is interpreted, so nothing can be lost in translation.

**Muting is a filter, not a switch.** The host talks to the terminal as well as
drawing on it — Device Attributes queries, mouse reporting, focus events,
bracketed paste — and some of that blocks waiting for a reply. Hold the drawing;
forward the conversation. See `internal/wrap/filter.go`.

## Where the dissolve lives

Taking the screen is instantaneous and total — the alternate buffer is
all-or-nothing. The dissolve is not a transition between the host and the
picture at all: it happens **inside the picture**, which assembles out of a flat
field made of pixels we computed ourselves (`internal/scene/reveal.go`, forty
lines).

An earlier version dissolved the host's *text* into the picture one cell at a
time. That effect needs to know what every cell of the host's screen holds, and
a terminal will not tell you, so it was reconstructed from an emulated model of
the screen.

Every attribute the model failed to carry was silently dropped, and the drop
landed on the user's real screen: unset colours came back as black and
permanently replaced the terminal's theme; palette entries were flattened to
RGB, so every colour shifted the moment the picture arrived and again when it
left; double-width characters shifted every following column, turning a line of
Chinese into `完 整 的`; a faint placeholder in the input box came back looking
like text the user had typed and could not edit.

Those were four faces of one mistake — handing back a value we computed for
something the terminal owns — and there were more waiting: hyperlinks,
underline colours, scroll regions. All of it existed to buy two seconds of
animation that could be had, unbreakably, by applying the same effect to our own
canvas instead of to somebody else's screen. Moving it deleted the entire class
of bug along with an emulator, six dependencies and four hundred lines.

The rule that survives, and that is worth keeping if this is ever extended:
**never hand back a concrete value for something the terminal owns.**

## Why `PostToolUse` marks busy

It looks redundant beside `UserPromptSubmit` and is not. Without it, a turn
interrupted by a single permission prompt stays marked idle for the rest of its
life: `PermissionRequest` clears the flag and nothing sets it again until the
next prompt is submitted. Any long turn that asked for approval would silently
never show the picture at all.

## Sessions are scoped to the wrapper

`poemtea run` mints an owner id and exports it to the agent as `POEMTEA_OWNER`.
The agent's hooks inherit it and stamp it on every record they write, and each
wrapper answers only for records carrying its own id.

That one field replaces two bad ideas. Records used to name the hook's parent
process and treat a dead parent as a dead session — but a hook is a fleeting
thing run from a shell that exits milliseconds later, so whether the picture
appeared came down to whether a poll happened to land inside that window. It
worked sometimes, unpredictably, and most often during bursts of tool calls,
because each one rewrote the file and minted a fresh live parent. And "is *any*
agent busy" meant an agent in another window could take this terminal's screen.

To drive a session by hand, name the owner yourself:

```bash
POEMTEA_OWNER=test poemtea run -- claude   # terminal A
POEMTEA_OWNER=test poemtea busy            # terminal B — the picture arrives
POEMTEA_OWNER=test poemtea idle            # and leaves
```

## The two ways in do not share a way out

A picture raised because the agent is working comes down when the turn ends. A
picture raised because nothing has moved for half a minute comes down when you
touch the keyboard, and at no other time.

They shared the exit rule once, and it flickered. The screensaver can only fire
on a machine whose agent stopped working long before, so "idle for longer than
`-linger`" was already satisfied at the instant that picture appeared: the layer
went up and came down again every three frames. See `turnIsOver` in `run.go`.

## Telling you apart from your mouse

The host turns on mouse reporting and focus events, so the terminal sends a
stream of escape sequences whenever the pointer crosses the window or you switch
applications — on the same channel as your typing. The picture must not flinch
away from any of that.

But a click is you, and so is scrolling. The distinction is not key versus
mouse; it is deliberate versus incidental, and the button code says which:

| Bit | Meaning | Counts? |
|---|---|---|
| 32 | the pointer moved, with or without a button held | no — a drag is motion too |
| 64 | the wheel turned; the low two bits carry direction | up and down only |
| — | a button went down | yes |
| — | a button was released | no — the far end of a press already counted |

Both encodings are handled: SGR, where the final byte says press or release, and
the original one, where three bits of the button say it instead. See
`internal/wrap/keys.go`.

## The corpus

杜牧 (803–852), every 绝句 in 全唐詩 卷520–527 — 206 of them. Nobody selected
them. The definition is arithmetic:

> a 绝句 is four 句 of uniform length, five or seven characters each.

`internal/poem/corpus.go` is generated; `corpus.py` beside it rebuilds the file
from wikisource, where 全唐詩 is public domain. Add a poem by teaching the
generator to find it, not by typing it in.

A quatrain is set two 句 to a line, the way it is printed, so a poem is two
lines of thirty-two columns. It has not lost half its lines.

The readings are 全唐詩's own and are not always the familiar ones — 红烛 for
银烛, 折戟沈沙 for 沉沙, 十三馀 for 十三余, 尊前 for 樽前. Where the source
marks a variant inline, the base reading is kept and the note dropped.

The generator is mostly defence against its source. Those volumes were
transcribed by different hands: three page layouts, some poems punctuated and
some not, footnote markers mid-line, 校勘 notes set inline inside the verse.
Three versions of it were wrong and all three were quiet about it:

- splitting an N首 section into fours on the strength of its title turned every
  律诗 into two quatrains Du Mu never wrote — 《长安杂题长句六首》 alone
  produced twelve;
- requiring two 句 to a line dropped every unpunctuated poem, 《山行》 among
  them;
- turning every HTML tag into a line break cut a 句 into pieces wherever a
  variant was marked, so 《遣怀》 and 《赤壁》 could not be found even by
  searching for them.

What caught all three is the list at the foot of the script: named poems that
must come out, 律诗 lines that must not. It runs before anything is written.

## Drawing at this size

The canvas is one pixel per half character cell, so a 138×35 terminal is a
138×70 image — roughly a Game Boy sprite scene. At that size a shape is read
from its silhouette and almost nothing else, which is the lesson behind most of
`internal/scene/tea.go`:

- a cylinder with shading reads as a tin however it is lit, so the pot needed an
  ovoid belly, a spout and a handle before it was a pot;
- two near-black bands at the frame's edges read as a letterbox crop, so the
  pillars needed vermilion lacquer, a plinth, a bracket and a beam before they
  were columns;
- half the frame given to a flat plane reads as ground, so the table had to lose
  a third of its height and gain the dark band of its own thickness, grain, and
  a shadow under the pot;
- the lid, in the pot's own clay and touching it, was part of the pot until a
  dark edge was drawn under it.

Text is composited at cell level, never into the pixel buffer: at half-cell
resolution a letter is illegible, so the image and the words live at different
resolutions in the same grid.
