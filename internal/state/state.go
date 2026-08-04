// Package state is the contract between whatever is running an agent and
// whatever is drawing the picture.
//
// It is deliberately a directory of small files rather than a socket or an API.
// One file per session means several terminals running several agents cannot
// race each other, and adding support for a new tool — Codex, Aider, anything —
// is a three-line shell hook rather than an integration.
//
// Every record carries an owner: the id of the wrapper the agent was launched
// by, handed down through the environment. Two problems collapse into that one
// field.
//
// The first is liveness. Records used to record the hook's parent process and
// treat a dead parent as a dead session — but a hook is a fleeting thing, run
// from a shell that exits milliseconds later, so its parent is always about to
// die. Whether the screensaver appeared came down to whether a 250ms poll
// happened to land inside that window: it worked sometimes, unpredictably, and
// most reliably during bursts of tool calls, because each one rewrote the file
// and minted a fresh live parent. An owner needs no such guessing — the wrapper
// reading the record is itself the proof that the session is alive.
//
// The second is crosstalk. "Is any agent busy" is true when an unrelated agent
// in another window is busy, so a terminal you were not even looking at could
// take this one's screen. Scoping to the owner makes each wrapper answer only
// for the agent it launched.
package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type Session struct {
	Owner   string `json:"owner"`
	Busy    bool   `json:"busy"`
	Updated int64  `json:"updated"` // unix seconds
}

// staleAfter is the backstop for an agent that dies without ever saying it
// stopped. It is not the primary mechanism — Stop and SessionEnd are, and a
// keystroke overrides everything — so it can afford to be generous.
const staleAfter = 15 * time.Minute

// OwnerEnv names the variable the wrapper exports into the agent's environment,
// where the agent's hooks inherit it.
const OwnerEnv = "POEMTEA_OWNER"

func Dir() string {
	if d := os.Getenv("POEMTEA_DIR"); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".poemtea"
	}
	return filepath.Join(home, ".poemtea", "sessions")
}

func path(id string) string {
	if id == "" {
		id = "default"
	}
	safe := make([]byte, 0, len(id))
	for i := 0; i < len(id); i++ {
		c := id[i]
		if c == '-' || c == '_' || (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
			safe = append(safe, c)
		}
	}
	if len(safe) == 0 {
		safe = []byte("default")
	}
	return filepath.Join(Dir(), string(safe)+".json")
}

// Set records whether a session is working. Written atomically, so a reader
// polling several times a second can never catch half a file.
func Set(owner, id string, busy bool) error {
	if err := os.MkdirAll(Dir(), 0o755); err != nil {
		return err
	}
	b, err := json.Marshal(Session{
		Owner:   owner,
		Busy:    busy,
		Updated: time.Now().Unix(),
	})
	if err != nil {
		return err
	}
	p := path(id)
	tmp := p + ".tmp" + strconv.Itoa(os.Getpid())
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

// AnyBusy reports whether any session belonging to owner is working.
//
// It also sweeps records that have gone quiet, whoever owns them. Scoping the
// sweep to our own owner would have leaked: every agent running outside a
// wrapper writes records with an empty owner — the desktop app, another
// terminal, anything with the hooks installed — and nobody would ever have been
// responsible for cleaning those up. A record untouched for staleAfter is dead
// regardless of whose it was, and a live owner rewrites its own records far
// more often than that.
func AnyBusy(owner string) bool {
	busy := false
	forEach(func(full string, s Session) {
		if time.Since(time.Unix(s.Updated, 0)) > staleAfter {
			_ = os.Remove(full)
			return
		}
		if s.Owner == owner && s.Busy {
			busy = true
		}
	})
	return busy
}

// ClearOwner removes this wrapper's records on the way out, so a session that
// ended while busy cannot leave anything behind.
func ClearOwner(owner string) {
	forEach(func(full string, s Session) {
		if s.Owner == owner {
			_ = os.Remove(full)
		}
	})
}

func forEach(fn func(path string, s Session)) {
	entries, err := os.ReadDir(Dir())
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		full := filepath.Join(Dir(), e.Name())
		b, err := os.ReadFile(full)
		if err != nil {
			continue
		}
		var s Session
		if json.Unmarshal(b, &s) != nil {
			continue
		}
		fn(full, s)
	}
}
