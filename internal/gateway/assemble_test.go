package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/Dessau/internal/config"
	"github.com/intentdriven/Dessau/internal/registry"
	"github.com/intentdriven/Dessau/internal/runtime"
	"github.com/intentdriven/Dessau/internal/stats"
)

// childModelArg is the --model value the fake child below was started with.
// It stands for the absolute path a real install uses, which must never reach
// a client.
const childModelArg = "/models/mlx-community/Qwen3-8B-4bit"

// mlxChild is a fake model server that answers the way mlx-lm 0.31.3 does,
// through the port of its builders in mlxport_test.go, including the timing
// difference the gateway has to live with: a streamed answer's headers go out
// as soon as the request is admitted and each chunk is written as it is
// generated, while an unstreamed answer's headers wait for the whole answer
// to be generated — and its generation, which writes nothing until the end,
// never learns that the client has gone.
type mlxChild struct {
	q mlxRequest
	// eventDelay is how long generating each streamed event takes, and so
	// how long the unstreamed answer takes in all: one delay per event the
	// stream would have carried.
	eventDelay time.Duration
	// keepalives is how many ": keepalive" comments the stream carries
	// before its first chunk, as the server writes them during prefill.
	keepalives int
	// cutAfter, when positive, ends the answer after that many streamed
	// events by closing the connection, which is what a model server that
	// raises in its token loop — or crashes — does: no "[DONE]". Unstreamed,
	// half the body goes out before the close.
	cutAfter int
	// endAfter, when positive, ends the answer after that many streamed
	// events by returning from the handler: the response ends cleanly, with
	// no "[DONE]". The pinned server speaks HTTP/1.0, so its stream ends at
	// the connection's close and a client reads a plain end of file, not the
	// unexpected one cutAfter gives.
	endAfter int
	// errorEvent, when set, is written as one streamed event after the first
	// chunk, and the stream then ends without "[DONE]" — or, with
	// doneAfterError, with it.
	errorEvent     string
	doneAfterError bool
	// holdHeaders, when set, never sends a status line at all until the
	// request goes away.
	holdHeaders bool

	mu     sync.Mutex
	bodies []map[string]any
	// gone is closed when a streamed generation found its client had gone
	// before the answer ended, which is the only way mlx-lm stops early.
	gone     chan struct{}
	goneOnce sync.Once
	// sent is the exact streamed body the child wrote, for the relay tests.
	sent bytes.Buffer
}

func newMLXChild(q mlxRequest) *mlxChild {
	if q.model == "" {
		q.model = childModelArg
	}
	return &mlxChild{q: q, gone: make(chan struct{})}
}

func (c *mlxChild) lastBody() map[string]any {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.bodies) == 0 {
		return nil
	}
	return c.bodies[len(c.bodies)-1]
}

func (c *mlxChild) sentBody() []byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]byte(nil), c.sent.Bytes()...)
}

func (c *mlxChild) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	c.mu.Lock()
	c.bodies = append(c.bodies, body)
	c.mu.Unlock()
	if c.holdHeaders {
		<-r.Context().Done()
		return
	}
	q := c.q
	q.chat = r.URL.Path == chatCompletionsPath

	stream, _ := body["stream"].(bool)
	if stream {
		includeUsage := false
		if opts, ok := body["stream_options"].(map[string]any); ok {
			includeUsage, _ = opts["include_usage"].(bool)
		}
		c.stream(w, r, q.run(true, includeUsage))
		return
	}

	// Unstreamed: the whole answer is generated before anything is written,
	// and nothing in that loop looks at the connection.
	events := len(q.run(true, false))
	time.Sleep(time.Duration(events) * c.eventDelay)
	answer := q.run(false, false)[0]
	h := w.Header()
	h.Set("Content-Type", "application/json")
	h.Set("Access-Control-Allow-Methods", "*")
	h.Set("Access-Control-Allow-Headers", "*")
	if c.cutAfter > 0 {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, answer[:len(answer)/2])
		c.closeConn(w)
		return
	}
	h.Set("Content-Length", strconvItoa(len(answer)))
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, answer)
}

func (c *mlxChild) stream(w http.ResponseWriter, r *http.Request, events []string) {
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Access-Control-Allow-Methods", "*")
	h.Set("Access-Control-Allow-Headers", "*")
	w.WriteHeader(http.StatusOK)
	rc := http.NewResponseController(w)
	_ = rc.Flush()

	write := func(s string) bool {
		c.mu.Lock()
		c.sent.WriteString(s)
		c.mu.Unlock()
		if _, err := io.WriteString(w, s); err != nil {
			return false
		}
		return rc.Flush() == nil
	}
	gone := func() { c.goneOnce.Do(func() { close(c.gone) }) }

	for i := 0; i < c.keepalives; i++ {
		if !write(": keepalive " + strconvItoa(i+1) + "/" + strconvItoa(c.keepalives) + "\n\n") {
			gone()
			return
		}
	}
	for i, ev := range events {
		select {
		case <-time.After(c.eventDelay):
		case <-r.Context().Done():
			// The next write would fail; the server's finally: ctx.stop().
			gone()
			return
		}
		if c.cutAfter > 0 && i == c.cutAfter {
			c.closeConn(w)
			return
		}
		if c.endAfter > 0 && i == c.endAfter {
			return
		}
		if !write("data: " + ev + "\n\n") {
			gone()
			return
		}
		if c.errorEvent != "" && i == 0 {
			write("data: " + c.errorEvent + "\n\n")
			if c.doneAfterError {
				break
			}
			c.closeConn(w)
			return
		}
	}
	write("data: [DONE]\n\n")
}

// closeConn drops the connection under a response already begun, which is
// what a server that raises out of its handler, or dies, leaves a client with.
func (c *mlxChild) closeConn(w http.ResponseWriter) {
	conn, buf, err := http.NewResponseController(w).Hijack()
	if err != nil {
		panic(err)
	}
	_ = buf.Flush()
	_ = conn.Close()
}

func strconvItoa(n int) string { return strconv.Itoa(n) }

// childPool hands every request to the one fake child.
type childPool struct{ url string }

func (p childPool) Acquire(context.Context, string) (*runtime.Upstream, func(), error) {
	return &runtime.Upstream{
		RepoID:  testChildRepo,
		BaseURL: p.url,
		// The fake child is on a loopback port; the pool hands over a
		// transport that dials its server's private socket instead.
		Transport: http.DefaultTransport,
		ModelArg:  childModelArg,
	}, func() {}, nil
}
func (childPool) Resident() []runtime.Resident         { return nil }
func (childPool) Pinned() []string                     { return nil }
func (childPool) Unload(string) error                  { return nil }
func (childPool) Footprint(string) int64               { return 0 }
func (childPool) Release(string, runtime.Caller) error { return nil }

const testChildRepo = "mlx-community/Qwen3-8B-4bit"

// childGateway is a gateway in front of one fake child. rec is nil unless
// recording is asked for.
func childGateway(t *testing.T, child *mlxChild, cfg config.Config, recording bool) (*httptest.Server, *stats.Recorder) {
	t.Helper()
	up := httptest.NewServer(child)
	t.Cleanup(up.Close)
	models := &stubModels{models: []registry.Model{{ChatTemplate: true,
		RepoID: testChildRepo, Path: childModelArg, State: registry.StateReady,
	}}}
	opts := Options{Config: cfg, Pool: childPool{url: up.URL}, Models: models}
	var rec *stats.Recorder
	if recording {
		rec = stats.New(stats.Options{})
		rec.SetEnabled(true)
		opts.Config.Statistics = true
		opts.Stats = rec
	}
	srv := httptest.NewServer(New(opts).Handler())
	t.Cleanup(srv.Close)
	return srv, rec
}

func send(t *testing.T, ctx context.Context, srv *httptest.Server, path, body string) (*http.Response, []byte, error) {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, srv.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := srv.Client().Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	out, err := io.ReadAll(resp.Body)
	return resp, out, err
}

// baseRequest is the per-request constants a fake answer repeats.
func baseRequest() mlxRequest {
	return mlxRequest{
		id:           "chatcmpl-6f1d2c3b-0a4e-4b8e-9c1d-2e3f4a5b6c7d",
		fingerprint:  "0.31.3-0.29.3-macOS-26.0-arm64-arm-64bit-applegpu_g15s",
		created:      1759500000,
		promptTokens: 2481,
		cached:       0,
	}
}

func words(state string, ws ...string) []mlxGen {
	var out []mlxGen
	for _, w := range ws {
		out = append(out, mlxGen{text: w, state: state})
	}
	return out
}

func finished(gens []mlxGen, reason string) []mlxGen {
	gens[len(gens)-1].finish = reason
	return gens
}

// goldenCases are the answers the golden test holds the assembler to: each
// is answered by the same port of the model server's builders, streamed and
// unstreamed, for both endpoints.
func goldenCases() map[string]struct {
	path string
	q    mlxRequest
} {
	type tc = struct {
		path string
		q    mlxRequest
	}
	cases := map[string]tc{}

	plain := baseRequest()
	// Characters Python escapes and Go's encoder treats specially, so the
	// golden compares escaping as well as structure: non-ASCII, a character
	// above the BMP, a quote, a backslash, a newline, and the three that Go
	// escapes for HTML.
	plain.gens = finished(words("normal", "Grüß ", "dich, ", "Alice ", "🙂", " \"<b>\" ", "& a\\b\n", "done."), "stop")
	cases["chat, plain text"] = tc{chatCompletionsPath, plain}

	length := baseRequest()
	length.cached = -1
	length.gens = finished(words("normal", "one ", "two ", "three"), "length")
	cases["chat, cut at max_tokens"] = tc{chatCompletionsPath, length}

	thinking := baseRequest()
	thinking.cached = 1024
	thinking.gens = append(words("reasoning", "Bob ", "asked ", "for ", "a sum."),
		finished(words("normal", "It ", "is ", "4."), "stop")...)
	cases["chat, reasoning then answer"] = tc{chatCompletionsPath, thinking}

	tools := baseRequest()
	tools.gens = append(words("normal", "Checking."),
		mlxGen{text: `<tool_call>{"name": "get_time", "arguments": {"city": "Zürich"}}</tool_call>`, state: "tool"},
		mlxGen{text: "", state: "normal"},
		mlxGen{text: `<tool_call>{"name": "get_weather", "arguments": {"city": "Carol's town"}}</tool_call>`, state: "tool", finish: "stop"},
	)
	tools.tools = map[string][]mlxToolCall{
		`<tool_call>{"name": "get_time", "arguments": {"city": "Zürich"}}</tool_call>`: {{
			name: "get_time", arguments: `{"city": "Zürich"}`, id: "call_0a1b2c",
		}},
		`<tool_call>{"name": "get_weather", "arguments": {"city": "Carol's town"}}</tool_call>`: {{
			name: "get_weather", arguments: `{"city": "Carol's town"}`, id: "call_3d4e5f",
		}},
	}
	cases["chat, tool calls"] = tc{chatCompletionsPath, tools}

	toolLength := baseRequest()
	toolLength.gens = append(words("normal", "Let me look."),
		mlxGen{text: `<tool_call>{"name": "get_time"`, state: "tool", finish: "length"},
	)
	toolLength.tools = map[string][]mlxToolCall{
		`<tool_call>{"name": "get_time"`: {{name: "get_time", arguments: `{}`, id: "call_trunc"}},
	}
	cases["chat, tool call cut at max_tokens"] = tc{chatCompletionsPath, toolLength}

	text := baseRequest()
	text.id = "cmpl-0b9a8c7d-1e2f-4a3b-8c4d-5e6f7a8b9c0d"
	text.gens = finished(words("normal", "Once ", "upon ", "a ", "time, ", "Bob."), "stop")
	cases["text completion"] = tc{"/v1/completions", text}

	textLength := baseRequest()
	textLength.id = "cmpl-1c2d3e4f-5a6b-4c7d-8e9f-0a1b2c3d4e5f"
	textLength.cached = -1
	textLength.gens = finished(words("normal", "and ", "then"), "length")
	cases["text completion, cut at max_tokens"] = tc{"/v1/completions", textLength}

	return cases
}

// A non-streamed request is answered with exactly the bytes the gateway would
// have relayed had the model server answered it unstreamed: the same object,
// field for field — id, object, model, created, system_fingerprint, every
// choice's index, message (role, content, reasoning, tool_calls) or text,
// finish_reason, and the usage counts with their cached-token detail — and the
// same escaping. The model server is asked for a stream to get there.
func TestANonStreamedAnswerIsAssembledToTheModelServersOwnShape(t *testing.T) {
	for name, tc := range goldenCases() {
		t.Run(name, func(t *testing.T) {
			child := newMLXChild(tc.q)
			srv, _ := childGateway(t, child, config.Default(), false)

			reqBody := `{"model":"` + testChildRepo + `","max_tokens":512,"prompt":"Once","messages":[{"role":"user","content":"hi"}]}`
			resp, got, err := send(t, context.Background(), srv, tc.path, reqBody)
			if err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("status = %d, want 200: %s", resp.StatusCode, got)
			}

			if stream, _ := child.lastBody()["stream"].(bool); !stream {
				t.Errorf("the model server was asked for an unstreamed answer; want a stream it can be told to stop")
			}

			// What a client received for this request before the change: the
			// model server's unstreamed answer, through the relay.
			q := tc.q
			q.model = childModelArg
			q.chat = tc.path == chatCompletionsPath
			unstreamed := q.run(false, false)[0]
			want := httptest.NewRecorder()
			relayRewritingModel(want, &http.Response{
				Header: http.Header{"Content-Type": []string{"application/json"}},
				Body:   io.NopCloser(strings.NewReader(unstreamed)),
			}, childModelArg, testChildRepo, relayOptions{})

			if !bytes.Equal(got, want.Body.Bytes()) {
				t.Errorf("assembled answer differs from the unstreamed one\n got: %s\nwant: %s", got, want.Body.Bytes())
			}
			if strings.Contains(string(got), childModelArg) {
				t.Errorf("the model server's own path reached the client: %s", got)
			}
			if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", ct)
			}
			if cc := resp.Header.Get("Cache-Control"); cc != "" {
				t.Errorf("Cache-Control = %q; the unstreamed answer carries none", cc)
			}
			if got := resp.Header.Get("Access-Control-Allow-Methods"); got != "*" {
				t.Errorf("Access-Control-Allow-Methods = %q; the model server's other headers are relayed as before", got)
			}
		})
	}
}

// The issue's case: an answer that takes longer to generate than the header
// wait. The wait now bounds only how long the model server takes to start
// answering, so the answer arrives whole.
func TestANonStreamedAnswerLongerThanTheHeaderWaitIsAnswered(t *testing.T) {
	q := baseRequest()
	q.gens = finished(words("normal", "a ", "long ", "answer ", "takes ", "a ", "while"), "stop")
	child := newMLXChild(q)
	child.eventDelay = 300 * time.Millisecond // seven events: 2.1 s against a 1 s wait

	cfg := config.Default()
	cfg.UpstreamHeaderTimeoutSec = 1
	srv, _ := childGateway(t, child, cfg, false)

	resp, got, err := send(t, context.Background(), srv, chatCompletionsPath,
		`{"model":"`+testChildRepo+`","messages":[{"role":"user","content":"hi"}]}`)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", resp.StatusCode, got)
	}
	var ans struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(got, &ans); err != nil || len(ans.Choices) != 1 {
		t.Fatalf("not one answer: %v: %s", err, got)
	}
	if c := ans.Choices[0]; c.Message.Content != "a long answer takes a while" || c.FinishReason != "stop" {
		t.Errorf("answer = %+v, want the whole answer, finished", c)
	}
}

// A client that hangs up while its unstreamed answer is being assembled stops
// the generation: the gateway's request to the model server goes with it, and
// a model server writing a stream finds out at its next chunk. Unstreamed, the
// pinned server would have generated to max_tokens for nobody.
func TestAClientThatHangsUpStopsTheGeneration(t *testing.T) {
	q := baseRequest()
	var gens []mlxGen
	for i := 0; i < 100; i++ {
		gens = append(gens, mlxGen{text: "word ", state: "normal"})
	}
	q.gens = finished(gens, "length")
	child := newMLXChild(q)
	child.eventDelay = 50 * time.Millisecond // five seconds of answer

	srv, _ := childGateway(t, child, config.Default(), false)

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	if resp, _, err := send(t, ctx, srv, chatCompletionsPath,
		`{"model":"`+testChildRepo+`","messages":[{"role":"user","content":"hi"}]}`); err == nil {
		t.Fatalf("the request was answered %d before the client hung up; the test needs a longer answer", resp.StatusCode)
	}

	select {
	case <-child.gone:
	case <-time.After(2 * time.Second):
		t.Fatal("the model server was still generating two seconds after the client hung up")
	}
}

// A stream that ends without its "[DONE]" — the model server raised in its
// token loop, or died — is not an answer, and the client is told so with an
// error status rather than handed half an object under a 200.
func TestAnAnswerCutShortIsAnErrorNotHalfAnObject(t *testing.T) {
	for name, set := range map[string]func(*mlxChild){
		"the connection closes":                  func(c *mlxChild) { c.cutAfter = 2 },
		"the stream ends cleanly without [DONE]": func(c *mlxChild) { c.endAfter = 2 },
		"an error event":                         func(c *mlxChild) { c.errorEvent = `{"error": "RuntimeError: [metal] out of memory"}` },
		// Shaped like a completion event, so that only its error field tells
		// it apart, and followed by a "[DONE]" as if nothing had happened.
		"an error event, then [DONE]": func(c *mlxChild) {
			c.errorEvent = `{"choices": [], "error": {"message": "RuntimeError: [metal] out of memory"}}`
			c.doneAfterError = true
		},
	} {
		t.Run(name, func(t *testing.T) {
			q := baseRequest()
			q.gens = finished(words("normal", "one ", "two ", "three ", "four"), "stop")
			child := newMLXChild(q)
			set(child)
			srv, rec := childGateway(t, child, config.Default(), true)

			resp, got, err := send(t, context.Background(), srv, chatCompletionsPath,
				`{"model":"`+testChildRepo+`","messages":[{"role":"user","content":"hi"}]}`)
			if err != nil {
				t.Fatalf("the client's read failed (%v); it should have been given an error answer", err)
			}
			if resp.StatusCode != http.StatusBadGateway {
				t.Errorf("status = %d, want 502: %s", resp.StatusCode, got)
			}
			var e struct {
				Error struct {
					Message string `json:"message"`
				} `json:"error"`
			}
			if err := json.Unmarshal(got, &e); err != nil || e.Error.Message == "" {
				t.Errorf("the body is not an error object (%v): %s", err, got)
			}
			if strings.Contains(string(got), "out of memory") {
				t.Errorf("the model server's own error text reached the client: %s", got)
			}
			if r := onlyRecord(t, rec); r.Class != stats.ClassUnreachable {
				t.Errorf("recorded as %q, want %q", r.Class, stats.ClassUnreachable)
			}
		})
	}
}

// An answer larger than the gateway relays in one piece is refused with an
// error, never cut short under a 200, and is never held whole in memory.
func TestAnAssembledAnswerIsBoundedByTheResponseCap(t *testing.T) {
	if testing.Short() {
		t.Skip("streams more than the 64 MiB response cap")
	}
	piece := strings.Repeat("x", 1<<20)
	q := baseRequest()
	var gens []mlxGen
	for i := 0; i <= maxResponseBody>>20; i++ {
		gens = append(gens, mlxGen{text: piece, state: "normal"})
	}
	q.gens = finished(gens, "length")
	child := newMLXChild(q)
	srv, _ := childGateway(t, child, config.Default(), false)

	resp, got, err := send(t, context.Background(), srv, chatCompletionsPath,
		`{"model":"`+testChildRepo+`","messages":[{"role":"user","content":"hi"}]}`)
	if err != nil {
		t.Fatalf("the client's read failed (%v); it should have been given an error answer", err)
	}
	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("status = %d (%d bytes), want 502", resp.StatusCode, len(got))
	}
	if !strings.Contains(string(got), "stream") {
		t.Errorf("the refusal does not tell the client it can ask for the answer streamed: %s", got)
	}
}

// What the model server is asked for. An unstreamed request becomes a stream
// with the counts asked for, and a stream_options of the client's — which
// OpenAI does not accept without stream, and which the model server ignores
// unstreamed — is replaced rather than merged, so a malformed one cannot make
// the server raise. A streamed request, and one asking for logprobs, which the
// pinned server writes only into an unstreamed answer, go as they came.
func TestWhatTheModelServerIsAskedFor(t *testing.T) {
	cases := []struct {
		name, body  string
		wantStream  any
		wantOptions any
	}{
		{"stream absent", `{"messages":[{"role":"user","content":"hi"}]}`, true, map[string]any{"include_usage": true}},
		{"stream false", `{"stream":false,"messages":[{"role":"user","content":"hi"}]}`, true, map[string]any{"include_usage": true}},
		{"stream false with options of the client's", `{"stream":false,"stream_options":{"include_usage":false,"x":1},"messages":[{"role":"user","content":"hi"}]}`,
			true, map[string]any{"include_usage": true}},
		{"stream false with malformed options", `{"stream":false,"stream_options":"yes","messages":[{"role":"user","content":"hi"}]}`,
			true, map[string]any{"include_usage": true}},
		{"stream true", `{"stream":true,"messages":[{"role":"user","content":"hi"}]}`, true, nil},
		// The pinned server refuses a stream that is not a bool
		// (validate_model_parameters: _validate("stream", bool)), so a falsy
		// one is not an unstreamed request the gateway may answer: it goes as
		// it came, to be refused there.
		{"stream 0", `{"stream":0,"messages":[{"role":"user","content":"hi"}]}`, 0, nil},
		{"stream empty string", `{"stream":"","messages":[{"role":"user","content":"hi"}]}`, "", nil},
		{"stream empty list", `{"stream":[],"messages":[{"role":"user","content":"hi"}]}`, []any{}, nil},
		{"logprobs", `{"logprobs":true,"messages":[{"role":"user","content":"hi"}]}`, nil, nil},
		{"top_logprobs", `{"top_logprobs":3,"messages":[{"role":"user","content":"hi"}]}`, nil, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			q := baseRequest()
			q.gens = finished(words("normal", "ok"), "stop")
			child := newMLXChild(q)
			srv, _ := childGateway(t, child, config.Default(), false)
			body := `{"model":"` + testChildRepo + `",` + strings.TrimPrefix(c.body, "{")
			if _, _, err := send(t, context.Background(), srv, chatCompletionsPath, body); err != nil {
				t.Fatal(err)
			}
			got := child.lastBody()
			if s := got["stream"]; !equalJSON(s, c.wantStream) {
				t.Errorf("stream = %v, want %v", s, c.wantStream)
			}
			if o := got["stream_options"]; !equalJSON(o, c.wantOptions) {
				t.Errorf("stream_options = %v, want %v", o, c.wantOptions)
			}
		})
	}
}

func equalJSON(a, b any) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return bytes.Equal(x, y)
}

// A streamed request is relayed exactly as it was before any of this: the
// model server is asked what the client asked, and the client receives the
// relay of what the model server wrote, byte for byte.
func TestAStreamedRequestIsRelayedUnchanged(t *testing.T) {
	q := baseRequest()
	q.gens = finished(words("normal", "Hello ", "Bob."), "stop")
	child := newMLXChild(q)
	child.keepalives = 2
	srv, _ := childGateway(t, child, config.Default(), false)

	resp, got, err := send(t, context.Background(), srv, chatCompletionsPath,
		`{"model":"`+testChildRepo+`","stream":true,"stream_options":{"include_usage":true},"messages":[{"role":"user","content":"hi"}]}`)
	if err != nil {
		t.Fatal(err)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("Content-Type = %q, want a stream", ct)
	}
	want := httptest.NewRecorder()
	streamRewriteSSE(want, bytes.NewReader(child.sentBody()), childModelArg, testChildRepo, relayOptions{})
	if !bytes.Equal(got, want.Body.Bytes()) {
		t.Errorf("the streamed answer changed\n got: %q\nwant: %q", got, want.Body.Bytes())
	}
	sent := child.lastBody()
	if len(sent) != 4 || sent["stream"] != true || !equalJSON(sent["stream_options"], map[string]any{"include_usage": true}) {
		t.Errorf("the model server was asked %v, want the client's request with only its model rewritten", sent)
	}
}

// The header wait now fires only when the model server has not started
// answering, and the 504 says that, and names the setting where it is: in
// config.json, which is the only place it is. Only a request that asked for
// logprobs, which is answered in one piece, is told the wait covered the whole
// answer.
func TestTheHeaderWaitSaysWhatItMeasured(t *testing.T) {
	for _, c := range []struct {
		name, fields string
		want, wrong  []string
	}{
		{"unstreamed", ``,
			[]string{"did not start answering within 1s"}, []string{"logprobs", "finish"}},
		{"streamed", `"stream":true,`,
			[]string{"did not start answering within 1s"}, []string{"logprobs", "finish"}},
		{"a stream that is not a bool", `"stream":0,`,
			[]string{"did not start answering within 1s"}, []string{"logprobs", "finish"}},
		{"logprobs", `"logprobs":true,`,
			[]string{"did not finish answering within 1s", "logprobs"}, nil},
		{"top_logprobs", `"top_logprobs":2,`,
			[]string{"did not finish answering within 1s", "logprobs"}, nil},
	} {
		t.Run(c.name, func(t *testing.T) {
			child := newMLXChild(baseRequest())
			child.holdHeaders = true
			cfg := config.Default()
			cfg.UpstreamHeaderTimeoutSec = 1
			srv, rec := childGateway(t, child, cfg, true)

			resp, got, err := send(t, context.Background(), srv, chatCompletionsPath,
				`{"model":"`+testChildRepo+`",`+c.fields+`"messages":[{"role":"user","content":"hi"}]}`)
			if err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != http.StatusGatewayTimeout {
				t.Fatalf("status = %d, want 504", resp.StatusCode)
			}
			msg := string(got)
			for _, want := range append(c.want, "upstream_header_timeout_sec", "config.json") {
				if !strings.Contains(msg, want) {
					t.Errorf("the 504 does not say %q: %s", want, msg)
				}
			}
			for _, wrong := range append(c.wrong, "Settings", "reading the prompt") {
				if strings.Contains(msg, wrong) {
					t.Errorf("the 504 says %q: %s", wrong, msg)
				}
			}
			if r := onlyRecord(t, rec); r.Class != stats.ClassUnreachable {
				t.Errorf("recorded as %q, want %q", r.Class, stats.ClassUnreachable)
			}
		})
	}
}

// An assembled answer is recorded as what it was: an unstreamed request the
// model server answered in full, with the counts from its usage event.
func TestAnAssembledAnswerIsRecordedWithItsCounts(t *testing.T) {
	q := baseRequest()
	q.gens = finished(words("normal", "one ", "two ", "three"), "stop")
	child := newMLXChild(q)
	srv, rec := childGateway(t, child, config.Default(), true)

	resp, got, err := send(t, context.Background(), srv, chatCompletionsPath,
		`{"model":"`+testChildRepo+`","messages":[{"role":"user","content":"hi"}]}`)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("status %v, err %v: %s", resp, err, got)
	}
	r := onlyRecord(t, rec)
	if r.Class != stats.ClassOK || r.Streamed {
		t.Errorf("recorded as %q, streamed %v; want ok and unstreamed, which is what the client asked for", r.Class, r.Streamed)
	}
	if r.PromptTokens != 2481 || r.CompletionTokens != 3 {
		t.Errorf("recorded %d and %d tokens, want 2481 and 3", r.PromptTokens, r.CompletionTokens)
	}
	if strings.Contains(string(got), `"choices":[]`) {
		t.Errorf("the counts-only event reached the client: %s", got)
	}
}

// The cap counts the answer, not the stream's framing: every chunk repeats
// the request's constants and the role, which an unstreamed answer carries
// once. A long answer of short chunks must not be refused for the size of a
// stream it was never going to be sent as.
func TestTheAssemblyCapCountsTheAnswerNotTheFraming(t *testing.T) {
	q := baseRequest()
	q.chat = true
	q.model = childModelArg
	var gens []mlxGen
	for i := 0; i < 1000; i++ {
		gens = append(gens, mlxGen{text: "x", state: "normal"})
	}
	q.gens = finished(gens, "length")
	var stream strings.Builder
	for _, ev := range q.run(true, true) {
		stream.WriteString("data:" + ev + "\n\n") // no space after the colon: also SSE
	}
	stream.WriteString("data: [DONE]\n\n")
	if stream.Len() < 100_000 {
		t.Fatalf("the stream is %d bytes; the test needs framing far beyond the limit", stream.Len())
	}

	got, err := assembleAnswer(strings.NewReader(stream.String()), 4096)
	if err != nil {
		t.Fatalf("an answer of 1,000 characters was refused under a 4 KiB cap: %v", err)
	}
	var ans struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(got, &ans); err != nil || len(ans.Choices) != 1 || len(ans.Choices[0].Message.Content) != 1000 {
		t.Fatalf("assembled %s (%v), want one message of 1,000 characters", got, err)
	}

	if _, err := assembleAnswer(strings.NewReader(stream.String()), 512); err != errAnswerTooLarge {
		t.Errorf("an answer past the cap gave %v, want errAnswerTooLarge", err)
	}
}
