package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/Dessau/internal/config"
	"github.com/intentdriven/Dessau/internal/hub"
	"github.com/intentdriven/Dessau/internal/registry"
)

const (
	onDisk = "1111111111111111111111111111111111111111"
	newer  = "2222222222222222222222222222222222222222"

	hConfig  = "c0c0c0c0c0c0c0c0c0c0c0c0c0c0c0c0c0c0c0c0"
	hConfig2 = "c1c1c1c1c1c1c1c1c1c1c1c1c1c1c1c1c1c1c1c1"
	hWeights = "0e0e0e0e0e0e0e0e0e0e0e0e0e0e0e0e0e0e0e0e0e0e0e0e0e0e0e0e0e0e0e0e"
	hWeight2 = "0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f"
	hReadme  = "a0a0a0a0a0a0a0a0a0a0a0a0a0a0a0a0a0a0a0a0"
	hReadme2 = "a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1"
)

// upstreamRepo is one repository as a fake Hub describes it to a check.
type upstreamRepo struct {
	commit string
	files  map[string]string // path -> hash at commit
	config string            // config.json at commit
}

// checkFakeHub is a Hub for update checks: it answers the commit lookup, the
// listing at a commit and config.json at a commit, and records every request.
type checkFakeHub struct {
	srv     *httptest.Server
	mu      sync.Mutex
	repos   map[string]upstreamRepo
	reqs    []string
	ratelim string
	stall   chan struct{}
	// cdn, when set, is where config.json is handed off to, as the Hub does
	// for a file a repository keeps in LFS.
	cdn string
}

func newCheckFakeHub(t *testing.T, repos map[string]upstreamRepo) *checkFakeHub {
	t.Helper()
	h := &checkFakeHub{repos: repos}
	h.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.mu.Lock()
		h.reqs = append(h.reqs, r.URL.Path)
		stall := h.stall
		rl := h.ratelim
		h.mu.Unlock()
		if stall != nil {
			select {
			case <-stall:
			case <-r.Context().Done():
				return
			}
		}
		if rl != "" {
			w.Header().Set("RateLimit", rl)
		}
		h.mu.Lock()
		defer h.mu.Unlock()
		for id, repo := range h.repos {
			switch {
			case r.URL.Path == "/api/models/"+id:
				fmt.Fprintf(w, `{"id":%q,"sha":%q}`, id, repo.commit)
				return
			case r.URL.Path == "/api/models/"+id+"/tree/"+repo.commit:
				var out []hub.File
				for p, oid := range repo.files {
					f := hub.File{Path: p, Type: "file", Size: 1}
					if len(oid) == 64 {
						f.LFS = &struct {
							OID  string `json:"oid"`
							Size int64  `json:"size"`
						}{OID: oid, Size: 1}
					} else {
						f.OID = oid
					}
					out = append(out, f)
				}
				json.NewEncoder(w).Encode(out)
				return
			case r.URL.Path == "/"+id+"/resolve/"+repo.commit+"/config.json":
				if h.cdn != "" {
					http.Redirect(w, r, h.cdn+"/config.json", http.StatusFound)
					return
				}
				fmt.Fprint(w, repo.config)
				return
			}
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(h.srv.Close)
	return h
}

func (h *checkFakeHub) requests() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]string(nil), h.reqs...)
}

// newCheckApp is an app whose Hub is h, with update checks as given and the
// pacing shortened, holding one ready model per id at commit onDisk with the
// recorded hashes, and a log that can be read.
func newCheckApp(t *testing.T, h *checkFakeHub, on bool, recorded map[string]map[string]string) (*App, *bytes.Buffer) {
	t.Helper()
	var logs bytes.Buffer
	cfg := config.Default()
	cfg.UpdateCheck = on
	a, err := New(Options{Paths: config.NewPaths(t.TempDir()), Config: cfg,
		Log: slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelInfo}))})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.Close() })
	a.Hub.BaseURL = h.srv.URL
	a.updatePause = 0
	for id, files := range recorded {
		putReadyModel(t, a, registry.Model{RepoID: id, Commit: onDisk, FileHashes: files})
		dir := a.Paths.ModelDir(id)
		for p := range files {
			full := filepath.Join(dir, filepath.FromSlash(p))
			os.MkdirAll(filepath.Dir(full), 0o755)
			os.WriteFile(full, []byte("x"), 0o644)
		}
	}
	return a, &logs
}

func recordedFiles() map[string]string {
	return map[string]string{"config.json": hConfig, "model.safetensors": hWeights, "README.md": hReadme}
}

func update(t *testing.T, a *App, id string) *registry.UpdateCheck {
	t.Helper()
	m, err := a.Registry.Get(id)
	if err != nil {
		t.Fatal(err)
	}
	return m.Update
}

// Off sends nothing: a week of the schedule's ticks with checks off reaches
// the Hub not once, whatever is due (itd-2610030857275099 criterion 1).
func TestUpdateChecksOffSendNothingOverAWeek(t *testing.T) {
	h := newCheckFakeHub(t, map[string]upstreamRepo{"org/m": {commit: newer, files: recordedFiles()}})
	a, _ := newCheckApp(t, h, false, map[string]map[string]string{"org/m": recordedFiles()})
	start := time.Now()
	for at := start; at.Before(start.Add(7 * 24 * time.Hour)); at = at.Add(updateCheckTick) {
		a.updateCheckTickAt(context.Background(), at)
	}
	if got := h.requests(); len(got) != 0 {
		t.Errorf("%d requests left the Mac with checks off: %v", len(got), got[:min(len(got), 5)])
	}
	if u := update(t, a, "org/m"); u != nil {
		t.Errorf("a model was marked with checks off: %+v", u)
	}
}

// On, a model never checked is checked at the first tick, and then once per
// interval: daily over a week is seven or eight rounds, never one a tick
// (criterion 2).
func TestUpdateChecksOnRunAtStartAndEachInterval(t *testing.T) {
	h := newCheckFakeHub(t, map[string]upstreamRepo{"org/m": {commit: onDisk, files: recordedFiles()}})
	a, _ := newCheckApp(t, h, true, map[string]map[string]string{"org/m": recordedFiles()})
	start := time.Now()
	rounds := 0
	for at := start; at.Before(start.Add(7 * 24 * time.Hour)); at = at.Add(updateCheckTick) {
		before := len(h.requests())
		a.updateCheckTickAt(context.Background(), at)
		// The check stamps the real clock; move it to the simulated one so
		// the schedule sees the time the tick stands for.
		if m, _ := a.Registry.Get("org/m"); m.Update != nil && len(h.requests()) > before {
			a.Registry.SetUpdate("org/m", onDisk, registry.UpdateCheck{Status: m.Update.Status, CheckedAt: at})
			rounds++
		}
		if at == start && rounds != 1 {
			t.Fatal("a model never checked was not checked at the first tick")
		}
	}
	if rounds < 7 || rounds > 8 {
		t.Errorf("%d rounds in a week of daily checks", rounds)
	}
}

// At a start, a model checked within the interval is not asked about again.
func TestAStartWithinTheIntervalAsksNothing(t *testing.T) {
	h := newCheckFakeHub(t, map[string]upstreamRepo{"org/m": {commit: onDisk, files: recordedFiles()}})
	a, _ := newCheckApp(t, h, true, map[string]map[string]string{"org/m": recordedFiles()})
	a.Registry.SetUpdate("org/m", onDisk, registry.UpdateCheck{Status: registry.UpdateCurrent, CheckedAt: time.Now().Add(-time.Hour)})
	a.updateCheckTickAt(context.Background(), time.Now())
	if got := h.requests(); len(got) != 0 {
		t.Errorf("a model checked an hour ago was asked about again: %v", got)
	}
}

// A model whose version Dessau never recorded is not checked, and carries no
// mark (criterion 4).
func TestAModelOfUnknownVersionIsNotChecked(t *testing.T) {
	h := newCheckFakeHub(t, map[string]upstreamRepo{"org/m": {commit: newer, files: recordedFiles()}})
	a, _ := newCheckApp(t, h, true, nil)
	putReadyModel(t, a, registry.Model{RepoID: "org/m"})
	a.updateCheckTickAt(context.Background(), time.Now())
	if got := h.requests(); len(got) != 0 {
		t.Errorf("a model of unknown version was checked: %v", got)
	}
	if u := update(t, a, "org/m"); u != nil {
		t.Errorf("a model of unknown version carries a mark: %+v", u)
	}
}

// An upstream change that touches only a README marks nothing (criterion 3).
func TestAReadmeOnlyChangeMarksNothing(t *testing.T) {
	files := recordedFiles()
	files["README.md"] = hReadme2
	files["LICENSE"] = hReadme
	h := newCheckFakeHub(t, map[string]upstreamRepo{"org/m": {commit: newer, files: files}})
	a, _ := newCheckApp(t, h, true, map[string]map[string]string{"org/m": recordedFiles()})
	a.CheckForUpdates(context.Background(), a.updateCheckDue(time.Now()))
	if u := update(t, a, "org/m"); u == nil || u.Status != registry.UpdateCurrent {
		t.Errorf("Update = %+v, want current", u)
	}
	for _, p := range h.requests() {
		if strings.Contains(p, "config.json") {
			t.Error("config.json was read though it did not change")
		}
	}
}

// A changed weight marks the model, naming the newer commit.
func TestAChangedWeightMarksTheModel(t *testing.T) {
	files := recordedFiles()
	files["model.safetensors"] = hWeight2
	h := newCheckFakeHub(t, map[string]upstreamRepo{"org/m": {commit: newer, files: files}})
	a, _ := newCheckApp(t, h, true, map[string]map[string]string{"org/m": recordedFiles()})
	round := a.CheckForUpdates(context.Background(), a.updateCheckDue(time.Now()))
	if u := update(t, a, "org/m"); u == nil || u.Status != registry.UpdateAvailable || u.Commit != newer {
		t.Errorf("Update = %+v, want available at %s", u, newer)
	}
	if round.Checked != 1 || round.Marked != 1 {
		t.Errorf("round = %+v", round)
	}
}

// A newer version whose config.json names its own code is marked as one
// Dessau will not run (criterion 8); one that does not is offered.
func TestANewerVersionThatShipsCodeIsMarkedAsOneDessauWillNotRun(t *testing.T) {
	files := recordedFiles()
	files["config.json"] = hConfig2
	for cfg, want := range map[string]string{
		`{"model_type":"qwen3","model_file":"modeling.py"}`: registry.UpdateRunsOwnCode,
		`{"model_type":"qwen3"}`:                            registry.UpdateAvailable,
	} {
		h := newCheckFakeHub(t, map[string]upstreamRepo{"org/m": {commit: newer, files: files, config: cfg}})
		a, _ := newCheckApp(t, h, true, map[string]map[string]string{"org/m": recordedFiles()})
		a.CheckForUpdates(context.Background(), a.updateCheckDue(time.Now()))
		if u := update(t, a, "org/m"); u == nil || u.Status != want {
			t.Errorf("config %s: Update = %+v, want %s", cfg, u, want)
		}
	}
}

// A file the record holds no hash for is unknown, not changed: on disk it was
// downloaded and simply not recorded (iss-2610031239271873); absent from disk
// it is a file the newer version added.
func TestAnUnrecordedFileIsAddedOnlyWhenItIsNotOnDisk(t *testing.T) {
	files := recordedFiles()
	files["tokenizer.json"] = hConfig2
	h := newCheckFakeHub(t, map[string]upstreamRepo{"org/m": {commit: newer, files: files}})
	a, _ := newCheckApp(t, h, true, map[string]map[string]string{"org/m": recordedFiles()})
	dir := a.Paths.ModelDir("org/m")
	os.WriteFile(filepath.Join(dir, "tokenizer.json"), []byte("{}"), 0o644)
	a.CheckForUpdates(context.Background(), a.updateCheckDue(time.Now()))
	if u := update(t, a, "org/m"); u == nil || u.Status != registry.UpdateCurrent {
		t.Errorf("a downloaded file whose hash was not recorded marked the model: %+v", u)
	}

	os.Remove(filepath.Join(dir, "tokenizer.json"))
	a.CheckForUpdates(context.Background(), a.updateCheckDue(time.Now().Add(25*time.Hour)))
	if u := update(t, a, "org/m"); u == nil || u.Status != registry.UpdateAvailable {
		t.Errorf("a file the newer version added did not mark the model: %+v", u)
	}
}

// When HuggingFace cannot be reached, marks stay as they were and one line
// is logged for the round (criterion 6).
func TestAnUnreachableHubLeavesMarksAsTheyWere(t *testing.T) {
	h := newCheckFakeHub(t, nil)
	a, logs := newCheckApp(t, h, true, map[string]map[string]string{"org/a": recordedFiles(), "org/b": recordedFiles()})
	mark := registry.UpdateCheck{Status: registry.UpdateAvailable, Commit: newer, CheckedAt: time.Now().Add(-48 * time.Hour)}
	a.Registry.SetUpdate("org/a", onDisk, mark)
	h.srv.Close()

	a.updateCheckTickAt(context.Background(), time.Now())
	if u := update(t, a, "org/a"); u == nil || *u != mark {
		t.Errorf("the mark changed: %+v", u)
	}
	if u := update(t, a, "org/b"); u != nil {
		t.Errorf("an unreachable Hub marked a model: %+v", u)
	}
	if n := strings.Count(logs.String(), "\n"); n != 1 || !strings.Contains(logs.String(), "could not reach HuggingFace") {
		t.Errorf("the round logged %d lines, want one saying the Hub was not reached:\n%s", n, logs.String())
	}
	// And the next tick does not try again before the interval is out.
	before := logs.Len()
	a.updateCheckTickAt(context.Background(), time.Now().Add(time.Minute))
	if logs.Len() != before {
		t.Error("an unreachable Hub was asked again a minute later")
	}
}

// A Hub that answers but stalls does not slow serving: the round holds no
// lock a request needs, and each request it makes is bounded.
func TestAStalledHubDoesNotSlowServing(t *testing.T) {
	h := newCheckFakeHub(t, map[string]upstreamRepo{"org/m": {commit: newer, files: recordedFiles()}})
	h.stall = make(chan struct{})
	defer close(h.stall)
	a, _ := newCheckApp(t, h, true, map[string]map[string]string{"org/m": recordedFiles()})
	a.updateTimeout = 2 * time.Second

	done := make(chan struct{})
	go func() {
		defer close(done)
		a.updateCheckTickAt(context.Background(), time.Now())
	}()
	waitFor(t, "the check to reach the Hub", func() bool { return len(h.requests()) > 0 })
	t0 := time.Now()
	_ = a.Registry.List()
	_ = a.Pool.Resident()
	_ = a.Downloading()
	m, _ := a.Registry.Get("org/m")
	_, _ = a.ServedWindow(m)
	if d := time.Since(t0); d > 200*time.Millisecond {
		t.Errorf("serving paths took %v while the check waited on the Hub", d)
	}
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("the round did not give up on a stalled Hub")
	}
	if u := update(t, a, "org/m"); u != nil {
		t.Errorf("a stalled Hub marked the model: %+v", u)
	}
}

// A round stops once the Hub says few requests are left, leaving the rest for
// the next round.
func TestARoundStopsWhenTheHubsBudgetRunsLow(t *testing.T) {
	repos := map[string]upstreamRepo{}
	recorded := map[string]map[string]string{}
	for _, id := range []string{"org/a", "org/b", "org/c"} {
		repos[id] = upstreamRepo{commit: onDisk, files: recordedFiles()}
		recorded[id] = recordedFiles()
	}
	h := newCheckFakeHub(t, repos)
	h.ratelim = `"api";r=3;t=60`
	a, _ := newCheckApp(t, h, true, recorded)
	round := a.CheckForUpdates(context.Background(), a.updateCheckDue(time.Now()))
	if !round.StoppedEarly || round.Checked != 1 {
		t.Errorf("round = %+v, want one model checked and the round stopped", round)
	}
	if n := len(h.requests()); n != 1 {
		t.Errorf("%d requests, want the one before the budget ran low", n)
	}
}

// A decision model's newer version is marked as one that will be offered once
// reviewed, and offered once a Dessau release has reviewed it (criterion 9).
func TestADecisionModelWaitsForAReviewedVersion(t *testing.T) {
	files := recordedFiles()
	files["model.safetensors"] = hWeight2
	for reviewed, want := range map[string]string{
		"":    registry.UpdateAwaitingReview,
		newer: registry.UpdateAvailable,
	} {
		h := newCheckFakeHub(t, map[string]upstreamRepo{"org/decide": {commit: newer, files: files}})
		a, _ := newCheckApp(t, h, true, map[string]map[string]string{"org/decide": recordedFiles()})
		a.reviewed = func(id string) (bool, []string) {
			return id == "org/decide", []string{onDisk, reviewed}
		}
		a.CheckForUpdates(context.Background(), a.updateCheckDue(time.Now()))
		if u := update(t, a, "org/decide"); u == nil || u.Status != want || u.Commit != newer {
			t.Errorf("reviewed %q: Update = %+v, want %s", reviewed, u, want)
		}
	}
}

// A download records a check: resolving the commit it fetched answered the
// question, so the model is not asked about again until the interval is out.
func TestADownloadRecordsACheck(t *testing.T) {
	a := newTestApp(t)
	a.Hub.BaseURL = fakeHub(t).URL
	if err := a.Download("org/repo"); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "the model to become ready", func() bool {
		m, err := a.Registry.Get("org/repo")
		return err == nil && m.Ready()
	})
	u := update(t, a, "org/repo")
	if u == nil || u.Status != registry.UpdateCurrent || time.Since(u.CheckedAt) > time.Minute {
		t.Errorf("Update = %+v, want a current check stamped now", u)
	}
	if due := a.updateCheckDue(time.Now()); len(due) != 0 {
		t.Errorf("a model just downloaded is due a check: %v", due)
	}
}

// Turning checks off stops a round already running: no further model is
// asked about once the setting is off (review of step 2).
func TestTurningChecksOffStopsARoundInFlight(t *testing.T) {
	repos := map[string]upstreamRepo{}
	recorded := map[string]map[string]string{}
	for _, id := range []string{"org/a", "org/b", "org/c"} {
		repos[id] = upstreamRepo{commit: onDisk, files: recordedFiles()}
		recorded[id] = recordedFiles()
	}
	h := newCheckFakeHub(t, repos)
	a, _ := newCheckApp(t, h, true, recorded)
	a.updatePause = 200 * time.Millisecond
	done := make(chan struct{})
	go func() {
		defer close(done)
		a.updateCheckTickAt(context.Background(), time.Now())
	}()
	waitFor(t, "the first model to be asked about", func() bool { return len(h.requests()) > 0 })
	cfg := a.Config()
	cfg.UpdateCheck = false
	if err := a.SetConfig(cfg); err != nil {
		t.Fatal(err)
	}
	<-done
	if got := h.requests(); len(got) != 1 {
		t.Errorf("%d requests after checks were turned off mid-round, want only the one before: %v", len(got), got)
	}
}

// Each model is due on its own clock: one checked between two rounds is
// checked an interval after its own check, not two (review of step 2).
func TestEachModelIsDueOnItsOwnClock(t *testing.T) {
	h := newCheckFakeHub(t, map[string]upstreamRepo{
		"org/a": {commit: onDisk, files: recordedFiles()},
		"org/b": {commit: onDisk, files: recordedFiles()},
	})
	a, _ := newCheckApp(t, h, true, map[string]map[string]string{"org/a": recordedFiles(), "org/b": recordedFiles()})
	t0 := time.Now()
	a.Registry.SetUpdate("org/a", onDisk, registry.UpdateCheck{Status: registry.UpdateCurrent, CheckedAt: t0})
	a.Registry.SetUpdate("org/b", onDisk, registry.UpdateCheck{Status: registry.UpdateCurrent, CheckedAt: t0.Add(-12 * time.Hour)})
	// A round ran just now, which checked org/a. org/b is due twelve hours
	// from now; a round then must ask about it, not wait for the next whole
	// interval after this one.
	a.updateAttempted[dlKey("org/a")] = t0
	a.updateCheckTickAt(context.Background(), t0.Add(12*time.Hour+time.Minute))
	got := h.requests()
	if len(got) != 1 || got[0] != "/api/models/org/b" {
		t.Errorf("requests = %v, want org/b alone, due on its own clock", got)
	}
}

// The Hub's remaining budget as some other request last saw it does not end
// a round: each round starts from what its own answers say.
func TestAStaleBudgetFromAnotherRequestDoesNotEndARound(t *testing.T) {
	repos := map[string]upstreamRepo{}
	recorded := map[string]map[string]string{}
	for _, id := range []string{"org/a", "org/b"} {
		repos[id] = upstreamRepo{commit: onDisk, files: recordedFiles()}
		recorded[id] = recordedFiles()
	}
	h := newCheckFakeHub(t, repos)
	a, _ := newCheckApp(t, h, true, recorded)
	h.ratelim = `"api";r=1;t=60`
	a.Hub.RepoInfo(context.Background(), "org/a") // some other caller hears a low budget
	h.mu.Lock()
	h.ratelim, h.reqs = "", nil
	h.mu.Unlock()
	round := a.CheckForUpdates(context.Background(), a.updateCheckDue(time.Now()))
	if round.StoppedEarly || round.Checked != 2 {
		t.Errorf("round = %+v, want both models checked", round)
	}
}

// A config.json the record holds no hash for is read when the version moved,
// so a new model_file is never missed for want of a recorded hash.
func TestAnUnrecordedConfigIsReadWhenTheVersionMoves(t *testing.T) {
	rec := recordedFiles()
	delete(rec, "config.json")
	files := recordedFiles()
	files["model.safetensors"] = hWeight2
	h := newCheckFakeHub(t, map[string]upstreamRepo{"org/m": {commit: newer, files: files, config: `{"model_file":"x.py"}`}})
	a, _ := newCheckApp(t, h, true, map[string]map[string]string{"org/m": rec})
	os.WriteFile(filepath.Join(a.Paths.ModelDir("org/m"), "config.json"), []byte("{}"), 0o644)
	a.CheckForUpdates(context.Background(), a.updateCheckDue(time.Now()))
	if u := update(t, a, "org/m"); u == nil || u.Status != registry.UpdateRunsOwnCode {
		t.Errorf("Update = %+v, want runs_own_code", u)
	}
}

// A config.json the repository keeps in LFS is handed to the Hub's content
// CDN, which a check does not read from; the newer version is offered rather
// than the model counted unreachable at every check, and the update's own
// Precheck is what refuses one that names a model_file.
func TestAConfigOnTheContentCDNDoesNotStopTheCheck(t *testing.T) {
	cdn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"model_type":"qwen3"}`)
	}))
	defer cdn.Close()
	files := recordedFiles()
	files["config.json"] = hConfig2
	h := newCheckFakeHub(t, map[string]upstreamRepo{"org/m": {commit: newer, files: files}})
	h.cdn = cdn.URL
	a, _ := newCheckApp(t, h, true, map[string]map[string]string{"org/m": recordedFiles()})
	round := a.CheckForUpdates(context.Background(), a.updateCheckDue(time.Now()))
	if u := update(t, a, "org/m"); u == nil || u.Status != registry.UpdateAvailable {
		t.Errorf("Update = %+v (round %+v), want available", u, round)
	}
}

// A round cut short leaves the models it did not reach due at the next tick,
// and a round cut short by the switch says nothing about the Hub.
func TestARoundCutShortLeavesTheRestDue(t *testing.T) {
	repos := map[string]upstreamRepo{}
	recorded := map[string]map[string]string{}
	for _, id := range []string{"org/a", "org/b"} {
		repos[id] = upstreamRepo{commit: onDisk, files: recordedFiles()}
		recorded[id] = recordedFiles()
	}
	h := newCheckFakeHub(t, repos)
	h.ratelim = `"api";r=1;t=60`
	a, logs := newCheckApp(t, h, true, recorded)
	now := time.Now()
	a.updateCheckTickAt(context.Background(), now)
	if n := len(h.requests()); n != 1 {
		t.Fatalf("%d requests in the first round, want 1", n)
	}
	h.mu.Lock()
	h.ratelim = ""
	h.mu.Unlock()
	if due := a.updateCheckDue(now.Add(time.Minute)); len(due) != 1 {
		t.Errorf("%d models due a minute later, want the one the round did not reach", len(due))
	}
	if strings.Contains(logs.String(), "could not reach") {
		t.Error("a round cut short by the budget says the Hub was not reached")
	}
}
