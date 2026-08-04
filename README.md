# poemtea

Your agent is running, and there is a poem on the screen.

`poemtea` wraps a coding agent's CLI. When the agent has been working for a
while, the terminal dims and a slow procedural picture assembles out of it, with
a short poem underneath. When the agent answers — or the moment you touch the
keyboard — it comes apart again and your screen is exactly as you left it.

## Build

Go 1.24+. Two dependencies, both small: `creack/pty` and `golang.org/x/term`.

```bash
go build -o ~/.local/bin/poemtea .
```

## Look at it first

```bash
poemtea demo
```

Full-screen scene, no agent involved. `n` for the next poem, `q` to quit. This
is the workbench for the art.

## Wire it to an agent

```bash
poemtea install-hooks     # prints the config; it does not write anything
```

Merge the printed block into `~/.claude/settings.json`, then:

```bash
poemtea run -- claude
```

### What the hooks mean

The whole rule is: **the agent working on its own is busy; the agent waiting on
you is not.**

| Event | Marks | Why |
|---|---|---|
| `UserPromptSubmit` | busy | you handed the work over |
| `PostToolUse` | busy | it is (back) working on its own |
| `PermissionRequest` | **idle** | it is waiting on *you* — give the screen back at once |
| `Notification` | **idle** | likewise, anything wanting your attention |
| `Stop` | idle | the turn is over |
| `SessionEnd` | idle | the session is over |

`PostToolUse` looks redundant and is not. Without it, one permission prompt
leaves the turn marked idle for the rest of its life — `PermissionRequest`
clears the flag and nothing sets it again until the next prompt is submitted —
so any long turn that asked for approval would silently never show the picture.

## Using it

The trigger is the whole turn, not "thinking" specifically:

```
you press enter
  │  UserPromptSubmit -> busy
  ▼
  ├─ 0s ─────── 8s ────────────────────────────┐
  │              └─ still busy: picture assembles │
  ▼                                             ▼
Stop -> idle ──> quiet for 1.5s ──> it comes apart
```

The think/tool/think oscillation inside a turn is deliberately invisible: one
turn is one continuous busy period, so one turn is at most one interruption.

**Any key press sends the picture away** — a real press, not the mouse drifting
across the window and not you switching to another application. The host turns
on mouse and focus reporting, so those arrive on the same channel as typing and
have to be told apart (`internal/wrap/keys.go`). The key still reaches the agent the instant
you type it, so you can start writing your next message while the picture
clears. This is also the escape hatch: if the busy flag ever sticks (a
hook that never fired, an agent that crashed), one key gets your terminal back.

To quit, quit the agent. `poemtea` exits with it and restores the terminal.

| Flag | Default | |
|---|---|---|
| `-after` | 8 | seconds of agent work before the picture appears |
| `-linger` | 1.5 | seconds of quiet before it goes away |
| `-idle` | 30 | seconds of nothing at all before it appears anyway; `0` turns this off |

There are two ways in. The first is the agent working, which is the point of the
program. The second is the ordinary screensaver one: nothing has moved for half
a minute — no key pressed, and nothing drawn by the agent either. Requiring both
to be still means it can never cover an answer while it is still being written,
only one you have already stopped reading.

## How it works

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

### Where the dissolve lives

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

### Sessions are scoped to the wrapper

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

## Adding another agent

Nothing here is specific to Claude Code. `~/.poemtea/sessions/` is a protocol,
not an integration: one small JSON file per session, `{owner, busy, updated}`,
written atomically. Anything that can run a command when it starts
and stops working can join in with a three-line hook, and the renderer does not
change.

## When something looks wrong

```bash
POEMTEA_DEBUG=/tmp/poemtea.log poemtea run -- claude
```

A healthy run logs four lines — `start`, `layer up`, `layer leaving: <why>`,
`unmute`. It logs to a file and never to the screen, because a warning printed
into the middle of someone's terminal is worse than the bug it reports.

If a terminal is ever left in a strange state, `reset` fixes it.

### tmux is another terminal in the way

Two symptoms that look like bugs in this program are one misconfigured tmux: the
colour drains out of the picture, and frames arrive in pieces. tmux decides what
the terminal outside it can do from that terminal's terminfo entry, and
`xterm-256color` claims neither 24-bit colour nor synchronized output — so every
RGB value is downsampled to the 256 palette and the `?2026` markers each frame is
wrapped in are swallowed. `COLORTERM=truecolor` in the environment does not
persuade it.

```bash
# ~/.tmux.conf — name whatever TERM the outer terminal actually reports
set -ag terminal-features ",xterm-256color:RGB:sync"
```

That is read when a client attaches and never again, so a client that is already
attached keeps the capabilities it was born with. Reattach, and check before
concluding anything — testing in the old client will tell you the fix did not
work:

```bash
tmux display -p '#{client_termfeatures}'   # RGB and sync must both appear
```

## Known limitations

- **Truecolor is assumed.** The scene emits 24-bit colour; a terminal that
  speaks only 256 (macOS Terminal.app, for one) snaps every value to its nearest
  palette entry and the gradients band. The fix is to quantise and dither
  ourselves rather than let a rounding rule make the artistic decision.
- **Bandwidth.** Roughly 0.35 MB/s at 120x35 and 0.75 MB/s at 200x50, since a
  rain scene changes most of the frame. Fine locally, poor over SSH.
- **It does not know whether you are at the keyboard.** A turn longer than
  `-after` takes the screen whether you were staring at it thinking or had
  walked away.
- **One scene** (rain) and **English only**. The poem is drawn by us, so CJK
  would need wide-character handling in the text layer.
- **Quitting the agent leaves its last frame on screen** rather than restoring
  the shell view you had before, because we hold the alternate buffer it would
  otherwise have used.

## The corpus

`internal/poem/poem.go` carries two rules in its header, and they matter more
than the code around them: everything must be unambiguously public domain
(**including the translation**, which is the one people get wrong), and every
line must be an image rather than an argument. An image can be abandoned
mid-read and lose nothing. An aphorism asks to be finished — and anything that
asks to be finished turns waiting back into a task.
