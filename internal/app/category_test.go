package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

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
