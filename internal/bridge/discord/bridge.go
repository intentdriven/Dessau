// Package discord bridges direct messages and mentions of a Discord bot to a
// model on this Mac.
//
// It is the first thing in Gropius that carries a conversation off the machine,
// and it does so only under adr-2609181004167097: opt-in per bridge and off by
// default, outbound only (no port is opened and the bind mode is untouched),
// with the bot token held as a secret on every surface the API key is, and with
// the log recording the FACT of a bridged request and never its content.
//
// The bridge is, by its nature, a reader of both sides of a conversation — the
// message it relays and the answer it posts — which makes it the second such
// reader beside the merge that adr-2609061610102325 grants. That reading is
// admitted by name in internal/archtest's readers' lists, with this reason.
// What is recorded is a separate matter and is unchanged: the statistics store
// gains one fixed class, the source, and nothing a platform supplied; the log
// line carries the channel and user identifiers as opaque numbers, the model,
// the sizes and the timing.
//
// Nothing of a conversation is written to disk. A channel's history lives in
// memory for as long as the bridge is running and goes when it stops.
package discord

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/intentdriven/Gropius/internal/gateway"
)

// The bridge's four states, as the panel and `gropius config show` spell them.
const (
	// StateOff is the switch off: no connection, no goroutine, nothing held.
	StateOff = "off"
	// StateConnecting is a connection being opened or re-opened, including
	// every retry after a drop.
	StateConnecting = "connecting"
	// StateConnected is a live gateway session; the time beside it is when it
	// was established.
	StateConnected = "connected"
	// StateStopped is the switch on and the bridge not running, with the
	// reason beside it. Only Discord refusing the credentials or the intents
	// produces it: everything else is retried, because everything else is
	// weather.
	StateStopped = "stopped"
)

// Options configures a Bridge. Every seam that reaches the network or the
// clock is here so that a test can speak the whole protocol to it without
// Discord ever being contacted.
type Options struct {
	// Ask runs one completion in process — the gateway's own entry point, so
	// a bridged request takes the same admission, merge, rewrite and
	// observation an HTTP request takes.
	Ask func(ctx context.Context, req gateway.AskRequest) error
	// ChatModels lists the repo ids a channel may be answered by, the
	// server's default first. It is read afresh for every `/model` and every
	// answer, so a model downloaded while the bridge is running is offered
	// without a restart.
	ChatModels func() []string
	// ServedContext is the window a model is served at on this Mac, or 0 when
	// that is not known. A channel's history is bounded by it.
	ServedContext func(model string) int64
	// Log is the operator's log. Nothing of a message ever reaches it.
	Log *slog.Logger
	// GatewayURL is Discord's gateway, overridden by tests. Empty means the
	// real one.
	GatewayURL string
	// APIBase is Discord's REST root, overridden by tests. Empty means the
	// real one.
	APIBase string
	// HTTPClient is used for both the WebSocket handshake and the REST calls.
	// Empty means a client of this package's own, which verifies TLS the way
	// Go verifies it everywhere: nothing here relaxes certificate checking,
	// and the one place that could — a transport handed in — is the caller's
	// own and is never built with InsecureSkipVerify by anything in this tree.
	HTTPClient *http.Client
	// Now is the clock. Tests move it rather than sleeping.
	Now func() time.Time
}

// Bridge owns at most one gateway session and the goroutine that runs it.
//
// It takes none of the app's locks and is never called with one held
// (adr-2609091239058072): App.SetConfig calls Apply after the settings are in
// force and the save lock is the only thing it holds, and Apply does its work
// behind a mutex of its own that nothing else in the process takes.
type Bridge struct {
	opts Options
	log  *slog.Logger
	now  func() time.Time

	mu sync.Mutex
	// running describes the session the goroutine below is running, so Apply
	// can tell "already running under these settings" from "running under
	// different ones".
	on      bool
	token   string
	cancel  context.CancelFunc
	done    chan struct{}
	state   string
	since   time.Time
	reason  string
	closing bool
}

// New builds a Bridge. It connects to nothing: the bridge is off until Apply
// is called with the switch on, which is condition 1 of the ADR expressed as
// the only way this type can ever open a connection.
func New(opts Options) *Bridge {
	if opts.Log == nil {
		opts.Log = slog.Default()
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.GatewayURL == "" {
		opts.GatewayURL = defaultGatewayURL
	}
	if opts.APIBase == "" {
		opts.APIBase = defaultAPIBase
	}
	if opts.HTTPClient == nil {
		opts.HTTPClient = &http.Client{Timeout: restTimeout}
	}
	// A bridge assembled without a server behind it answers nothing rather
	// than panicking on the first message: the three seams below are the
	// composition root's to supply, and a build that forgot one must fail
	// visibly on the panel rather than take the process down from a goroutine
	// somebody else's message started.
	if opts.Ask == nil {
		opts.Ask = func(context.Context, gateway.AskRequest) error {
			return errors.New("this build has no completion path wired to the bridge")
		}
	}
	if opts.ChatModels == nil {
		opts.ChatModels = func() []string { return nil }
	}
	if opts.ServedContext == nil {
		opts.ServedContext = func(string) int64 { return 0 }
	}
	return &Bridge{opts: opts, log: opts.Log, now: opts.Now, state: StateOff}
}

// Apply puts the settings in force: starts the bridge, stops it, or restarts
// it under a new token.
//
// It is called at startup and after every save, and it is the only way the
// bridge changes. A save that did not touch either setting is a no-op, which
// is what keeps an unrelated save from dropping a live session.
func (b *Bridge) Apply(on bool, token string) {
	b.mu.Lock()
	if b.closing {
		b.mu.Unlock()
		return
	}
	// The switch on with no token is not an error and never refuses the save
	// (itd-2609180959397172): the bridge says so as its own state and nothing
	// else happens.
	if on && token == "" {
		b.stopLocked()
		b.on, b.token = true, ""
		b.setStateLocked(StateStopped, "no bot token — paste one in Settings")
		b.mu.Unlock()
		return
	}
	if !on {
		b.stopLocked()
		b.on, b.token = false, ""
		b.setStateLocked(StateOff, "")
		b.mu.Unlock()
		return
	}
	if b.on && b.token == token && b.done != nil {
		b.mu.Unlock()
		return // already running under exactly these settings
	}
	b.stopLocked()
	b.on, b.token = true, token
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	b.cancel, b.done = cancel, done
	b.setStateLocked(StateConnecting, "")
	b.mu.Unlock()

	go func() {
		defer close(done)
		b.run(ctx, token)
	}()
}

// Close stops the bridge for good and waits for its goroutine.
func (b *Bridge) Close() error {
	b.mu.Lock()
	b.closing = true
	b.stopLocked()
	b.on, b.token = false, ""
	b.setStateLocked(StateOff, "")
	b.mu.Unlock()
	return nil
}

// stopLocked cancels the running session and waits for it to finish. It is
// called with b.mu held and releases it across the wait, because the
// goroutine it is waiting for takes the same mutex to report its state.
func (b *Bridge) stopLocked() {
	cancel, done := b.cancel, b.done
	b.cancel, b.done = nil, nil
	if cancel == nil {
		return
	}
	cancel()
	b.mu.Unlock()
	<-done
	b.mu.Lock()
}

// State reports what the bridge is doing: one of the four words above, when
// the live session was established, and the reason it stopped.
//
// It is three plain values rather than a struct so that nothing outside this
// package has to import it to ask. internal/app renders them onto the snapshot
// the panel reads.
func (b *Bridge) State() (state string, since time.Time, reason string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.state, b.since, b.reason
}

func (b *Bridge) setState(state, reason string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.setStateLocked(state, reason)
}

func (b *Bridge) setStateLocked(state, reason string) {
	b.state, b.reason = state, reason
	if state == StateConnected {
		b.since = b.now()
		return
	}
	if state == StateOff {
		b.since = time.Time{}
	}
}

// reconnectBackoff is how long the bridge waits before re-opening a session
// that dropped, doubling to a ceiling.
//
// A ceiling of half a minute rather than minutes: the case this exists for is
// a Mac that went to sleep, and the person who wakes it expects the bot to
// answer them, not to be down for as long as the backoff happened to have
// grown while the lid was shut.
const (
	minBackoff = 2 * time.Second
	maxBackoff = 30 * time.Second
)

// run keeps a session open for as long as the bridge is on.
//
// The one thing that ends it short of the switch being turned off is Discord
// refusing the credentials or the intents, which is not weather and will not
// come right by being retried: the ADR's condition is that the reason reaches
// the panel, so the loop stops and says why.
func (b *Bridge) run(ctx context.Context, token string) {
	backoff := minBackoff
	var resume resumeState
	for ctx.Err() == nil {
		b.setState(StateConnecting, "")
		out := b.runSession(ctx, token, &resume)
		if ctx.Err() != nil {
			return
		}
		if out.fatal != "" {
			b.log.Error("the Discord bridge stopped", "reason", out.fatal)
			b.setState(StateStopped, out.fatal)
			return
		}
		if !out.resumable {
			resume.clear()
		}
		if out.connected {
			// A session that actually ran is not a failure to connect, so the
			// next drop starts from the shortest wait again rather than from
			// whatever the last outage grew it to.
			backoff = minBackoff
		}
		if out.err != nil {
			// The error is the operator's, at the detailed level: a dial
			// failure names the host and the network error, which is exactly
			// what somebody diagnosing a bridge that will not connect needs
			// and exactly what nobody watching a working one wants.
			b.log.Debug("the Discord bridge lost its session and will reconnect",
				"err", out.err, "in", backoff)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff *= 2; backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}
