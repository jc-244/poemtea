package wrap

// Not everything arriving on stdin is you.
//
// The host turns on mouse reporting and focus events, so the terminal sends it
// a stream of escape sequences whenever the pointer moves across the window or
// you switch to another application — and it answers the host's questions on
// the same channel. Treating any byte as a key press means the picture flinches
// away from a mouse drifting past, or from you clicking on a different window
// entirely, which is not what "you are back" means.
//
// Only a deliberate press counts. Everything the terminal says on its own does
// not.
type keyDetect struct{ pending []byte }

// sawKey reports whether a chunk from the keyboard contains a real key press.
// Input itself is never filtered — every byte still reaches the host untouched.
// This only decides whether to take the picture away.
func (k *keyDetect) sawKey(p []byte) bool {
	buf := p
	if len(k.pending) > 0 {
		buf = append(k.pending, p...)
		k.pending = nil
	}

	for i := 0; i < len(buf); {
		if buf[i] != 0x1b {
			return true // ordinary typing
		}
		end, complete, key := scanKey(buf, i)
		if !complete {
			k.pending = append(k.pending[:0], buf[i:]...)
			// A lone escape byte with nothing behind it is the Escape key. It
			// would otherwise wait forever for a sequence that never comes.
			return len(k.pending) == 1
		}
		if key {
			return true
		}
		i = end
	}
	return false
}

// scanKey returns where the sequence ends, whether it was complete, and whether
// it was a person pressing something.
func scanKey(b []byte, i int) (end int, complete, key bool) {
	if i+1 >= len(b) {
		return i, false, false
	}
	switch b[i+1] {
	case '[':
		return scanKeyCSI(b, i)
	case 'O':
		// SS3: function keys and the arrows in application mode.
		if i+2 >= len(b) {
			return i, false, false
		}
		return i + 3, true, true
	case ']', 'P', '_', '^':
		// OSC and friends coming *inward* are the terminal answering a question
		// the host asked — a colour report, a capability. Never a key.
		for j := i + 2; j < len(b); j++ {
			if b[j] == 0x07 {
				return j + 1, true, false
			}
			if b[j] == 0x1b && j+1 < len(b) && b[j+1] == '\\' {
				return j + 2, true, false
			}
		}
		return i, false, false
	default:
		// Escape followed by a character is Alt+that character.
		return i + 2, true, true
	}
}

func scanKeyCSI(b []byte, i int) (end int, complete, key bool) {
	j := i + 2
	for j < len(b) && b[j] >= 0x20 && b[j] <= 0x3f {
		j++
	}
	if j >= len(b) {
		return i, false, false
	}
	final := b[j]
	body := b[i+2 : j]

	switch {
	case final == 'M' && len(body) == 0:
		// The original mouse encoding: three raw bytes follow, and they can look
		// like anything, so they have to be stepped over rather than parsed.
		if j+3 >= len(b) {
			return i, false, false
		}
		return j + 4, true, false
	case (final == 'M' || final == 'm') && len(body) > 0 && body[0] == '<':
		return j + 1, true, false // SGR mouse report
	case (final == 'I' || final == 'O') && len(body) == 0:
		return j + 1, true, false // focus in / focus out
	case final == 'R' || final == 'c' || final == 'y' || final == 'n' || final == 't':
		return j + 1, true, false // the terminal answering: position, identity, mode, status, window
	default:
		// Arrows, Home, End, function keys, bracketed-paste markers and the
		// pasted text itself.
		return j + 1, true, true
	}
}
