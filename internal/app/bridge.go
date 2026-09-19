package app

import (
	"sync"
	"time"

	"github.com/intentdriven/Gropius/internal/config"
)

// Bridge is a bridge to a third-party messaging platform, as the app needs it:
// something that can be put into whatever state the settings describe, that
// can say what it is doing, and that can be shut down.
//
// It is an interface of plain values rather than the bridge's own type so
// that nothing in the composition root, the control plane or the panel has to
// know which platform is being bridged. Today there is one (Discord,
// adr-2609181004167097); a second would implement this and be applied beside
// it.
type Bridge interface {
	// Apply puts the settings in force: start, stop, or restart under a new
	// token. It is called at startup and after every save, never with one of
	// the app's locks held (adr-2609091239058072: the bridge takes neither).
	Apply(on bool, token string)
	// State reports what the bridge is doing, when its live session was
	// established, and why it stopped.
	State() (state string, since time.Time, reason string)
	// Close stops it for good.
	Close() error
}

// BridgeState is what the panel and `gropius config show` are told about the
// bridge: off, connecting, connected since a moment, or stopped for a reason.
//
// The reason is the bridge's own sentence about a credential or a
// configuration — never the token, and never anything a platform sent.
type BridgeState struct {
	// State is one of off, connecting, connected, stopped.
	State string `json:"state"`
	// Since is when the bridge last connected, in whole UTC seconds: the live
	// session's moment while it is connected, and the previous one while it
	// is re-opening or stopped. It is what answers "the panel shows when it
	// last connected", which is asked of exactly the path where the bridge is
	// NOT connected. It is absent until the bridge has connected under the
	// token now in force: a credential that has never connected is credited
	// with no moment, whatever the one before it did.
	Since int64 `json:"since,omitempty"`
	// Reason is why the bridge stopped, absent when it has not.
	Reason string `json:"reason,omitempty"`
}

// bridgeOff is the state of an app with no bridge wired at all — every test,
// and any build that does not carry one. It is a value rather than an absence
// so the panel has one shape to draw.
const bridgeOff = "off"

// bridges holds the app's bridge behind a lock of its own.
//
// A leaf lock outside adr-2609091239058072's order: nothing is called while it
// is held except the bridge's own methods, and the bridge takes none of the
// app's locks.
type bridges struct {
	mu sync.Mutex
	b  Bridge
}

// SetBridge wires the bridge and puts the current settings in force.
//
// It is called once, from the composition root, after the gateway exists —
// the bridge asks the gateway for completions, so it cannot be built with the
// app. A nil bridge leaves the app exactly as it was before bridges existed.
func (a *App) SetBridge(b Bridge) {
	a.bridge.mu.Lock()
	a.bridge.b = b
	a.bridge.mu.Unlock()
	if b == nil {
		return
	}
	a.applyBridge(a.Config())
}

// applyBridge puts the switch and the token into force, at startup and after
// every save.
//
// One function, called from both, for the reason applyStatistics gives: a
// start and a save that worked the state out separately would be two answers
// to whether the bridge is running, and the one nobody can see is the one that
// would be wrong.
func (a *App) applyBridge(c config.Config) {
	b := a.currentBridge()
	if b == nil {
		return
	}
	b.Apply(c.DiscordBridge, c.DiscordToken)
}

// BridgeState is what the bridge is doing, for the snapshot.
func (a *App) BridgeState() BridgeState {
	b := a.currentBridge()
	if b == nil {
		return BridgeState{State: bridgeOff}
	}
	state, since, reason := b.State()
	out := BridgeState{State: state, Reason: reason}
	if !since.IsZero() {
		out.Since = since.UTC().Unix()
	}
	return out
}

func (a *App) currentBridge() Bridge {
	a.bridge.mu.Lock()
	defer a.bridge.mu.Unlock()
	return a.bridge.b
}

// closeBridge stops the bridge as the app shuts down. Switching the app off
// closes the connection, which is the same promise the switch makes
// (adr-2609181004167097 condition 1).
func (a *App) closeBridge() error {
	b := a.currentBridge()
	if b == nil {
		return nil
	}
	return b.Close()
}
