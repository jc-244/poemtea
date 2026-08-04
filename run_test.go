package main

import (
	"testing"
	"time"
)

func TestTurnIsOver(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	linger := 1.5

	for _, tc := range []struct {
		name          string
		raisedByAgent bool
		busy          bool
		idleSince     time.Time
		want          bool
	}{
		{
			// The regression this function exists for. A screensaver layer goes
			// up on a machine that has been quiet for half a minute, so the
			// agent went idle long ago — under the old rule the layer came down
			// on the very next frame, went up again on the one after, and
			// flickered for as long as nobody touched the keyboard.
			name:          "screensaver layer ignores a long-finished turn",
			raisedByAgent: false,
			idleSince:     now.Add(-30 * time.Second),
			want:          false,
		},
		{
			name:          "agent layer comes down once the turn has been over for linger",
			raisedByAgent: true,
			idleSince:     now.Add(-2 * time.Second),
			want:          true,
		},
		{
			name:          "agent layer stays up until linger has passed",
			raisedByAgent: true,
			idleSince:     now.Add(-1 * time.Second),
			want:          false,
		},
		{
			name:          "a working agent is not a finished one",
			raisedByAgent: true,
			busy:          true,
			idleSince:     now.Add(-30 * time.Second),
			want:          false,
		},
		{
			// No turn has ever ended, so there is nothing to be over.
			name:          "no turn has ended yet",
			raisedByAgent: true,
			idleSince:     time.Time{},
			want:          false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := turnIsOver(tc.raisedByAgent, tc.busy, tc.idleSince, now, linger)
			if got != tc.want {
				t.Errorf("turnIsOver = %v, want %v", got, tc.want)
			}
		})
	}
}
