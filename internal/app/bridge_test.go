package app

import (
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/Gropius/internal/config"
)

// fakeBridge records what the app asked of it.
type fakeBridge struct {
	mu      sync.Mutex
	applied []applied
	closed  bool

	state  string
	since  time.Time
	reason string
}

type applied struct {
	On    bool
	Token string
}

func (f *fakeBridge) Apply(on bool, token string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.applied = append(f.applied, applied{On: on, Token: token})
	switch {
	case !on:
		f.state, f.reason = "off", ""
	case token == "":
		f.state, f.reason = "stopped", "no bot token"
	default:
		f.state, f.reason, f.since = "connected", "", time.Unix(1_700_000_000, 0)
	}
}

// reconnecting is the state a bridge is in between a dropped session and the
// next attempt: not connected, and still knowing when it last was.
func (f *fakeBridge) reconnecting() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.state, f.reason = "connecting", ""
}

func (f *fakeBridge) State() (string, time.Time, string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.state, f.since, f.reason
}

func (f *fakeBridge) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = true
	return nil
}

func (f *fakeBridge) calls() []applied {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]applied(nil), f.applied...)
}

// newBridgeTestApp is newTestApp with a configuration of the caller's own.
func newBridgeTestApp(t *testing.T, cfg config.Config) *App {
	t.Helper()
	a, err := New(Options{Paths: config.NewPaths(t.TempDir()), Config: cfg})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = a.Close() })
	return a
}

// The switch and the token reach the bridge when they are wired and again on
// every save, so a bridge switched on in Settings starts without a restart
// and one switched off stops there and then (adr-2609181004167097 condition
// 1).
func TestTheBridgeIsAppliedAtStartupAndOnEverySave(t *testing.T) {
	cfg := config.Default()
	cfg.DiscordBridge = true
	cfg.DiscordToken = "the-first-token"
	a := newBridgeTestApp(t, cfg)

	fake := &fakeBridge{}
	a.SetBridge(fake)
	if got := fake.calls(); len(got) != 1 || !got[0].On || got[0].Token != "the-first-token" {
		t.Fatalf("wiring the bridge applied %v, want the stored settings once", got)
	}

	next := a.Config()
	next.DiscordToken = "a-rotated-token"
	if err := a.SetConfig(next); err != nil {
		t.Fatalf("SetConfig: %v", err)
	}
	calls := fake.calls()
	if len(calls) != 2 || calls[1].Token != "a-rotated-token" {
		t.Fatalf("a save applied %v, want the rotated token", calls)
	}

	off := a.Config()
	off.DiscordBridge = false
	if err := a.SetConfig(off); err != nil {
		t.Fatalf("SetConfig: %v", err)
	}
	calls = fake.calls()
	if len(calls) != 3 || calls[2].On {
		t.Fatalf("switching off applied %v", calls)
	}
}

// What the bridge is doing reaches the snapshot the panel draws, which is the
// third surface the switch, the token and the state live on.
func TestTheBridgesStateReachesTheSnapshot(t *testing.T) {
	a := newBridgeTestApp(t, config.Default())

	// An app with no bridge wired — every test, and any build without one —
	// reports off rather than nothing, so the panel has one shape to draw.
	if got := a.BridgeState(); got.State != bridgeOff || got.Since != 0 {
		t.Errorf("an app with no bridge reports %+v, want off", got)
	}

	fake := &fakeBridge{}
	a.SetBridge(fake)
	fake.Apply(true, "a-token")
	got := a.BridgeState()
	if got.State != "connected" {
		t.Errorf("state = %q", got.State)
	}
	if got.Since != 1_700_000_000 {
		t.Errorf("since = %d, want the moment the session was established", got.Since)
	}

	fake.Apply(true, "")
	got = a.BridgeState()
	if got.State != "stopped" || got.Reason == "" {
		t.Errorf("state = %+v, want stopped with a reason", got)
	}
}

// The moment the bridge last connected reaches the snapshot while it is NOT
// connected, which is the only path the criterion it answers is about: a
// session drops, the bridge re-opens it by itself, and the panel says when it
// last worked (iss-2609190242334438).
func TestTheLastConnectedMomentReachesTheSnapshotWhileReconnecting(t *testing.T) {
	a := newBridgeTestApp(t, config.Default())
	fake := &fakeBridge{}
	a.SetBridge(fake)
	fake.Apply(true, "a-token")

	fake.reconnecting()
	got := a.BridgeState()
	if got.State != "connecting" {
		t.Fatalf("state = %q, want connecting", got.State)
	}
	if got.Since != 1_700_000_000 {
		t.Errorf("since = %d, want the moment the bridge last connected", got.Since)
	}
}

// A save is never refused over the bridge, whatever the token looks like: a
// credential Discord will not accept is the bridge's state to report and not
// a reason to turn away an operator changing something else.
func TestASaveIsNeverRefusedOverTheBridgesSettings(t *testing.T) {
	a := newBridgeTestApp(t, config.Default())
	a.SetBridge(&fakeBridge{})

	for _, token := range []string{"", "not-a-token", "   "} {
		next := a.Config()
		next.DiscordBridge = true
		next.DiscordToken = token
		next.IdleTimeoutSec = 123 // the unrelated setting the operator came for
		if err := a.SetConfig(next); err != nil {
			t.Fatalf("a save carrying token %q was refused: %v", token, err)
		}
		if a.Config().IdleTimeoutSec != 123 {
			t.Error("the unrelated setting did not land")
		}
	}
}

// Shutting the app down closes the bridge, which is the same promise the
// switch makes: nothing goes on leaving this Mac once Gropius stops.
func TestClosingTheAppClosesTheBridge(t *testing.T) {
	a := newBridgeTestApp(t, config.Default())
	fake := &fakeBridge{}
	a.SetBridge(fake)
	if err := a.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if !fake.closed {
		t.Error("the app shut down and left the bridge connected")
	}
}
