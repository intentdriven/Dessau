package discord

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// fakeDiscord stands in for Discord's gateway and its REST API.
//
// The whole point of it is that Discord is never contacted: the session's
// state machine, the mention filter, the editor's throttle and cut, the
// conversation bound and the two slash commands are all exercised against
// this. It speaks the parts of the protocol the bridge uses — Hello, Identify,
// Ready, heartbeats, Resume, MESSAGE_CREATE and INTERACTION_CREATE — and
// records everything the bridge sends back.
type fakeDiscord struct {
	t   *testing.T
	srv *httptest.Server

	// heartbeatMS is what Hello names. Tests that care about heartbeats set it
	// small; everything else leaves it long enough never to fire.
	heartbeatMS int64
	// closeAfterIdentify, when set, closes the connection with that code as
	// soon as the bridge has identified — which is how Discord refuses a token
	// or a set of intents.
	closeAfterIdentify websocket.StatusCode

	// frames carries every gateway frame the bridge sent.
	frames chan frame
	// calls carries every REST call the bridge made.
	calls chan restCall
	// sessions is signalled once per accepted connection.
	sessions chan struct{}

	mu         sync.Mutex
	conn       *websocket.Conn
	seq        int64
	editHdr    http.Header
	handshakes int
}

// restCall is one REST request the bridge made, as the fake saw it.
type restCall struct {
	Method string
	Path   string
	Auth   string
	Body   map[string]any
	Raw    string
}

func newFakeDiscord(t *testing.T) *fakeDiscord {
	t.Helper()
	f := &fakeDiscord{
		t:           t,
		heartbeatMS: 45_000,
		frames:      make(chan frame, 64),
		calls:       make(chan restCall, 256),
		sessions:    make(chan struct{}, 8),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/", f.handleREST)
	mux.HandleFunc("/", f.handleGateway)
	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeDiscord) gatewayURL() string {
	return "ws://" + strings.TrimPrefix(f.srv.URL, "http://") + "/?v=10&encoding=json"
}

func (f *fakeDiscord) apiBase() string { return f.srv.URL + "/api" }

// options is a Bridge wired to this fake, with the clock a test can move.
func (f *fakeDiscord) options(now func() time.Time) Options {
	return Options{
		GatewayURL:    f.gatewayURL(),
		APIBase:       f.apiBase(),
		HTTPClient:    f.srv.Client(),
		Log:           slog.New(slog.NewTextHandler(io.Discard, nil)),
		Now:           now,
		ChatModels:    func() []string { return []string{firstModel, secondModel} },
		ServedContext: func(string) int64 { return 8192 },
	}
}

func (f *fakeDiscord) handleGateway(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		return
	}
	ctx := context.Background()
	f.mu.Lock()
	f.conn = conn
	f.handshakes++
	f.mu.Unlock()
	select {
	case f.sessions <- struct{}{}:
	default:
	}

	hello, _ := json.Marshal(map[string]any{
		"op": opHello,
		"d":  map[string]any{"heartbeat_interval": f.heartbeatMS},
	})
	if conn.Write(ctx, websocket.MessageText, hello) != nil {
		return
	}
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		var fr frame
		if json.Unmarshal(data, &fr) != nil {
			continue
		}
		select {
		case f.frames <- fr:
		default:
		}
		switch fr.Op {
		case opHeartbeat:
			ack, _ := json.Marshal(map[string]any{"op": opHeartbeatACK})
			_ = conn.Write(ctx, websocket.MessageText, ack)
		case opIdentify:
			if f.closeAfterIdentify != 0 {
				_ = conn.Close(f.closeAfterIdentify, "refused")
				return
			}
			f.dispatch("READY", map[string]any{
				"session_id":         "sess-1",
				"resume_gateway_url": "ws://" + strings.TrimPrefix(f.srv.URL, "http://"),
				"user":               map[string]any{"id": botUserID},
				"application":        map[string]any{"id": appID},
			})
		case opResume:
			f.dispatch("RESUMED", map[string]any{})
		}
	}
}

// The chat models the fake server offers, the first being the default a
// channel starts on. TWO of them, because `/model` picking a model and the
// next message being answered by it — the acceptance criterion — cannot be
// exercised against a server with only one (iss-2609190242018424).
const (
	firstModel  = "mlx-community/Qwen3-8B-4bit"
	secondModel = "mlx-community/Llama-3.2-3B-Instruct-4bit"
)

// The identifiers the fake uses for itself, so a test can write a mention.
const (
	botUserID = "111111111111111111"
	appID     = "222222222222222222"
	humanID   = "333333333333333333"
	channelID = "444444444444444444"
	guildID   = "555555555555555555"
)

// dispatch sends one event down the live connection, with the next sequence
// number.
func (f *fakeDiscord) dispatch(kind string, data map[string]any) {
	f.mu.Lock()
	conn := f.conn
	f.seq++
	seq := f.seq
	f.mu.Unlock()
	if conn == nil {
		return
	}
	payload, err := json.Marshal(map[string]any{"op": opDispatch, "t": kind, "s": seq, "d": data})
	if err != nil {
		f.t.Fatal(err)
	}
	_ = conn.Write(context.Background(), websocket.MessageText, payload)
}

// drop closes the live connection the way a sleeping Mac does: without a
// close frame the bridge could read a reason from.
func (f *fakeDiscord) drop() {
	f.mu.Lock()
	conn := f.conn
	f.conn = nil
	f.mu.Unlock()
	if conn != nil {
		_ = conn.CloseNow()
	}
}

// message sends a MESSAGE_CREATE. A guild id makes it a channel message; a
// mention makes it one for us.
func (f *fakeDiscord) message(text string, guild bool, mentionsUs bool) {
	d := map[string]any{
		"id":         "900000000000000001",
		"channel_id": channelID,
		"content":    text,
		"type":       0,
		"author":     map[string]any{"id": humanID, "bot": false},
	}
	if guild {
		d["guild_id"] = guildID
	}
	if mentionsUs {
		d["mentions"] = []any{map[string]any{"id": botUserID}}
	} else {
		d["mentions"] = []any{}
	}
	f.dispatch("MESSAGE_CREATE", d)
}

// command sends an INTERACTION_CREATE for one of the bridge's slash commands.
func (f *fakeDiscord) command(name string, options map[string]string) {
	var opts []any
	for k, v := range options {
		opts = append(opts, map[string]any{"name": k, "value": v})
	}
	f.dispatch("INTERACTION_CREATE", map[string]any{
		"id":         "800000000000000001",
		"token":      "interaction-token",
		"type":       interactionCommand,
		"channel_id": channelID,
		"data":       map[string]any{"name": name, "options": opts},
	})
}

func (f *fakeDiscord) handleREST(w http.ResponseWriter, r *http.Request) {
	raw, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	var body map[string]any
	_ = json.Unmarshal(raw, &body)
	call := restCall{
		Method: r.Method,
		Path:   strings.TrimPrefix(r.URL.Path, "/api"),
		Auth:   r.Header.Get("Authorization"),
		Body:   body,
		Raw:    string(raw),
	}
	select {
	case f.calls <- call:
	default:
	}
	f.mu.Lock()
	hdr := f.editHdr
	f.mu.Unlock()
	for k, vs := range hdr {
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"id":"700000000000000001"}`))
}

// setRateLimitHeaders makes every REST answer carry the headers a bucket that
// has just been emptied would.
func (f *fakeDiscord) setRateLimitHeaders(h http.Header) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.editHdr = h
}

func (f *fakeDiscord) dials() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.handshakes
}

// ── waiting ──────────────────────────────────────────────

// waitFrame waits for the next frame with the given opcode.
func (f *fakeDiscord) waitFrame(op int) frame {
	f.t.Helper()
	deadline := time.After(5 * time.Second)
	for {
		select {
		case fr := <-f.frames:
			if fr.Op == op {
				return fr
			}
		case <-deadline:
			f.t.Fatalf("no frame with op %d arrived", op)
		}
	}
}

// waitCall waits for the next REST call whose path contains want.
func (f *fakeDiscord) waitCall(method, want string) restCall {
	f.t.Helper()
	deadline := time.After(5 * time.Second)
	for {
		select {
		case c := <-f.calls:
			if (method == "" || c.Method == method) && strings.Contains(c.Path, want) {
				return c
			}
		case <-deadline:
			f.t.Fatalf("no %s call to a path containing %q arrived", method, want)
		}
	}
}

// waitSession waits for a connection to be accepted.
func (f *fakeDiscord) waitSession() {
	f.t.Helper()
	select {
	case <-f.sessions:
	case <-time.After(5 * time.Second):
		f.t.Fatal("the bridge never connected")
	}
}

// waitState waits for the bridge to report one of the given states, and
// returns it with its reason.
func waitState(t *testing.T, b *Bridge, want ...string) (string, string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		state, _, reason := b.State()
		for _, w := range want {
			if state == w {
				return state, reason
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	state, _, reason := b.State()
	t.Fatalf("the bridge is %q (%s), want one of %v", state, reason, want)
	return "", ""
}

// quiet is a moment long enough that something the bridge was going to do
// would have happened. It is used only to assert that nothing did.
const quiet = 300 * time.Millisecond

// drainCalls empties what has already arrived, so a later assertion about
// what the bridge does next is not answered by what it did before.
func (f *fakeDiscord) drainCalls() {
	for {
		select {
		case <-f.calls:
		default:
			return
		}
	}
}

// noCall fails if any REST call arrives within a quiet moment.
func (f *fakeDiscord) noCall() {
	f.t.Helper()
	select {
	case c := <-f.calls:
		f.t.Fatalf("the bridge made a %s call to %s, and should have made none", c.Method, c.Path)
	case <-time.After(quiet):
	}
}
