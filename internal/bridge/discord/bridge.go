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
// memory for as long as the bridge is running — across a dropped session and
// the resume that follows it — and goes the moment it stops running, whether
// that is the switch, the app closing, or Discord refusing the credentials.
package discord

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
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
	//
	// Empty means a client of this package's own, with no transport of its
	// own — so the handshake is verified the way Go verifies every handshake,
	// and nothing in this package can arrange otherwise. That is what
	// TestTheBridgeNeverWeakensTheOutboundHandshake holds: this is the one
	// place in the product that dials out holding a bearer credential, and the
	// handshake is the whole of what stands between the token and anyone on
	// the path.
	//
	// The Timeout on the default client bounds the handshake only. The
	// WebSocket library moves it onto the dial's context and hands the
	// connection a client with none, so a live session is not cut off after
	// it; the REST calls bound themselves with a context apiece.
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

	// applyMu serialises whole applications of the settings. mu guards the
	// values; this guards the sequence — stop the running session, wait for
	// it, start the next — which is not one step and must not interleave with
	// another Apply's, or a save and a shutdown racing would leave a session
	// running that nothing holds a handle to. It is taken before mu and
	// nothing taken under mu takes it, so it adds no order to anything else.
	applyMu sync.Mutex

	mu sync.Mutex
	// running describes the session the goroutine below is running, so Apply
	// can tell "already running under these settings" from "running under
	// different ones".
	on     bool
	token  string
	cancel context.CancelFunc
	done   chan struct{}
	state  string
	// since is the moment the bridge last connected UNDER THE TOKEN IN FORCE.
	// It is kept while the session is being re-opened, so the panel can say
	// when it last connected rather than losing the fact on exactly the path
	// the criterion is about; it goes when the switch goes off, and when the
	// token changes, because a credential that has not connected has no
	// moment to show.
	since   time.Time
	reason  string
	closing bool
	// convos is every channel this bridge has seen while it has been on.
	//
	// IT BELONGS TO THE BRIDGE AND NOT TO THE CONNECTION (iss-2609190241509478).
	// A gateway session drops whenever the Wi-Fi blinks or the lid closes,
	// and the bridge resumes by itself; a conversation held on the session
	// would make every one of those a silent `/reset` that also forgot the
	// model the channel chose. The bound itd-2609180959397172 asks for is the
	// bridge RUNNING: its life is exactly the life of the goroutine below,
	// which is made with it and drops it as it ends — the switch, a token
	// change, the app closing, and Discord refusing the credentials alike.
	// Nothing of a conversation is ever written to disk either way.
	convos *conversations
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
	// Trimmed on the way in (iss-2609190106563320). A token copied from a
	// browser or a terminal usually carries a trailing newline, and Discord
	// refuses it with 4004 — which puts "paste a fresh one" on the panel,
	// advice that fails again for the same invisible reason. A bot token has
	// no leading or trailing space in it, so there is nothing to lose.
	token = strings.TrimSpace(token)
	b.applyMu.Lock()
	defer b.applyMu.Unlock()
	b.mu.Lock()
	if b.closing {
		b.mu.Unlock()
		return
	}
	// The last-connected moment belongs to the credential that connected. A
	// token the operator has just pasted has never connected, whatever the
	// one before it did, and a panel that credited it with the old one's
	// moment would be answering the operator's "did this work?" with a
	// session that was somebody else's (iss-2609190312313645). Before the
	// branches, so it covers the switch on under a new token, the token being
	// changed under a running bridge, and the token being taken away.
	if b.token != token {
		b.since = time.Time{}
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
	b.convos = newConversations()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	b.cancel, b.done = cancel, done
	b.setStateLocked(StateConnecting, "")
	b.mu.Unlock()

	go func() {
		// The conversations go with this goroutine, WHICHEVER WAY IT ENDS
		// (iss-2609190312064731): the switch, a token change, the app
		// closing, or Discord refusing the credentials and stopping the loop
		// for good. The fatal path returns without ever reaching stopLocked,
		// so a store dropped there alone outlived the bridge on exactly the
		// stop an operator cannot undo from Settings.
		//
		// Registered BEFORE the close, so it runs after it: whoever is
		// waiting on done goes on to build the next store, and this must not
		// be able to drop that one.
		defer close(done)
		defer b.dropConversations()
		b.run(ctx, token)
	}()
}

// Close stops the bridge for good and waits for its goroutine.
func (b *Bridge) Close() error {
	b.applyMu.Lock()
	defer b.applyMu.Unlock()
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

// State reports what the bridge is doing: one of the four words above, when it
// last connected — the live session's moment while it is connected, and the
// previous one while it is not — and the reason it stopped.
//
// It is three plain values rather than a struct so that nothing outside this
// package has to import it to ask. internal/app renders them onto the snapshot
// the panel reads.
func (b *Bridge) State() (state string, since time.Time, reason string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.state, b.since, b.reason
}

// dropConversations forgets every channel's conversation. It runs as the
// bridge's goroutine ends, which is the one moment that covers every way the
// bridge stops running.
func (b *Bridge) dropConversations() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.convos = nil
}

// conversations is the store this bridge's sessions share. A session asks for
// it once, at the moment it is built, and holds the pointer for as long as it
// runs; the store itself outlives every one of them and goes when the bridge
// stops.
func (b *Bridge) conversations() *conversations {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.convos == nil {
		// Unreachable as the code stands: the store is made before the
		// goroutine that opens a session, and dropped only as that goroutine
		// ends. It SAYS SO rather than absorbing it in silence
		// (iss-2609190312312041), because a later change that dropped the
		// store under a live session would otherwise leave that session
		// writing a history nobody owns, with nothing in the log and no test
		// the wiser. The session is given a store of its own and thrown away
		// with it, which is the safe direction: a stopped bridge holds no
		// conversation, and a message in flight is answered without a history
		// rather than on a nil map.
		b.log.Info("the Discord bridge answered without a conversation store; this is a bug in its lifetime",
			"bridge", bridgeName)
		return newConversations()
	}
	return b.convos
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
	// Connecting and stopped KEEP it: it is the last-connected moment, and a
	// bridge that has dropped is the one case where a person wants it. Only
	// the switch going off clears it, because then there is no bridge to have
	// last connected.
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
		// Said when the session goes, not when the next attempt is made: the
		// wait below is up to half a minute, and for the whole of it the
		// panel would otherwise go on reading "connected since" for a session
		// that is gone (iss-2609190242334438). The moment it last connected
		// is kept, which is what the panel shows while it reconnects.
		b.setState(StateConnecting, "")
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
