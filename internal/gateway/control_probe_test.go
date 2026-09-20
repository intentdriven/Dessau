package gateway

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/intentdriven/Dessau/internal/config"
	"github.com/intentdriven/Dessau/internal/mlxtest"
	"github.com/intentdriven/Dessau/internal/registry"
)

func postModel(t *testing.T, url, path, model string) (*http.Response, map[string]any) {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"model": model})
	resp, err := http.Post(url+path, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp, out
}

// The models list carries the measured window and its bound beside the
// declared and the served one, only while the measurement is current: a stale
// one is not published.
func TestModelsListCarriesTheMeasuredContext(t *testing.T) {
	fake := mlxtest.Start(mlxtest.Options{ModelArg: "/m"})
	defer fake.Close()
	models := &stubModels{models: []registry.Model{
		{RepoID: "org/plain", State: registry.StateReady, ContextLength: 131072},
		{RepoID: "org/current", State: registry.StateReady, ContextLength: 131072,
			Measured: &registry.Measurement{Window: 91000, Bound: registry.BoundPrefillDeadline, Runtime: "0.31.3"}},
		{RepoID: "org/stale", State: registry.StateReady, ContextLength: 131072,
			Measured: &registry.Measurement{Window: 91000, Bound: registry.BoundModel, Runtime: "0.31.3", Stale: registry.StaleRuntime}},
	}}
	g := New(Options{Config: config.Default(), Pool: &stubPool{srv: fake}, Models: models})
	srv := httptest.NewServer(g.Handler())
	defer srv.Close()
	byID := map[string]map[string]any{}
	for _, e := range allModelEntries(t, srv) {
		byID[e["id"].(string)] = e
	}
	if e := byID["org/plain"]; e["measured_context"] != nil || e["measured_bound"] != nil {
		t.Errorf("an unmeasured model carries a measurement: %v", e)
	}
	if e := byID["org/stale"]; e["measured_context"] != nil {
		t.Errorf("a stale measurement is published: %v", e)
	}
	e := byID["org/current"]
	if e["measured_context"] != float64(91000) || e["measured_bound"] != registry.BoundPrefillDeadline {
		t.Errorf("entry = %v, want measured_context 91000 with the prefill_deadline bound", e)
	}
	if e["context_length"] != float64(131072) || e["served_context"] != float64(131072) {
		t.Errorf("the declared and served windows moved: %v", e)
	}
}

// Measure now queues a probe and the snapshot shows the queue; adopting
// writes the served window and answers under the settings lock.
func TestMeasureNowQueuesAndAdoptWritesTheServedWindow(t *testing.T) {
	srv, a := newTestControlApp(t, config.Default())
	if err := a.Registry.Put(registry.Model{
		RepoID: "org/m", Path: a.Paths.ModelDir("org/m"), State: registry.StateReady, Bytes: 1, ContextLength: 131072,
	}); err != nil {
		t.Fatal(err)
	}
	resp, out := postModel(t, srv.URL, "/api/models/measure", "org/m")
	if resp.StatusCode != http.StatusAccepted || out["status"] != "queued" {
		t.Fatalf("measure = %d %v", resp.StatusCode, out)
	}
	state, _ := getJSON(t, srv, "/api/state")
	if q, _ := state["probe_queue"].([]any); len(q) != 1 || q[0] != "org/m" {
		t.Errorf("probe_queue = %v", state["probe_queue"])
	}
	if _, ok := state["idle_jobs"]; !ok {
		t.Error("the snapshot carries no idle_jobs")
	}
	if resp, _ := postModel(t, srv.URL, "/api/models/measure", "org/absent"); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("measuring an absent model = %d, want 400", resp.StatusCode)
	}

	if resp, _ := postModel(t, srv.URL, "/api/models/adopt", "org/m"); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("adopting nothing = %d, want 400", resp.StatusCode)
	}
	if err := a.Registry.SetMeasurement("org/m", &registry.Measurement{
		Window: 65536, Bound: registry.BoundModel, At: 1,
		Runtime: "0.31.3", BudgetBytes: a.Pool.MemoryBudget(), DecodeConcurrency: a.Pool.DecodeConcurrency(), ServedContext: 131072,
	}); err != nil {
		t.Fatal(err)
	}
	resp, out = postModel(t, srv.URL, "/api/models/adopt", "org/m")
	if resp.StatusCode != http.StatusOK || out["status"] != "adopted" {
		t.Fatalf("adopt = %d %v", resp.StatusCode, out)
	}
	if got := a.Config().ServedContextSetting("org/m", 131072); got != 65536 {
		t.Errorf("served window after adopting = %d", got)
	}
}

// A save that names neither probe setting leaves both alone: the settings
// handler decodes into a copy of what is in force.
func TestASaveThatNamesNeitherProbeSettingLeavesBothAlone(t *testing.T) {
	c := config.Default()
	c.ContextProbe = true
	c.IdleThresholdSec = 120
	srv, a := newTestControlApp(t, c)
	resp, err := http.Post(srv.URL+"/api/settings", "application/json", bytes.NewReader([]byte(`{"log_level":"detailed"}`)))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("save = %d", resp.StatusCode)
	}
	if got := a.Config(); !got.ContextProbe || got.IdleThresholdSec != 120 {
		t.Errorf("a save naming neither moved them: probe=%v threshold=%d", got.ContextProbe, got.IdleThresholdSec)
	}
}

// The snapshot carries the figure an unset idle threshold resolves to, for the
// same reason the grace pair is in there: the panel's idle-threshold field is
// blank for the default, and the figure that stands for belongs to the server
// rather than to a third copy of config.DefaultIdleThresholdSec written into
// the panel's markup (iss-2609190146152463).
func TestStateCarriesTheFigureAnUnsetIdleThresholdResolvesTo(t *testing.T) {
	_, srv := newBudgetControl(t, config.Default(), 128*gb, nil)

	if got := stateOf(t, srv).Defaults.IdleThresholdSec; got != config.DefaultIdleThresholdSec {
		t.Errorf("defaults.idle_threshold_sec = %d, want the server's own default %d",
			got, config.DefaultIdleThresholdSec)
	}
}
