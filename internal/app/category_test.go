package app

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/Dessau/internal/config"
	"github.com/intentdriven/Dessau/internal/registry"
)

// completionHub is a Hub that answers /api/models/<repo> with a fixed body per
// repo and counts every request it sees.
type completionHub struct {
	srv  *httptest.Server
	mu   sync.Mutex
	hits map[string]int
}

func newCompletionHub(t *testing.T, answers map[string]string) *completionHub {
	t.Helper()
	h := &completionHub{hits: map[string]int{}}
	h.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.mu.Lock()
		h.hits[r.URL.Path]++
		h.mu.Unlock()
		repo := strings.TrimPrefix(r.URL.Path, "/api/models/")
		body, ok := answers[repo]
		if !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(body))
	}))
	t.Cleanup(h.srv.Close)
	return h
}

func (h *completionHub) hitsFor(path string) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.hits[path]
}

func (h *completionHub) total() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	n := 0
	for _, c := range h.hits {
		n += c
	}
	return n
}

func putReadyModel(t *testing.T, a *App, m registry.Model) {
	t.Helper()
	m.State = registry.StateReady
	if m.Path == "" {
		m.Path = a.Paths.ModelDir(m.RepoID)
	}
	if err := a.Registry.Put(m); err != nil {
		t.Fatal(err)
	}
}

// The completion job asks the Hub only for the ready models it has no word
// for, records what it hears, and leaves every other model — one with a word,
// one the Hub is known to be silent about — unasked. Its requests are the
// Hub's, not a model's: no request statistic moves (iss-2609202237468921).
func TestCompleteCategoriesAsksOnlyForModelsWithNoHubWord(t *testing.T) {
	a := newTestApp(t)
	hub := newCompletionHub(t, map[string]string{
		"org/adopted": `{"pipeline_tag":"text-generation","tags":["mlx","conversational"]}`,
		"org/tagged":  `{"pipeline_tag":"automatic-speech-recognition","tags":["mlx"]}`,
		"org/silent":  `{"pipeline_tag":"text-generation","tags":["mlx"]}`,
	})
	a.Hub.BaseURL = hub.srv.URL
	a.Stats.SetEnabled(true)

	putReadyModel(t, a, registry.Model{RepoID: "org/adopted", ChatTemplate: true})
	putReadyModel(t, a, registry.Model{RepoID: "org/tagged", PipelineTag: "text-generation", Tags: []string{"conversational"}})
	putReadyModel(t, a, registry.Model{RepoID: "org/silent", HubSilent: true})
	if err := a.Registry.Put(registry.Model{RepoID: "org/failed", Path: a.Paths.ModelDir("org/failed"), State: registry.StateFailed}); err != nil {
		t.Fatal(err)
	}

	a.CompleteCategories(context.Background())

	m, _ := a.Registry.Get("org/adopted")
	if m.PipelineTag != "text-generation" || strings.Join(m.Tags, ",") != "mlx,conversational" {
		t.Errorf("the adopted model did not gain the Hub's words: %+v", m)
	}
	if !m.ChatTemplate {
		t.Error("recording the category lost the template")
	}
	if m, _ := a.Registry.Get("org/tagged"); m.PipelineTag != "text-generation" {
		t.Errorf("a model with a word was re-asked and overwritten: %+v", m)
	}
	if m, _ := a.Registry.Get("org/silent"); m.PipelineTag != "" || !m.HubSilent {
		t.Errorf("a model the Hub is silent about was asked again: %+v", m)
	}
	if got := hub.hitsFor("/api/models/org/adopted"); got != 1 {
		t.Errorf("the Hub was asked %d times for the adopted model, want once", got)
	}
	if got := hub.total(); got != 1 {
		t.Errorf("the Hub saw %d requests, want exactly one — for the adopted model, and nothing else", got)
	}
	if v := a.Stats.View(); len(v.Requests) != 0 || len(v.Models) != 0 {
		t.Errorf("the job's request reached the statistics: %d requests, %d models", len(v.Requests), len(v.Models))
	}
}

// A Hub that does not answer — offline, a 404, a body that will not decode —
// leaves the model exactly as it was: no word, not marked silent, so the next
// start asks again.
func TestCompleteCategoriesLeavesAModelAloneWhenTheHubDoesNotAnswer(t *testing.T) {
	a := newTestApp(t)
	hub := newCompletionHub(t, map[string]string{
		"org/garbled": `{"pipeline_tag": `,
	})
	a.Hub.BaseURL = hub.srv.URL
	putReadyModel(t, a, registry.Model{RepoID: "org/missing", ChatTemplate: true})
	putReadyModel(t, a, registry.Model{RepoID: "org/garbled"})

	a.CompleteCategories(context.Background())

	for _, id := range []string{"org/missing", "org/garbled"} {
		m, err := a.Registry.Get(id)
		if err != nil {
			t.Fatal(err)
		}
		if m.HasHubWord() || m.HubSilent {
			t.Errorf("%s changed although the Hub did not answer: %+v", id, m)
		}
	}
	if m, _ := a.Registry.Get("org/missing"); !m.ChatTemplate {
		t.Error("a failed request lost the template")
	}
	if got := hub.total(); got != 2 {
		t.Errorf("the Hub saw %d requests, want one per model", got)
	}
}

// A repository the Hub answers for and has no words about is asked once,
// marked, and not asked again at the next start.
func TestCompleteCategoriesAsksARepoWithNoWordsOnlyOnce(t *testing.T) {
	a := newTestApp(t)
	hub := newCompletionHub(t, map[string]string{
		"org/bare": `{"id":"org/bare"}`,
	})
	a.Hub.BaseURL = hub.srv.URL
	putReadyModel(t, a, registry.Model{RepoID: "org/bare", ChatTemplate: true})

	a.CompleteCategories(context.Background())
	a.CompleteCategories(context.Background())

	m, _ := a.Registry.Get("org/bare")
	if m.HasHubWord() || !m.HubSilent {
		t.Errorf("after an answer with no words: %+v, want HubSilent and no words", m)
	}
	if got := hub.hitsFor("/api/models/org/bare"); got != 1 {
		t.Errorf("the Hub was asked %d times for a repo it has no words for, want once", got)
	}
}

// A job with nothing to do touches nothing: no request leaves the machine.
func TestCompleteCategoriesWithNothingToDoAsksNothing(t *testing.T) {
	a := newTestApp(t)
	hub := newCompletionHub(t, nil)
	a.Hub.BaseURL = hub.srv.URL
	putReadyModel(t, a, registry.Model{RepoID: "org/tagged", PipelineTag: "text-generation"})

	a.CompleteCategories(context.Background())

	if got := hub.total(); got != 0 {
		t.Errorf("the Hub saw %d requests with nothing to complete", got)
	}
}

// gatedHandler is a log handler that holds the job's completion line until
// the test lets it go, so the job's goroutine is provably still running when
// Close is called.
type gatedHandler struct {
	slog.Handler
	reached chan struct{}
	release chan struct{}
	once    sync.Once
}

func (h *gatedHandler) Handle(ctx context.Context, r slog.Record) error {
	if strings.HasPrefix(r.Message, "recorded what the hub says") {
		h.once.Do(func() { close(h.reached) })
		<-h.release
	}
	return h.Handler.Handle(ctx, r)
}

// Close joins the completion job the way it joins every other background
// job: a start that is shutting down waits for the answer in flight to be
// recorded or abandoned rather than exiting mid-write.
func TestCloseWaitsForTheCompletionJob(t *testing.T) {
	// A text handler to io.Discard rather than slog.DiscardHandler, whose
	// Enabled says no and so is never handed the record.
	gate := &gatedHandler{Handler: slog.NewTextHandler(io.Discard, nil), reached: make(chan struct{}), release: make(chan struct{})}
	paths := config.NewPaths(t.TempDir())
	a, err := New(Options{Paths: paths, Config: config.Default(), Log: slog.New(gate)})
	if err != nil {
		t.Fatal(err)
	}
	hub := newCompletionHub(t, map[string]string{
		"org/adopted": `{"pipeline_tag":"text-generation","tags":["conversational"]}`,
	})
	a.Hub.BaseURL = hub.srv.URL
	putReadyModel(t, a, registry.Model{RepoID: "org/adopted"})

	a.StartCompletingCategories()
	select {
	case <-gate.reached:
	case <-time.After(10 * time.Second):
		t.Fatal("the job never reached its completion line")
	}

	closed := make(chan struct{})
	go func() {
		a.Close()
		close(closed)
	}()
	select {
	case <-closed:
		t.Fatal("Close returned while the completion job was still running")
	case <-time.After(200 * time.Millisecond):
	}
	close(gate.release)
	select {
	case <-closed:
	case <-time.After(10 * time.Second):
		t.Fatal("Close did not return once the job finished")
	}
	if m, _ := a.Registry.Get("org/adopted"); m.PipelineTag != "text-generation" {
		t.Errorf("the answer in flight was not recorded: %+v", m)
	}
}

// Stopping cuts a job short: a start that is shutting down does not sit out
// the pause between two models, and does not ask for the next one.
func TestCloseCancelsTheCompletionJobsPause(t *testing.T) {
	a := newTestApp(t)
	hub := newCompletionHub(t, map[string]string{
		"org/first":  `{"pipeline_tag":"text-generation"}`,
		"org/second": `{"pipeline_tag":"text-generation"}`,
	})
	a.Hub.BaseURL = hub.srv.URL
	putReadyModel(t, a, registry.Model{RepoID: "org/first"})
	putReadyModel(t, a, registry.Model{RepoID: "org/second"})

	a.StartCompletingCategories()
	waitFor(t, "the first model to be asked", func() bool { return hub.total() >= 1 })
	start := time.Now()
	a.Close()
	if took := time.Since(start); took >= categoryPause {
		t.Errorf("Close took %v, so it sat out the job's pause instead of cutting it short", took)
	}
	if got := hub.total(); got != 1 {
		t.Errorf("the Hub saw %d requests after Close, want the one before it", got)
	}
}
