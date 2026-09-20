package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/intentdriven/Dessau/internal/config"
	"github.com/intentdriven/Dessau/internal/mlxtest"
	"github.com/intentdriven/Dessau/internal/registry"
	"github.com/intentdriven/Dessau/internal/runtime"
	"github.com/intentdriven/Dessau/internal/stats"
)

// askGateway is a recording gateway a test can call Ask on directly, over a
// fake model server that streams an answer.
func askGateway(t *testing.T, reply string) (*Gateway, *stats.Recorder, *mlxtest.Server) {
	t.Helper()
	const modelPath = "/models/" + testModelID
	fake := mlxtest.Start(mlxtest.Options{ModelArg: modelPath, Reply: reply})
	t.Cleanup(fake.Close)
	models := &stubModels{models: []registry.Model{{
		RepoID: testModelID, Path: modelPath, State: registry.StateReady, ContextLength: 131072,
	}}}
	rec := stats.New(stats.Options{})
	rec.SetEnabled(true)
	cfg := config.Default()
	cfg.Statistics = true
	cfg.Models = map[string]config.ModelSettings{testModelID: {ServedContext: 65536}}
	g := New(Options{
		Config: cfg,
		Pool:   &stubPool{srv: fake},
		Models: models,
		Stats:  rec,
	})
	return g, rec, fake
}

// askBody is the shape a bridge sends: the ordinary chat-completions request.
func askBody(t *testing.T, text string) []byte {
	t.Helper()
	b, err := json.Marshal(map[string]any{
		"model":    testModelID,
		"messages": []map[string]string{{"role": "user", "content": text}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// A request made in process is recorded exactly as one over the network is,
// with the one difference the bridge exists to record: its source.
//
// adr-2609181004167097 condition 4: the statistics store gains one fixed
// class, the source, and nothing a platform supplied.
func TestAskIsRecordedAsABridgedRequest(t *testing.T) {
	g, rec, _ := askGateway(t, "the answer")

	var got strings.Builder
	err := g.Ask(context.Background(), AskRequest{
		Model:   testModelID,
		Body:    askBody(t, "hello"),
		Source:  stats.SourceBridge,
		OnEvent: func(payload []byte) { got.WriteString(payload2text(payload)) },
	})
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if !strings.Contains(got.String(), "the answer") {
		t.Errorf("the callback saw %q, want the model's answer", got.String())
	}

	rr := onlyRecord(t, rec)
	if rr.Source != stats.SourceBridge {
		t.Errorf("source = %q, want %q", rr.Source, stats.SourceBridge)
	}
	if rr.Model != testModelID {
		t.Errorf("model = %q", rr.Model)
	}
	if rr.Class != stats.ClassOK {
		t.Errorf("class = %q", rr.Class)
	}
	if !rr.Streamed {
		t.Error("a bridged request is always streamed, so the record should say so")
	}
	if rr.CompletionTokens == 0 || rr.PromptTokens == 0 {
		t.Errorf("token counts = %d in, %d out; Ask asks the model server for them",
			rr.PromptTokens, rr.CompletionTokens)
	}
	if rr.ServedContext != 65536 || rr.DeclaredContext != 131072 {
		t.Errorf("windows = %d served, %d declared: a bridged request is judged like every other",
			rr.ServedContext, rr.DeclaredContext)
	}
}

// A request over the network is still recorded as one. The source is a fixed
// class with two values, and the one that was always true has to stay true.
func TestAnHTTPRequestIsStillRecordedAsHTTP(t *testing.T) {
	srv, rec := measuredGateway(t, 0, 0)
	status, _ := completion(t, srv, `{"model":"`+testModelID+`","messages":[{"role":"user","content":"hi"}]}`)
	if status != 200 {
		t.Fatalf("status %d", status)
	}
	if got := onlyRecord(t, rec).Source; got != stats.SourceHTTP {
		t.Errorf("source = %q, want %q", got, stats.SourceHTTP)
	}
}

// The load-bearing rewrite happens on this path too: mlx-lm reads "model" as
// an instruction to LOAD, so the name the caller used is replaced by the
// backend's own --model value on the way out, and replaced back on the way in.
// A bridge that skipped it would send the model server a repo id to download.
func TestAskRewritesTheModelFieldBothWays(t *testing.T) {
	g, _, fake := askGateway(t, "ok")
	var seen string
	err := g.Ask(context.Background(), AskRequest{
		Model:  testModelID,
		Body:   askBody(t, "hi"),
		Source: stats.SourceBridge,
		OnEvent: func(payload []byte) {
			var ev struct {
				Model string `json:"model"`
			}
			if json.Unmarshal(payload, &ev) == nil && ev.Model != "" {
				seen = ev.Model
			}
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := fake.LastModelField(); got != "/models/"+testModelID {
		t.Errorf("the model server was asked for %q, want the backend's own --model value", got)
	}
	if seen != testModelID {
		t.Errorf("the caller saw model %q, want the name it asked for — the backend's path must "+
			"never reach it", seen)
	}
}

// A refusal carries two texts: the detailed one for the operator's log, and
// the generic one that may be repeated to whoever asked. The informative texts
// describe this Mac, and a bridge posts its answer onto somebody else's
// platform.
func TestAskRefusalKeepsTheDetailedTextOffThePlatform(t *testing.T) {
	g, rec, _ := askGateway(t, "ok")
	err := g.Ask(context.Background(), AskRequest{
		Model:  "a-model-this-mac-has-never-heard-of",
		Body:   askBody(t, "hi"),
		Source: stats.SourceBridge,
	})
	if err == nil {
		t.Fatal("a request for an unknown model was served")
	}
	var askErr *AskError
	if !errors.As(err, &askErr) {
		t.Fatalf("Ask returned %T, want an *AskError carrying both texts", err)
	}
	if askErr.Public() != genericRefusal {
		t.Errorf("the text that may travel is %q, want the generic refusal", askErr.Public())
	}
	if askErr.Error() == askErr.Public() {
		t.Error("the operator's text and the platform's text are the same, so nothing was withheld")
	}
	if got := onlyRecord(t, rec).Source; got != stats.SourceBridge {
		t.Errorf("a refused bridged request is recorded with source %q", got)
	}
}

// A conversation too large for the window is refused here rather than being
// sent, and the refusal that travels is still the generic one — the detailed
// text tells the reader to raise a setting on somebody else's Mac.
func TestAskRefusesAConversationOverTheServedWindow(t *testing.T) {
	g, _, _ := askGateway(t, "ok")
	err := g.Ask(context.Background(), AskRequest{
		Model:  testModelID,
		Body:   askBody(t, strings.Repeat("x", 4*100_000)),
		Source: stats.SourceBridge,
	})
	if err == nil {
		t.Fatal("a conversation far over the window was served")
	}
	var askErr *AskError
	if !errors.As(err, &askErr) {
		t.Fatalf("Ask returned %T", err)
	}
	// The one refusal whoever asked can act on travels as a sentence of its
	// own — and still names no figure, because the window is a setting on
	// this Mac (iss-2609190106563320).
	if askErr.Public() != tooLongRefusal {
		t.Errorf("the text that travels is %q, want the too-long sentence", askErr.Public())
	}
	for _, leak := range []string{"Settings", "65,536", "served context"} {
		if strings.Contains(askErr.Public(), leak) {
			t.Errorf("the text that travels carries %q, which describes this Mac", leak)
		}
	}
	if !strings.Contains(askErr.Error(), "more than the") {
		t.Errorf("the operator's text is %q, want the size refusal", askErr.Error())
	}
}

// Ask takes no HTTP request, so the observation cannot read a cancellation off
// one. A caller that goes away must still be recorded as having done so.
func TestAskRecordsACancelledRequest(t *testing.T) {
	g, rec, _ := askGateway(t, "ok")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := g.Ask(ctx, AskRequest{Model: testModelID, Body: askBody(t, "hi"), Source: stats.SourceBridge})
	if err == nil {
		t.Fatal("a cancelled request was served")
	}
	if got := onlyRecord(t, rec).Class; got != stats.ClassCancelled {
		t.Errorf("class = %q, want %q", got, stats.ClassCancelled)
	}
}

// payload2text pulls the generated text out of one streamed event, the way a
// bridge does. It lives in the test file on purpose: internal/gateway is not
// a reader of generated content (adr-2609061610102325), and a test may
// compose and read whatever conversation it needs.
func payload2text(payload []byte) string {
	var ev struct {
		Choices []struct {
			Delta struct {
				Content string `json:"content"`
			} `json:"delta"`
		} `json:"choices"`
	}
	if json.Unmarshal(payload, &ev) != nil || len(ev.Choices) == 0 {
		return ""
	}
	return ev.Choices[0].Delta.Content
}

// A pool refusal's text is the operator's and never the caller's.
//
// The texts name the resident memory budget in bytes, and a LaunchError's is
// the child process's verbatim with absolute local paths in it. An in-process
// caller is relaying to somebody else's platform and can drive a refusal at
// the rate it can be sent messages, so it is handed a fixed class and the
// operator keeps the text — the same rule handleCompletions follows for an
// unentitled network client (iss-2609190106273104).
func TestAskHandsBackAClassAndKeepsThePoolsTextForTheOperator(t *testing.T) {
	const modelPath = "/models/" + testModelID
	models := &stubModels{models: []registry.Model{{
		RepoID: testModelID, Path: modelPath, State: registry.StateReady, ContextLength: 131072,
	}}}
	rec := stats.New(stats.Options{})
	rec.SetEnabled(true)
	cfg := config.Default()
	cfg.Statistics = true

	var logged strings.Builder
	g := New(Options{
		Config: cfg,
		Pool: &refusingPool{err: &runtime.LaunchError{
			Err: errors.New("/Users/someone/Library/Application Support/Dessau/venv/bin/python: no such file"),
		}},
		Models: models,
		Stats:  rec,
		Log:    slog.New(slog.NewTextHandler(&logged, &slog.HandlerOptions{Level: slog.LevelDebug})),
	})

	err := g.Ask(context.Background(), AskRequest{
		Model: testModelID, Body: askBody(t, "hi"), Source: stats.SourceBridge,
	})
	var askErr *AskError
	if !errors.As(err, &askErr) {
		t.Fatalf("Ask returned %T", err)
	}
	if strings.Contains(askErr.Error(), "/Users/") {
		t.Errorf("the caller was handed a local filesystem path: %q", askErr.Error())
	}
	if askErr.Public() != genericRefusal {
		t.Errorf("the text that travels is %q", askErr.Public())
	}
	if askErr.Class() == "" {
		t.Error("the caller was given no class to record the refusal by")
	}
	// And the operator keeps what the caller no longer gets.
	if !strings.Contains(logged.String(), "/Users/") {
		t.Errorf("the operator's log does not carry the reason:\n%s", logged.String())
	}
}

// refusingPool is a pool that refuses every acquisition with one error.
type refusingPool struct {
	stubPool
	err error
}

func (p *refusingPool) Acquire(ctx context.Context, repoID string) (*runtime.Upstream, func(), error) {
	return nil, nil, p.err
}
