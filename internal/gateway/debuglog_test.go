package gateway

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/intentdriven/Dessau/internal/app"
	"github.com/intentdriven/Dessau/internal/config"
	"github.com/intentdriven/Dessau/internal/registry"
	"github.com/intentdriven/Dessau/internal/runtime"
)

// Debug logging for one model (itd-2609062346072707) is an action on the
// control plane, not a setting: arming marks a model in the pool so that its
// next launch runs at the model server's debug level, and the snapshot says
// which models are armed and which running process was launched that way.
// These tests hold the route, the snapshot, the refusal for a model that
// keeps no transcript, and that arming writes nothing to the settings.

// newDebugLogControl is newTestControlApp with one ready model in the
// registry, under the configuration given — which is where a test says
// which models keep no transcript.
func newDebugLogControl(t *testing.T, cfg config.Config) (*httptest.Server, *app.App) {
	t.Helper()
	paths := config.NewPaths(t.TempDir())
	a, err := app.New(app.Options{Paths: paths, Config: cfg})
	if err != nil {
		t.Fatalf("app.New: %v", err)
	}
	t.Cleanup(func() { a.Close() })
	if err := a.Registry.Put(registry.Model{
		RepoID: "org/keeper", Path: a.Paths.ModelDir("org/keeper"),
		Bytes: 1 << 20, State: registry.StateReady, Progress: 100,
	}); err != nil {
		t.Fatal(err)
	}
	ctrl := &Control{App: a}
	mux := http.NewServeMux()
	ctrl.Routes(mux)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, a
}

// refusalMessage reads the message of a control-plane refusal and closes the body.
func refusalMessage(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return errorMessage(t, string(body))
}

// The per-model state is served: the armed set on the snapshot, read from
// the pool the way the pinned set is, and the running fact on each resident
// entry — two fields, because armed and running are different runs.
func TestTheSnapshotCarriesTheDebugState(t *testing.T) {
	srv, a := newDebugLogControl(t, config.Default())

	if got := fetchState(t, srv).DebugArmed; len(got) != 0 {
		t.Fatalf("state.debug_armed = %v before anything was armed, want nothing", got)
	}

	resp := postJSON(t, srv, "/api/models/debug-log", `{"model":"org/keeper","armed":true}`)
	var armed struct {
		Status string `json:"status"`
		Model  string `json:"model"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&armed); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || armed.Status != "armed" || armed.Model != "org/keeper" {
		t.Fatalf("arming answered %d %+v, want 200 {armed org/keeper}", resp.StatusCode, armed)
	}

	st := fetchState(t, srv)
	if !reflect.DeepEqual(st.DebugArmed, []string{"org/keeper"}) {
		t.Errorf("state.debug_armed = %v, want the armed model — the panel draws the armed pill from this", st.DebugArmed)
	}
	// Read from the pool, which is what spends the mark at launch.
	if got := a.Pool.DebugArmed(); !reflect.DeepEqual(st.DebugArmed, got) {
		t.Errorf("state.debug_armed = %v but the pool holds %v", st.DebugArmed, got)
	}

	resp = postJSON(t, srv, "/api/models/debug-log", `{"model":"ORG/Keeper","armed":false}`)
	if err := json.NewDecoder(resp.Body).Decode(&armed); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || armed.Status != "disarmed" {
		t.Fatalf("disarming answered %d %+v, want 200 disarmed", resp.StatusCode, armed)
	}
	if got := fetchState(t, srv).DebugArmed; len(got) != 0 {
		t.Errorf("state.debug_armed = %v after disarming under another spelling, want nothing", got)
	}

	// The running fact rides each resident entry under its own key, so the
	// panel can move the pill from armed to running without a reload. No
	// model server runs in this test; the field's presence on the wire is
	// what is held here, and internal/runtime holds that it is set at launch.
	field, ok := reflect.TypeOf(runtime.Resident{}).FieldByName("DebugLog")
	if !ok || !strings.HasPrefix(field.Tag.Get("json"), "debug_log") {
		t.Errorf("runtime.Resident carries no debug_log field for the panel to read")
	}
}

// A model carrying the transcript exception refuses the arm with the reason:
// the exception means no prompts on disk, not "not in this one file"
// (itd-2609091715089488). The exception is read from the configuration in
// force, folded — the operator's spelling in Settings and the registry's
// need not agree — and disarming such a model is not refused: there is
// nothing to keep from being written.
func TestAnExceptedModelRefusesTheDebugArm(t *testing.T) {
	cfg := config.Default()
	cfg.Models = map[string]config.ModelSettings{"ORG/Keeper": {NoTranscript: true}}
	srv, a := newDebugLogControl(t, cfg)

	resp := postJSON(t, srv, "/api/models/debug-log", `{"model":"org/keeper","armed":true}`)
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("status = %d arming an excepted model, want 409", resp.StatusCode)
	}
	msg := refusalMessage(t, resp)
	for _, want := range []string{"no transcript", "refuse"} {
		if !strings.Contains(msg, want) {
			t.Errorf("the refusal %q does not say %q", msg, want)
		}
	}
	if got := a.Pool.DebugArmed(); len(got) != 0 {
		t.Errorf("the pool holds %v after a refused arm, want nothing", got)
	}

	resp = postJSON(t, srv, "/api/models/debug-log", `{"model":"org/keeper","armed":false}`)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d disarming an excepted model, want 200: there is nothing to refuse", resp.StatusCode)
	}
}

// The exception takes effect on the arm the moment it is saved: a model
// excepted through the settings route is refused on the next arm, with no
// restart and nothing wired between the two.
func TestExceptingAModelInSettingsRefusesItsNextDebugArm(t *testing.T) {
	srv, _ := newDebugLogControl(t, config.Default())
	resp := postJSON(t, srv, "/api/settings",
		`{"host":"0.0.0.0","port":11535,"api_key":"","decode_concurrency":1,"idle_timeout_sec":0,`+
			`"models":{"org/keeper":{"no_transcript":true}}}`)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d saving the exception, want 200", resp.StatusCode)
	}
	resp = postJSON(t, srv, "/api/models/debug-log", `{"model":"org/keeper","armed":true}`)
	resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("status = %d arming a model just excepted in Settings, want 409", resp.StatusCode)
	}
}

// The mark is an action and not a setting: arming it writes nothing to the
// configuration in force and nothing to config.json, in either direction.
func TestTheDebugArmWritesNoSettings(t *testing.T) {
	srv, a := newDebugLogControl(t, config.Default())
	cfg := a.Config()
	cfg.Statistics = true
	cfg.Models = pinnedModels("org/keeper")
	if err := a.SetConfig(cfg); err != nil {
		t.Fatal(err)
	}
	before := a.Config()
	onDisk, err := os.ReadFile(a.Paths.Config)
	if err != nil {
		t.Fatal(err)
	}

	for _, body := range []string{
		`{"model":"org/keeper","armed":true}`,
		`{"model":"org/keeper","armed":false}`,
	} {
		resp := postJSON(t, srv, "/api/models/debug-log", body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s: status = %d, want 200", body, resp.StatusCode)
		}
		if got := a.Config(); !reflect.DeepEqual(got, before) {
			t.Errorf("%s: the configuration in force changed: %+v, want %+v", body, got, before)
		}
		if got, err := os.ReadFile(a.Paths.Config); err != nil || string(got) != string(onDisk) {
			t.Errorf("%s: config.json changed (err=%v)", body, err)
		}
	}
}

// The route takes a model the registry holds and a body that says which way
// the mark goes; anything else is refused before the pool is touched.
func TestTheDebugArmRefusesWhatItCannotAct(t *testing.T) {
	srv, a := newDebugLogControl(t, config.Default())
	cases := []struct {
		body   string
		status int
		want   string
	}{
		{`{"model":"org/never-downloaded","armed":true}`, http.StatusNotFound, "not found"},
		{`{"armed":true}`, http.StatusBadRequest, `"model"`},
		{`{"model":"org/keeper"}`, http.StatusBadRequest, `"armed"`},
		{`not json`, http.StatusBadRequest, ""},
		{`{"model":"` + strings.Repeat("a", config.MaxConfigBytes) + `","armed":true}`, http.StatusBadRequest, "too large"},
	}
	for _, c := range cases {
		resp := postJSON(t, srv, "/api/models/debug-log", c.body)
		if resp.StatusCode != c.status {
			t.Errorf("%.60s: status = %d, want %d", c.body, resp.StatusCode, c.status)
		}
		if msg := refusalMessage(t, resp); !strings.Contains(msg, c.want) {
			t.Errorf("%.60s: message = %q, want it to say %q", c.body, msg, c.want)
		}
	}
	if got := a.Pool.DebugArmed(); len(got) != 0 {
		t.Errorf("the pool holds %v after refused requests, want nothing", got)
	}
}
