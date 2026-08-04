package wrap

import "bytes"

// While the picture is on screen we stop forwarding the host's output to the
// terminal — but "stop forwarding everything" hangs it.
//
// The probe against Claude Code caught this: it emits ESC[c (Device Attributes)
// and enables focus events, mouse reporting and bracketed paste. Those are not
// drawing, they are a conversation with the terminal, and some of them block
// waiting for a reply. Swallow the question and the host waits forever for an
// answer that will never come.
//
// So muting is a filter, not a switch: drawing is withheld, dialogue goes
// through untouched.

// Filter splits a byte stream into escape sequences and passes along only the
// ones that must not be withheld. It is a streaming parser — a sequence can be
// split across two reads from the pty, so partial state is carried over.
type Filter struct {
	pending []byte

	// CursorVisible is the only thing this program still needs to remember
	// about the host, and it is remembered by watching one sequence go past
	// rather than by modelling anything. We hide the cursor while the layer is
	// up; this is what to put back.
	CursorVisible bool
}

// Split partitions the host's output into what reaches the terminal now and
// what is held back to be replayed once the layer comes down.
//
//	unmuted — everything through, except the alternate-screen switch, which is
//	          withheld permanently so the buffer stays ours to use as a layer
//	muted   — only the host's conversation with the terminal: queries that block
//	          waiting for an answer, and modes it assumes took effect
//
// Held bytes are never interpreted, only stored. Replaying them verbatim is
// what makes the restore lossless — anything we did not model is something we
// would otherwise silently drop.
func (f *Filter) Split(forward, hold []byte, p []byte, muted bool) (fwd, held []byte) {
	buf := p
	if len(f.pending) > 0 {
		buf = append(f.pending, p...)
		f.pending = nil
	}

	i, run := 0, 0 // run: start of the current stretch of plain text
	flushRun := func(to int) {
		if to <= run {
			return
		}
		if muted {
			hold = append(hold, buf[run:to]...)
		} else {
			forward = append(forward, buf[run:to]...)
		}
	}

	for i < len(buf) {
		if buf[i] != 0x1b {
			i++
			continue
		}
		flushRun(i)

		end, complete, kind := scanEscape(buf, i)
		if !complete {
			f.pending = append(f.pending[:0], buf[i:]...)
			return forward, hold
		}
		seq := buf[i:end]
		if n := len(seq); n > 3 && (seq[n-1] == 'h' || seq[n-1] == 'l') && hasParam(seq[2:n-1], "25") {
			f.CursorVisible = seq[n-1] == 'h'
		}
		switch {
		case kind == altScreen:
			// Never forwarded and never replayed. The alternate buffer is the
			// layer; the host must not be able to take it or pop it.
		case !muted || kind == dialogue:
			forward = append(forward, seq...)
		default:
			hold = append(hold, seq...)
		}
		i = end
		run = i
	}
	flushRun(len(buf))
	return forward, hold
}

type verdict int

const (
	drawing   verdict = iota // paints the screen; held while muted
	dialogue                 // a question or a mode the host assumes took effect
	altScreen                // the alternate-buffer switch
)

// scanEscape finds the end of the escape sequence starting at i and decides
// whether it is drawing or dialogue. Returns (end, complete, verdict).
func scanEscape(b []byte, i int) (int, bool, verdict) {
	if i+1 >= len(b) {
		return i, false, drawing
	}
	switch b[i+1] {
	case '[':
		return scanCSI(b, i)
	case ']':
		// OSC carries both questions (which block) and settings such as the
		// window title (which do not). Only a question has to get through while
		// the layer is up; everything else is held and replayed, on the general
		// principle that the less we let past, the less can surprise us.
		end, complete, _ := scanString(b, i, dialogue)
		if !complete {
			return end, false, drawing
		}
		if bytes.IndexByte(b[i:end], '?') >= 0 {
			return end, true, dialogue
		}
		return end, true, drawing
	case 'P', '_', '^', 'X':
		// DCS / APC / PM / SOS. DCS carries terminal capability queries
		// (XTGETTCAP, DECRQSS) which expect replies.
		return scanString(b, i, dialogue)
	default:
		// Two-byte escapes: charset selection, save/restore cursor, index.
		// All screen state — withhold.
		if i+1 < len(b) && (b[i+1] == '(' || b[i+1] == ')' || b[i+1] == '*' || b[i+1] == '+') {
			if i+2 >= len(b) {
				return i, false, drawing
			}
			return i + 3, true, drawing
		}
		return i + 2, true, drawing
	}
}

func scanCSI(b []byte, i int) (int, bool, verdict) {
	j := i + 2
	for j < len(b) && b[j] >= 0x20 && b[j] <= 0x3f { // params and intermediates
		j++
	}
	for j < len(b) && b[j] >= 0x20 && b[j] <= 0x2f { // more intermediates
		j++
	}
	if j >= len(b) {
		return i, false, drawing
	}
	final := b[j]
	end := j + 1

	body := b[i+2 : j]
	switch final {
	case 'c', 'n':
		// Device Attributes and Device Status Report. Both are questions.
		return end, true, dialogue
	case 'h', 'l':
		// Mode set/reset. These configure the terminal rather than paint it —
		// mouse reporting, bracketed paste, focus events — and the host assumes
		// they took effect. Two exceptions:
		//   1049 — leaving the alternate screen while we are painting over it
		//          would expose the buffer underneath; the host only does that
		//          on its way out anyway.
		//     25 — cursor visibility. A cursor blinking in the middle of the
		//          rain is the single most obvious tell. We hold it hidden and
		//          restore the host's intent when we hand the screen back.
		if hasParam(body, "1049") || hasParam(body, "47") || hasParam(body, "1047") {
			return end, true, altScreen
		}
		if hasParam(body, "25") {
			return end, true, drawing
		}
		return end, true, dialogue
	case 'p':
		// DECRQM (request mode) ends in $p and expects a report.
		if len(body) > 0 && body[len(body)-1] == '$' {
			return end, true, dialogue
		}
		return end, true, drawing
	default:
		return end, true, drawing
	}
}

func hasParam(body []byte, want string) bool {
	start := 0
	for start < len(body) && (body[start] == '?' || body[start] == '>' || body[start] == '<' || body[start] == '=') {
		start++
	}
	cur := make([]byte, 0, 8)
	flush := func() bool {
		ok := string(cur) == want
		cur = cur[:0]
		return ok
	}
	for _, c := range body[start:] {
		if c == ';' {
			if flush() {
				return true
			}
			continue
		}
		cur = append(cur, c)
	}
	return flush()
}

// scanString handles the string-terminated families (OSC, DCS, APC, PM, SOS),
// which end at BEL or at ST (ESC \).
func scanString(b []byte, i int, v verdict) (int, bool, verdict) {
	for j := i + 2; j < len(b); j++ {
		if b[j] == 0x07 {
			return j + 1, true, v
		}
		if b[j] == 0x1b && j+1 < len(b) && b[j+1] == '\\' {
			return j + 2, true, v
		}
		if b[j] == 0x9c {
			return j + 1, true, v
		}
	}
	return i, false, v
}
