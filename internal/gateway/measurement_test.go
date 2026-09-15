package gateway

import (
	"encoding/json"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/intentdriven/Gropius/internal/config"
	"github.com/intentdriven/Gropius/internal/mlxtest"
	"github.com/intentdriven/Gropius/internal/registry"
	"github.com/intentdriven/Gropius/internal/runtime"
	"github.com/intentdriven/Gropius/internal/stats"
)

// measuredGateway is a recording gateway over a model that declares a
// window, with the pool reporting what a test says about in-flight requests
// and the server's footprint.
func measuredGateway(t *testing.T, inFlight int, footprint int64) (*httptest.Server, *stats.Recorder) {
	t.Helper()
	const modelPath = "/models/" + testModelID
	fake := mlxtest.Start(mlxtest.Options{ModelArg: modelPath, Reply: "GROPIUS OK"})
	t.Cleanup(fake.Close)
	models := &stubModels{models: []registry.Model{{
		RepoID: testModelID, Path: modelPath, State: registry.StateReady, ContextLength: 131072,
	}}}
	pool := &stubPool{srv: fake, footprint: footprint, waits: runtime.AcquireStats{InFlight: inFlight}}
	rec := stats.New(stats.Options{})
	rec.SetEnabled(true)
	cfg := config.Default()
	cfg.Statistics = true
	cfg.Models = map[string]config.ModelSettings{testModelID: {ServedContext: 65536}}
	g := New(Options{Config: cfg, Pool: pool, Models: models, Stats: rec})
	srv := httptest.NewServer(g.Handler())
	t.Cleanup(srv.Close)
	return srv, rec
}

// A completing request's record carries the two windows it was judged
// against and how many requests the model already had when it was admitted.
func TestARecordCarriesTheWindowsAndTheInFlightAtAdmission(t *testing.T) {
	srv, rec := measuredGateway(t, 2, 0)
	status, _ := completion(t, srv, `{"model":"mlx-community/Qwen3-8B-4bit","messages":[{"role":"user","content":"hi"}]}`)
	if status != 200 {
		t.Fatalf("status %d", status)
	}
	got := onlyRecord(t, rec)
	if got.DeclaredContext != 131072 || got.ServedContext != 65536 {
		t.Errorf("windows = %d declared, %d served", got.DeclaredContext, got.ServedContext)
	}
	if got.InFlight != 2 {
		t.Errorf("in_flight = %d, want the model's count at admission", got.InFlight)
	}
	if got.EstimatedPromptTokens <= 0 || got.RequestedTokens < got.EstimatedPromptTokens {
		t.Errorf("estimated %d, requested %d", got.EstimatedPromptTokens, got.RequestedTokens)
	}
}

// A request tipped over the window by the answer it asked for is refused on
// the judged figure, and the record carries that figure, so the dashboard's
// band and the refusal agree.
func TestARequestTippedOverByMaxTokensIsRecordedOnTheJudgedFigure(t *testing.T) {
	srv, rec := measuredGateway(t, 0, 0)
	status, _ := completion(t, srv, `{"model":"mlx-community/Qwen3-8B-4bit","messages":[{"role":"user","content":"hi"}],"max_tokens":65536}`)
	if status == 200 {
		t.Fatal("a request over the window by its max_tokens was served")
	}
	got := onlyRecord(t, rec)
	if got.RequestedTokens <= 65536 || got.EstimatedPromptTokens > 100 {
		t.Errorf("requested %d, estimated %d; the judged figure is the estimate plus max_tokens", got.RequestedTokens, got.EstimatedPromptTokens)
	}
}

// A request refused for its size still carries the gateway's estimate, so
// the refusal says how big the prompt was.
func TestARefusedRequestStillCarriesItsEstimatedSize(t *testing.T) {
	srv, rec := measuredGateway(t, 0, 0)
	// Over the served window of 65,536: the estimate is a token per four bytes.
	body := `{"model":"mlx-community/Qwen3-8B-4bit","messages":[{"role":"user","content":"` + strings.Repeat("x", 4*100_000) + `"}]}`
	status, _ := completion(t, srv, body)
	if status == 200 {
		t.Fatal("a prompt far over the window was served")
	}
	got := onlyRecord(t, rec)
	if got.Class != stats.ClassClientError {
		t.Errorf("class = %q", got.Class)
	}
	if got.EstimatedPromptTokens < 100_000 || got.ServedContext != 65536 {
		t.Errorf("a refused request carries estimate %d against served %d", got.EstimatedPromptTokens, got.ServedContext)
	}
}

// A client that sets a sampling parameter is recorded as having overridden
// it, by name; no number the client sent reaches the record.
func TestOverridesAreNamedAndNeverValued(t *testing.T) {
	srv, rec := measuredGateway(t, 0, 0)
	status, _ := completion(t, srv, `{"model":"mlx-community/Qwen3-8B-4bit","messages":[{"role":"user","content":"hi"}],"temperature":0.123456,"max_completion_tokens":777,"top_p":0.5}`)
	if status != 200 {
		t.Fatalf("status %d", status)
	}
	got := onlyRecord(t, rec)
	if !reflect.DeepEqual(got.Overrides, []string{"temperature", "top_p", "max_tokens"}) {
		t.Errorf("overrides = %v", got.Overrides)
	}
	raw, _ := json.Marshal(got)
	for _, value := range []string{"0.123456", "777", "0.5"} {
		if strings.Contains(string(raw), value) {
			t.Errorf("the record carries the client's value %s:\n%s", value, raw)
		}
	}
}

// A completing request carries the model server's latest sampled footprint,
// named as a sample.
func TestARecordCarriesTheLatestFootprintSample(t *testing.T) {
	srv, rec := measuredGateway(t, 0, 48<<30)
	status, _ := completion(t, srv, `{"model":"mlx-community/Qwen3-8B-4bit","messages":[{"role":"user","content":"hi"}]}`)
	if status != 200 {
		t.Fatalf("status %d", status)
	}
	if got := onlyRecord(t, rec); got.FootprintBytes != 48<<30 {
		t.Errorf("footprint_bytes = %d", got.FootprintBytes)
	}
}
