package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/jc-244/poemtea/internal/state"
)

// runMark is the target of the hooks. Claude Code passes a JSON object on
// stdin containing session_id, which is exactly the key we want: it is stable
// for the life of a session and distinct across concurrent ones.
//
// Anything that cannot supply a session id — a shell one-liner from some other
// tool — falls back to an environment variable and then to the parent process,
// so wiring up a new agent stays a one-line job.
func runMark(busy bool) error {
	// The owner comes from the wrapper, via the agent's environment. With no
	// wrapper there is nothing to draw on, and the record would belong to
	// nobody — so write it under an empty owner and let it be ignored.
	return state.Set(os.Getenv(state.OwnerEnv), identify(), busy)
}

type hookInput struct {
	SessionID string `json:"session_id"`
}

func identify() string {
	if fi, err := os.Stdin.Stat(); err == nil && fi.Mode()&os.ModeCharDevice == 0 {
		if b, err := io.ReadAll(io.LimitReader(os.Stdin, 1<<20)); err == nil && len(b) > 0 {
			var in hookInput
			if json.Unmarshal(b, &in) == nil && in.SessionID != "" {
				return in.SessionID
			}
		}
	}
	if v := os.Getenv("POEMTEA_SESSION"); v != "" {
		return v
	}
	return fmt.Sprintf("pid%d", os.Getppid())
}

// The rule these hooks encode, in one sentence:
//
//	the agent working on its own is busy; the agent waiting on you is not.
//
// Every event below follows from that. In particular PermissionRequest and
// Notification must clear the flag, or the picture will cover a prompt that is
// waiting for an answer — and the whole premise of this program is that it
// never stands between you and the tool.
//
// PostToolUse is the one that looks redundant and is not. Without it, a turn
// interrupted by a single permission prompt stays marked idle for the rest of
// its life: PermissionRequest clears the flag and nothing sets it again until
// the next prompt is submitted. Any long turn that asked for approval would
// silently never show the picture at all.
const hooksSnippet = `{
  "hooks": {
    "UserPromptSubmit": [
      { "hooks": [ { "type": "command", "command": "poemtea busy" } ] }
    ],
    "PostToolUse": [
      { "hooks": [ { "type": "command", "command": "poemtea busy" } ] }
    ],
    "PermissionRequest": [
      { "hooks": [ { "type": "command", "command": "poemtea idle" } ] }
    ],
    "Notification": [
      { "hooks": [ { "type": "command", "command": "poemtea idle" } ] }
    ],
    "Stop": [
      { "hooks": [ { "type": "command", "command": "poemtea idle" } ] }
    ],
    "SessionEnd": [
      { "hooks": [ { "type": "command", "command": "poemtea idle" } ] }
    ]
  }
}`

// runInstallHooks prints the configuration rather than writing it. Editing a
// user's settings.json behind their back is the sort of thing this project is
// supposed to be the opposite of.
func runInstallHooks() error {
	home, _ := os.UserHomeDir()
	fmt.Printf(`Add this to %s/.claude/settings.json, merging with any existing "hooks":

%s

Then run your agent through the wrapper:

    poemtea run -- claude

  busy  UserPromptSubmit   you handed the work over
        PostToolUse        it is (back) working on its own
  idle  PermissionRequest  it is waiting on YOU -- give the screen back at once
        Notification       likewise, for anything wanting your attention
        Stop / SessionEnd  the turn, or the session, is over

The -after threshold then decides which turns were long enough to be worth
interrupting for.

The hooks need no arguments: "poemtea run" exports POEMTEA_OWNER into the
agent's environment, the hooks inherit it, and each wrapper reacts only to the
agent it launched. Any other agent can join in by writing the same state; see
internal/state for the format.
`, home, hooksSnippet)
	return nil
}
