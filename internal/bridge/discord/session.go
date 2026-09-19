package discord

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"net"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
)

// defaultGatewayURL is Discord's gateway at API version 10, JSON encoded.
// Version and encoding are in the query string because that is where Discord
// reads them; nothing negotiates either.
const defaultGatewayURL = "wss://gateway.discord.gg/?v=10&encoding=json"

// The gateway opcodes this bridge speaks. Everything else that arrives is
// ignored, which is the protocol's own instruction for a client that does not
// implement an opcode.
const (
	opDispatch       = 0
	opHeartbeat      = 1
	opIdentify       = 2
	opResume         = 6
	opReconnect      = 7
	opInvalidSession = 9
	opHello          = 10
	opHeartbeatACK   = 11
)

// intents is what this bridge asks Discord to send it: guild messages and
// direct messages, and NEITHER privileged intent.
//
// GUILD_MESSAGES (1<<9) delivers a message in a channel the bot is in, and
// without MESSAGE_CONTENT its content arrives empty UNLESS the bot is
// mentioned — which is exactly and only the case this bridge answers
// (itd-2609180959397172). DIRECT_MESSAGES (1<<12) delivers a direct message,
// whose content a bot always receives. Asking for MESSAGE_CONTENT would make
// the bridge a reader of every message in every channel it is invited to, and
// it is never asked for.
const intents = 1<<9 | 1<<12

// readLimit bounds one gateway message. The connection is to somebody else's
// servers and the frames arrive before anything of this package looks at them,
// so the ceiling is the library's rather than this code's to enforce: without
// it a hostile or broken peer can make the process allocate without bound.
// A READY payload is the largest legitimate message, and for a bot in many
// guilds a megabyte is not enough for it — so the limit is four, and Identify
// asks for a small large_threshold besides, which is what bounds what READY
// carries. Exceeding it closes the connection, and that is not weather: see
// the fatal case in outcome (iss-2609190106555510).
const readLimit = 4 << 20

// largeThreshold is how many members a guild must have before Discord leaves
// its offline members out of READY. Discord's own minimum, and the smallest
// READY it will send.
const largeThreshold = 50

// resumeState is what a dropped session needs to be picked up again rather
// than started afresh: Discord's own id for it, the URL it said to resume at,
// and the last sequence number seen.
//
// The sequence is behind a mutex because it is the one field two goroutines
// touch: the read loop advances it on every dispatch and the heartbeat
// goroutine sends it on every beat. Unsynchronised, a beat can carry a number
// the session has already passed — and a resume built from a stale one asks
// Discord for events twice or skips them (iss-2609190054030656).
type resumeState struct {
	mu        sync.Mutex
	sessionID string
	resumeURL string
	seq       int64
	// botID and appID are who Discord said we are. They arrive in READY and
	// NOT in RESUMED, so they have to outlive the connection that learned
	// them: without that, the first resume left the bot's own id empty, every
	// mention check answered false, and the bridge went on answering direct
	// messages while silently ignoring every mention in every channel — with
	// nothing in the log and the panel reading connected
	// (iss-2609190106402401). A resume is the ordinary path after a Wi-Fi
	// blip or a lid close, so that was the steady state rather than an edge.
	botID string
	appID string
}

func (r *resumeState) canResume() bool { return r.sessionID != "" && r.resumeURL != "" }

// advance records the sequence of a frame that carried one.
func (r *resumeState) advance(seq int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq = seq
}

// sequence is the last one seen.
func (r *resumeState) sequence() int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.seq
}

// clear forgets the session, so the next attempt identifies afresh. It is a
// method rather than an assignment of a fresh value because the mutex above
// makes resumeState a type that is not copied.
//
// The identity is kept. It is a fact about the bot rather than about the
// session, it does not change between connections under one token, and
// keeping it means a fresh Identify that has not yet had its READY still
// knows which mentions are its own.
func (r *resumeState) clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessionID, r.resumeURL, r.seq = "", "", 0
}

// identify records who Discord said we are.
func (r *resumeState) identify(botID, appID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.botID, r.appID = botID, appID
}

// who is the bot's own user id and its application id.
func (r *resumeState) who() (botID, appID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.botID, r.appID
}

// sessionOutcome is how one gateway session ended.
type sessionOutcome struct {
	// fatal, when set, stops the bridge and is shown on the panel. It is set
	// only for a close code that says retrying cannot help.
	fatal string
	// resumable says the resume state may be reused for the next attempt.
	resumable bool
	// connected says the session reached READY or RESUMED, so the reconnect
	// backoff starts over.
	connected bool
	err       error
}

// runSession opens one gateway connection and runs it until it ends.
//
// The shape is Discord's: Hello arrives first with the heartbeat interval,
// then the client either Identifies or Resumes, then dispatches arrive and are
// answered. Heartbeats run on a goroutine of their own so that a long read —
// or a long answer — cannot make the connection look dead to Discord.
func (b *Bridge) runSession(ctx context.Context, token string, resume *resumeState) sessionOutcome {
	url := b.opts.GatewayURL
	resuming := resume.canResume()
	if resuming {
		url = resume.resumeURL
	}
	dialCtx, cancelDial := context.WithTimeout(ctx, dialTimeout)
	conn, _, err := websocket.Dial(dialCtx, url, &websocket.DialOptions{HTTPClient: b.opts.HTTPClient})
	cancelDial()
	if err != nil {
		return sessionOutcome{resumable: resuming, err: fmt.Errorf("dialling the Discord gateway: %w", err)}
	}
	conn.SetReadLimit(readLimit)
	defer conn.CloseNow()

	s := &session{
		bridge: b,
		conn:   conn,
		token:  token,
		resume: resume,
		rest:   newREST(b.opts.APIBase, token, b.opts.HTTPClient, b.log),
		convos: b.conversations(),
	}
	defer s.stopWork()
	return s.run(ctx, resuming)
}

// dialTimeout bounds the handshake. A dial that hangs would otherwise hold the
// bridge in "connecting" forever with nothing to say about it.
const dialTimeout = 30 * time.Second

// session is one live gateway connection and the workers answering on it.
//
// It is built per connection and thrown away with it. The conversations are
// NOT: they are the bridge's, borrowed here, so a channel keeps its history
// and its model across a drop and the resume that follows. What ends them is
// the bridge stopping, which is the bound itd-2609180959397172 asks for.
type session struct {
	bridge *Bridge
	conn   *websocket.Conn
	token  string
	resume *resumeState
	rest   *rest
	convos *conversations

	// acked is cleared when a heartbeat is sent and set when Discord
	// acknowledges it. A heartbeat sent into an unacknowledged one means the
	// connection is a zombie and has to be torn down and resumed, which is the
	// protocol's own rule.
	acked atomic.Bool

	// work bounds what one session may have running at once. Everything that
	// answers a stranger — a completion, an interaction — takes a place here,
	// so a channel full of people cannot make this process grow a goroutine
	// per message.
	work     chan func()
	workOnce sync.Once
	workWG   sync.WaitGroup
}

// answerWorkers is how many bridged answers may be in flight at once, and
// queueDepth how many may be waiting. Both are small on purpose: a request
// holds a model server slot, and the pool is what actually decides how many
// can run — queueing more here would only mean a person waiting longer for an
// answer that was going to be refused anyway. Beyond the queue a message is
// dropped with a line in the log, because the alternative is a goroutine per
// message from a channel anyone may post in.
const (
	answerWorkers = 2
	queueDepth    = 16
)

func (s *session) startWork() {
	s.workOnce.Do(func() {
		s.work = make(chan func(), queueDepth)
		for range answerWorkers {
			s.workWG.Add(1)
			go func() {
				defer s.workWG.Done()
				for job := range s.work {
					job()
				}
			}()
		}
	})
}

func (s *session) stopWork() {
	if s.work == nil {
		return
	}
	// Closed only here, and only after the read loop has returned: submit is
	// called from the read loop alone, so there is nothing left that could
	// send into a closed channel.
	close(s.work)
	s.workWG.Wait()
}

// submit queues a job, reporting whether there was room. A full queue is a
// refusal, not a wait: the read loop must keep reading or heartbeats and close
// frames stop being handled.
func (s *session) submit(job func()) bool {
	select {
	case s.work <- job:
		return true
	default:
		return false
	}
}

// frame is one gateway message. The payload is left encoded so that an
// opcode this bridge does not implement costs nothing to skip.
type frame struct {
	Op   int             `json:"op"`
	Data json.RawMessage `json:"d"`
	Seq  *int64          `json:"s"`
	Type string          `json:"t"`
}

func (s *session) run(ctx context.Context, resuming bool) sessionOutcome {
	// Hello first, and within a bound: a gateway that accepts the connection
	// and then says nothing would otherwise hold the bridge open forever.
	helloCtx, cancelHello := context.WithTimeout(ctx, dialTimeout)
	typ, data, err := s.conn.Read(helloCtx)
	cancelHello()
	if err != nil {
		return s.outcome(resuming, false, err)
	}
	var hello frame
	if typ != websocket.MessageText || json.Unmarshal(data, &hello) != nil || hello.Op != opHello {
		return s.outcome(resuming, false, errors.New("the gateway did not open with Hello"))
	}
	var helloData struct {
		HeartbeatInterval int64 `json:"heartbeat_interval"`
	}
	if err := json.Unmarshal(hello.Data, &helloData); err != nil || helloData.HeartbeatInterval <= 0 {
		return s.outcome(resuming, false, errors.New("the gateway's Hello carried no heartbeat interval"))
	}
	interval := heartbeatInterval(helloData.HeartbeatInterval)

	if resuming {
		err = s.send(ctx, opResume, map[string]any{
			"token": s.token, "session_id": s.resume.sessionID, "seq": s.resume.sequence(),
		})
	} else {
		err = s.send(ctx, opIdentify, map[string]any{
			"token":           s.token,
			"intents":         intents,
			"large_threshold": largeThreshold,
			"properties": map[string]string{
				"os": "macOS", "browser": "Gropius", "device": "Gropius",
			},
		})
	}
	if err != nil {
		return s.outcome(resuming, false, err)
	}

	hbCtx, stopHeartbeat := context.WithCancel(ctx)
	defer stopHeartbeat()
	s.acked.Store(true)
	go s.heartbeat(hbCtx, interval)

	s.startWork()
	return s.readLoop(ctx, resuming)
}

// The range the figure in Hello is believed within. Discord's own is about 41
// seconds.
const (
	minHeartbeatIntervalMS = 1_000
	maxHeartbeatIntervalMS = 5 * 60 * 1_000
)

// heartbeatInterval turns the milliseconds Hello named into the interval the
// bridge beats on, bounded at both ends.
//
// BOUNDED AS THE INTEGER IT ARRIVES AS, before it becomes a duration
// (iss-2609190105084881). The figure comes off the network, and multiplying a
// large one by time.Millisecond overflows int64 nanoseconds into a NEGATIVE
// duration — which a ceiling check alone waves through, and which then
// reaches the jitter's Int64N and panics, taking the process down from a
// goroutine the operator never started. The floor is the other half: a Hello
// naming one millisecond passes every check and has the heartbeat goroutine
// spin on the connection.
func heartbeatInterval(ms int64) time.Duration {
	switch {
	case ms < minHeartbeatIntervalMS:
		ms = minHeartbeatIntervalMS
	case ms > maxHeartbeatIntervalMS:
		ms = maxHeartbeatIntervalMS
	}
	return time.Duration(ms) * time.Millisecond
}

func (s *session) readLoop(ctx context.Context, resuming bool) sessionOutcome {
	connected := false
	for {
		_, data, err := s.conn.Read(ctx)
		if err != nil {
			return s.outcome(resuming || connected, connected, err)
		}
		var f frame
		if json.Unmarshal(data, &f) != nil {
			continue // not a frame this bridge can read; the protocol says skip it
		}
		if f.Seq != nil {
			s.resume.advance(*f.Seq)
		}
		switch f.Op {
		case opHeartbeat:
			// Discord asking for one now, out of band.
			if err := s.sendHeartbeat(ctx); err != nil {
				return s.outcome(true, connected, err)
			}
		case opHeartbeatACK:
			s.acked.Store(true)
		case opReconnect:
			return sessionOutcome{resumable: true, connected: connected,
				err: errors.New("the gateway asked us to reconnect")}
		case opInvalidSession:
			var resumable bool
			_ = json.Unmarshal(f.Data, &resumable)
			return sessionOutcome{resumable: resumable, connected: connected,
				err: errors.New("the gateway invalidated the session")}
		case opDispatch:
			if s.dispatch(ctx, f) {
				connected = true
			}
		}
	}
}

// dispatch handles one event, reporting whether it established the session.
func (s *session) dispatch(ctx context.Context, f frame) bool {
	switch f.Type {
	case "READY":
		var ready struct {
			SessionID        string `json:"session_id"`
			ResumeGatewayURL string `json:"resume_gateway_url"`
			User             struct {
				ID string `json:"id"`
			} `json:"user"`
			Application struct {
				ID string `json:"id"`
			} `json:"application"`
		}
		if json.Unmarshal(f.Data, &ready) != nil {
			return false
		}
		s.resume.identify(ready.User.ID, ready.Application.ID)
		s.resume.sessionID = ready.SessionID
		s.resume.resumeURL = resumeURL(ready.ResumeGatewayURL, s.bridge.opts.GatewayURL)
		s.bridge.setState(StateConnected, "")
		s.bridge.log.Info("the Discord bridge connected", "bridge", bridgeName)
		// Registered once per start, as the spec has it: a bulk overwrite is
		// idempotent, so a restart replaces the pair rather than adding to it.
		if _, appID := s.resume.who(); appID != "" {
			s.submit(func() { s.registerCommands(ctx) })
		}
		return true
	case "RESUMED":
		s.bridge.setState(StateConnected, "")
		return true
	case "MESSAGE_CREATE":
		s.onMessage(ctx, f.Data)
	case "INTERACTION_CREATE":
		s.onInteraction(ctx, f.Data)
	}
	return false
}

// resumeURL turns what READY said into the URL a resume dials.
//
// Discord sends a bare "wss://host" with no query, and the version and the
// encoding have to be put back or the resumed connection speaks a different
// protocol from the one that was identified.
//
// A RESUME MAY NOT DOWNGRADE THE TRANSPORT. The value arrives over the
// network from the other end, so it is somewhere this connection could be
// talked into going: a plaintext URL from a gateway that was reached over TLS
// would move the whole session, token and all, onto a socket nobody is
// verifying. It is accepted as wss, or as ws only when the gateway this bridge
// was pointed at is itself ws — which is a test or a local proxy the operator
// configured, and is already no more plaintext than what they asked for.
// Anything else is discarded and the next attempt identifies afresh.
func resumeURL(raw, base string) string {
	got, err := url.Parse(raw)
	if err != nil || got.Host == "" {
		return ""
	}
	want, err := url.Parse(base)
	if err != nil {
		return ""
	}
	switch {
	case got.Scheme == "wss":
	case got.Scheme == "ws" && want.Scheme == "ws":
	default:
		return ""
	}
	// AND IT MAY NOT NAME ANOTHER HOST. This is the one field in the protocol
	// that redirects the credential: the next attempt dials what it names and
	// sends a Resume carrying the bot token (iss-2609190106410918). The peer
	// that would have to supply it is the gateway this bridge verified on the
	// way in, which makes this hardening rather than a hole — and it costs a
	// comparison.
	if !sameHost(got.Host, want.Host) {
		return ""
	}
	if got.RawQuery != "" {
		return raw
	}
	return raw + "/?v=10&encoding=json"
}

// sameHost reports whether a resume may go to this host: the one the bridge
// was pointed at, or a host under Discord's own domains — which is what
// Discord's own resume URLs name, and what the default gateway is a host of.
func sameHost(got, want string) bool {
	if strings.EqualFold(got, want) {
		return true
	}
	name := got
	if h, _, err := net.SplitHostPort(got); err == nil {
		name = h
	}
	name = strings.ToLower(strings.TrimSuffix(name, "."))
	for _, domain := range []string{"discord.gg", "discord.com", "discordapp.com"} {
		if name == domain || strings.HasSuffix(name, "."+domain) {
			return true
		}
	}
	return false
}

// outcome names how a session ended, reading the WebSocket close code for the
// two refusals that must stop the bridge rather than be retried.
//
// 4004 is the token: it is wrong, revoked or regenerated, and reconnecting
// with it will be refused for as long as it is done. 4014 is the intents: the
// bot has been configured in the portal in a way that forbids what was asked
// for. Both are the operator's to fix, so both stop the bridge and put the
// reason on the panel (adr-2609181004167097 condition 5's sibling: the product
// says what is true, where the person can see it). The others in the fatal
// range say the same kind of thing about the connection's shape.
func (s *session) outcome(resumable, connected bool, err error) sessionOutcome {
	switch websocket.CloseStatus(err) {
	case 4004:
		return sessionOutcome{fatal: "Discord refused the bot token — paste a fresh one in Settings"}
	case 4014:
		return sessionOutcome{fatal: "Discord refused the intents this bridge asks for — check the bot's settings in Discord's developer portal"}
	case 4010, 4011, 4012, 4013:
		return sessionOutcome{fatal: fmt.Sprintf("Discord refused the connection (close code %d)", websocket.CloseStatus(err))}
	case websocket.StatusMessageTooBig:
		// Retrying cannot help: the next READY is the same size and the limit
		// is this build's. It stops, with something on the panel, rather than
		// reconnecting forever with one Debug line to show for it
		// (iss-2609190106555510).
		return sessionOutcome{fatal: "Discord sent more than this bridge will read in one message — the bot is in too many servers for this build"}
	case 4007, 4009:
		// The sequence or the session is stale: reconnect, but afresh.
		return sessionOutcome{resumable: false, connected: connected, err: err}
	}
	return sessionOutcome{resumable: resumable, connected: connected, err: err}
}

// heartbeat keeps the session alive, and tears it down when Discord stops
// acknowledging.
//
// The first beat is jittered across the interval, which is Discord's own
// instruction: every bot reconnecting after an outage would otherwise beat in
// lockstep.
func (s *session) heartbeat(ctx context.Context, interval time.Duration) {
	first := time.Duration(rand.Int64N(int64(interval)))
	timer := time.NewTimer(first)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		if !s.acked.Swap(false) {
			// The last beat was never acknowledged: the connection is a
			// zombie. Closing it ends the read loop, which resumes.
			_ = s.conn.Close(websocket.StatusServiceRestart, "heartbeat not acknowledged")
			return
		}
		if err := s.sendHeartbeat(ctx); err != nil {
			return
		}
		timer.Reset(interval)
	}
}

func (s *session) sendHeartbeat(ctx context.Context) error {
	return s.send(ctx, opHeartbeat, s.resume.sequence())
}

// writeTimeout bounds one gateway write. Without it a stalled connection would
// block the heartbeat goroutine on a write that never completes, and the
// zombie check above would never run.
const writeTimeout = 30 * time.Second

func (s *session) send(ctx context.Context, op int, data any) error {
	payload, err := json.Marshal(map[string]any{"op": op, "d": data})
	if err != nil {
		return err
	}
	wctx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()
	return s.conn.Write(wctx, websocket.MessageText, payload)
}

// bridgeName is what this bridge is called in a log line. One constant, so the
// line the no-content test walks and the line an operator greps for are the
// same word.
const bridgeName = "discord"
