package discord

import (
	"testing"
	"time"
)

// A bridge Discord stopped is started again by saving the same settings: the
// panel's Save is the restart (the 2026-09-20 decision on
// iss-2609190312326963). The stopped run leaves its session handle closed
// rather than nil, and a re-save under the same switch and token must read
// that as not running.
func TestSavingTheSameSettingsRestartsABridgeDiscordStopped(t *testing.T) {
	f := newFakeDiscord(t)
	b := answering(t, f)
	connected(t, f, b)

	// 4014: Discord refusing the intents, which stops the bridge for good
	// and leaves the operator nothing in Settings to change.
	f.refuse(4014)
	if state, _ := waitState(t, b, StateStopped); state != StateStopped {
		t.Fatalf("state = %q after a fatal close, want stopped", state)
	}
	dials := f.dials()

	b.Apply(true, "a-token")
	if state, _ := waitState(t, b, StateConnected); state != StateConnected {
		t.Fatalf("state = %q after saving the same settings, want the bridge connected again", state)
	}
	if f.dials() <= dials {
		t.Errorf("saving the same settings dialled nothing: %d dials before, %d after", dials, f.dials())
	}

	// And a save while it runs is still not a restart.
	dials = f.dials()
	b.Apply(true, "a-token")
	time.Sleep(quiet)
	if f.dials() != dials {
		t.Errorf("saving the same settings under a running bridge reconnected it: %d then %d", dials, f.dials())
	}
}
