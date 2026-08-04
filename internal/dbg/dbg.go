// Package dbg is the only channel this program has for saying anything.
//
// poemtea sits between a user and the tool they are working in, and it owns the
// screen at exactly the moments something might go wrong. Printing a warning is
// not an option: the warning would land in the middle of someone's terminal,
// which is a worse outcome than the bug. So diagnostics go to a file, only when
// POEMTEA_DEBUG names one, and only for events that change state.
//
// Set it when something looks wrong:
//
//	POEMTEA_DEBUG=/tmp/poemtea.log poemtea run -- claude
package dbg

import (
	"fmt"
	"os"
	"sync"
	"time"
)

var (
	mu   sync.Mutex
	path = os.Getenv("POEMTEA_DEBUG")
)

// Printf records one event. Keep it to state transitions and failures — a log
// that fires per frame or per read is not a log, it is a second bug.
func Printf(format string, args ...any) {
	if path == "" {
		return
	}
	mu.Lock()
	defer mu.Unlock()

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s  %s\n", time.Now().Format("15:04:05.000"), fmt.Sprintf(format, args...))
}
