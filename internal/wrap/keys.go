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
// Only something deliberate counts: a key, a button going down, the wheel
// turned up or down. Everything the terminal says on its own does not.
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

// The two flags a mouse button code can carry, in both encodings.
const (
	mouseMoving = 32 // the pointer moved, with or without a button held
	mouseWheel  = 64 // the wheel turned; the low two bits say which way
)

// deliberate reports whether a mouse report is you doing something rather than
// the pointer merely being somewhere.
//
// A button going down is you, and so is the wheel turned up or down. Motion is
// not — the pointer crossing the window while you are reading is exactly the
// case this whole file exists to ignore, and a drag is motion too. Nor is a
// release: it is the far end of a press that has already been counted.
//
// The wheel's direction is in the code, so scrolling sideways can be told from
// scrolling down and left alone. A trackpad emits a great many of those on its
// way through a diagonal gesture.
func deliberate(code int, press bool) bool {
	switch {
	case code&mouseMoving != 0:
		return false
	case code&mouseWheel != 0:
		return code&3 <= 1 // 0 up, 1 down; 2 and 3 are sideways
	default:
		return press
	}
}

// sgrButton reads the button number at the head of an SGR mouse report.
func sgrButton(body []byte) int {
	n := 0
	for _, c := range body {
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int(c-'0')
	}
	return n
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
		// The original mouse encoding: three raw bytes follow, and two of them
		// can look like anything, so they are stepped over rather than parsed.
		// The first is the button, offset by 32 like the coordinates.
		if j+3 >= len(b) {
			return i, false, false
		}
		code := int(b[j+1]) - 32
		// This encoding has no separate release event: the low two bits say
		// which button, and all three set says one was let go.
		return j + 4, true, deliberate(code, code&3 != 3)
	case (final == 'M' || final == 'm') && len(body) > 0 && body[0] == '<':
		// SGR: ESC [ < button ; column ; row, ending in M for a press and m
		// for a release.
		return j + 1, true, deliberate(sgrButton(body[1:]), final == 'M')
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
