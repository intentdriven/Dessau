package contextprobe

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/Dessau/internal/registry"
	"github.com/intentdriven/Dessau/internal/runtime"
	"github.com/intentdriven/Dessau/internal/selftest"
)

// fakeGateway stands in for this Mac's own OpenAI endpoint: it counts the
// prompt the way the fake model server does, answers 200 with the count up
// to Accept, and above that answers the way the caller configures — a 500
// (the model's own limit), a 504 (the gateway's prefill deadline) or a 400
// (its served-window check).
type fakeGateway struct {
	srv *httptest.Server

	mu       sync.Mutex
	accept   int64
	above    int
	delay    time.Duration
	prompts  []string
	requests int
	auth     string
}

func newFakeGateway(t *testing.T, accept int64, above int) *fakeGateway {
	t.Helper()
	g := &fakeGateway{accept: accept, above: above}
	g.srv = httptest.NewServer(http.HandlerFunc(g.handle))
	t.Cleanup(g.srv.Close)
	return g
}

func (g *fakeGateway) handle(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Messages []struct {
			Content string `json:"content"`
		} `json:"messages"`
	}
	raw, _ := io.ReadAll(r.Body)
	_ = json.Unmarshal(raw, &body)
	prompt := ""
	for _, m := range body.Messages {
		prompt += m.Content
	}
	g.mu.Lock()
	g.prompts = append(g.prompts, prompt)
	g.requests++
	g.auth = r.Header.Get("Authorization")
	accept, above, delay := g.accept, g.above, g.delay
	g.mu.Unlock()
	tokens := int64(len(prompt)/4 + 3)
	if delay > 0 {
		select {
		case <-time.After(delay):
		case <-r.Context().Done():
			return
		}
	}
	if tokens > accept {
		http.Error(w, `{"error":"no"}`, above)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"usage": map[string]any{"prompt_tokens": tokens}})
}

func (g *fakeGateway) seen() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return append([]string(nil), g.prompts...)
}

// fakeSources is the app as the probe sees it.
type fakeSources struct {
	mu         sync.Mutex
	cands      []Candidate
	available  int64
	unloads    []string
	unloadErr  error
	saved      map[string]*registry.Measurement
	incomplete map[string]bool
	url, key   string
}

func newFakeSources(url string, cands ...Candidate) *fakeSources {
	return &fakeSources{cands: cands, available: 1 << 40, saved: map[string]*registry.Measurement{}, incomplete: map[string]bool{}, url: url}
}

func (f *fakeSources) Candidates() []Candidate {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]Candidate, len(f.cands))
	for i, c := range f.cands {
		c.Measured = f.saved[c.RepoID]
		c.Incomplete = f.incomplete[c.RepoID]
		out[i] = c
	}
	return out
}
func (f *fakeSources) Provenance(string) registry.Provenance {
	return registry.Provenance{Runtime: "0.31.3", BudgetBytes: 1, DecodeConcurrency: 4, ServedContext: 131072}
}
func (f *fakeSources) Available() int64 { f.mu.Lock(); defer f.mu.Unlock(); return f.available }
func (f *fakeSources) Unload(id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.unloads = append(f.unloads, id)
	return f.unloadErr
}
func (f *fakeSources) Save(id string, m *registry.Measurement) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.saved[id] = m
	return nil
}
func (f *fakeSources) MarkIncomplete(id string, on bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.incomplete[id] = on
	return nil
}
func (f *fakeSources) Endpoint() (string, string) { return f.url, f.key }
func (f *fakeSources) result(id string) *registry.Measurement {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.saved[id]
}
func (f *fakeSources) unloaded() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.unloads...)
}

// fakePool is the loop's view of the pool: nothing resident, quiet, unless a
// test says otherwise.
type fakePool struct {
	mu          sync.Mutex
	ready       []string
	inFlight    map[string]int
	lastUsed    map[string]time.Time
	waiting     int
	downloading int
	fits        bool
}

func newFakePool(models ...string) *fakePool {
	return &fakePool{ready: models, inFlight: map[string]int{}, lastUsed: map[string]time.Time{}, fits: true}
}
func (p *fakePool) Ready() []string { return p.ready }
func (p *fakePool) Acquire(context.Context, string) (selftest.Upstream, func(), error) {
	return selftest.Upstream{}, func() {}, errors.New("the probe never acquires")
}
func (p *fakePool) Activity() selftest.Activity {
	p.mu.Lock()
	defer p.mu.Unlock()
	act := selftest.Activity{Waiting: p.waiting, Downloading: p.downloading}
	for id, n := range p.inFlight {
		act.Models = append(act.Models, selftest.ModelActivity{RepoID: id, InFlight: n, LastUsed: p.lastUsed[id]})
	}
	for id, at := range p.lastUsed {
		if _, ok := p.inFlight[id]; !ok {
			act.Models = append(act.Models, selftest.ModelActivity{RepoID: id, LastUsed: at})
		}
	}
	return act
}
func (p *fakePool) Unload(string) error { return nil }
func (p *fakePool) Concurrency() int    { return 1 }
func (p *fakePool) Fits(string) bool    { p.mu.Lock(); defer p.mu.Unlock(); return p.fits }

func quiet() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func probeOf(src *fakeSources, enabled bool) *Probe {
	return New(Options{
		Sources: src, Enabled: func() bool { return enabled }, Log: quiet(),
		StepTimeout:  func(int) time.Duration { return 2 * time.Second },
		MemoryMargin: 0, UnloadWait: 100 * time.Millisecond,
	})
}

// runner drives the probe as a job of the self-test's loop, with the
// self-test's own set switched off, on a test cadence.
func runner(t *testing.T, pool *fakePool, p *Probe) *selftest.Runner {
	t.Helper()
	r := selftest.New(selftest.Options{
		Server: pool, Path: filepath.Join(t.TempDir(), selftest.FileName), Log: quiet(),
		Tick: 5 * time.Millisecond, Poll: 2 * time.Millisecond, Quiet: time.Nanosecond,
		Jobs: []selftest.Job{p}, SelfTest: func() bool { return false },
	})
	t.Cleanup(r.Close)
	return r
}

// runnerQuiet is runner with a real quiet period, for the held-back cases.
func runnerQuiet(t *testing.T, pool *fakePool, p *Probe, quietFor time.Duration) *selftest.Runner {
	t.Helper()
	r := selftest.New(selftest.Options{
		Server: pool, Path: filepath.Join(t.TempDir(), selftest.FileName), Log: quiet(),
		Tick: 5 * time.Millisecond, Poll: 2 * time.Millisecond, Quiet: quietFor,
		Jobs: []selftest.Job{p}, SelfTest: func() bool { return false },
	})
	t.Cleanup(r.Close)
	return r
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("gave up waiting for %s", what)
}

var model = Candidate{RepoID: "org/m", Declared: 131072, Served: 131072, Bytes: 1 << 30, KVChargePerToken: 1024}

// The switch off, nothing queued: nothing is due, whatever finished
// downloading. "Measure now" makes a model due whatever the switch says.
func TestNothingIsMeasuredWhileTheSwitchIsOff(t *testing.T) {
	src := newFakeSources("http://127.0.0.1:1", model)
	p := probeOf(src, false)
	if due := p.Due([]string{"org/m"}, time.Now()); due != "" {
		t.Errorf("Due = %q with the switch off", due)
	}
	p.MeasureNow("org/m")
	if due := p.Due([]string{"org/m"}, time.Now()); due != "org/m" {
		t.Errorf("Due = %q after Measure now", due)
	}
	// And a model with a current measurement is not due on the switch.
	src.saved["org/m"] = &registry.Measurement{Window: 1000, Bound: registry.BoundModel}
	on := probeOf(src, true)
	if due := on.Due([]string{"org/m"}, time.Now()); due != "" {
		t.Errorf("Due = %q for a model already measured", due)
	}
	src.saved["org/m"].Stale = registry.StaleRuntime
	if due := on.Due([]string{"org/m"}, time.Now()); due != "org/m" {
		t.Errorf("Due = %q for a model whose measurement is stale", due)
	}
}

// The probe finds the window the server accepts: a bisection against a
// server that refuses above a size converges within the tolerance, the
// bound is the model's, and every step unloaded first.
func TestAProbeFindsTheWindowTheServerAccepts(t *testing.T) {
	gw := newFakeGateway(t, 40_000, http.StatusInternalServerError)
	src := newFakeSources(gw.srv.URL, model)
	src.key = "k"
	p := probeOf(src, true)
	r := runner(t, newFakePool("org/m"), p)
	r.SetEnabled(true)
	waitFor(t, "a measurement", func() bool { return src.result("org/m") != nil })
	m := src.result("org/m")
	if m.Bound != registry.BoundModel {
		t.Errorf("bound = %q, want the model's own", m.Bound)
	}
	if m.Window < 35_000 || m.Window > 40_000 {
		t.Errorf("window = %d, want within a tolerance under 40,000", m.Window)
	}
	if m.Runtime != "0.31.3" || m.ServedContext != 131072 {
		t.Errorf("provenance not recorded: %+v", m)
	}
	gw.mu.Lock()
	auth := gw.auth
	gw.mu.Unlock()
	if auth != "Bearer k" {
		t.Errorf("the configured key was not sent: %q", auth)
	}
	if n := len(src.unloaded()); n < len(gw.seen()) {
		t.Errorf("%d unloads for %d steps; every step unloads first", n, len(gw.seen()))
	}
}

// Each step's prompt begins with its own nonce, so no two share a prefix and
// no retained cache can flatter a later reading.
func TestEveryStepReloadsAndSharesNoPrefix(t *testing.T) {
	gw := newFakeGateway(t, 20_000, http.StatusInternalServerError)
	src := newFakeSources(gw.srv.URL, model)
	p := probeOf(src, true)
	r := runner(t, newFakePool("org/m"), p)
	r.SetEnabled(true)
	waitFor(t, "a measurement", func() bool { return src.result("org/m") != nil })
	seen := gw.seen()
	if len(seen) < 3 {
		t.Fatalf("only %d steps", len(seen))
	}
	for i := range seen {
		for j := range seen {
			if i != j && strings.HasPrefix(seen[i], seen[j][:64]) {
				t.Errorf("steps %d and %d share a prefix", i, j)
			}
		}
	}
}

// A step the gateway's deadline stopped bounds Dessau's configuration, not
// the model: the figure is a floor and names the bound.
func TestAStepStoppedByTheDeadlineIsAFloor(t *testing.T) {
	gw := newFakeGateway(t, 20_000, http.StatusGatewayTimeout)
	src := newFakeSources(gw.srv.URL, model)
	p := probeOf(src, true)
	r := runner(t, newFakePool("org/m"), p)
	r.SetEnabled(true)
	waitFor(t, "a measurement", func() bool { return src.result("org/m") != nil })
	if m := src.result("org/m"); m.Bound != registry.BoundPrefillDeadline || m.IsLimit() {
		t.Errorf("measurement = %+v, want a floor bounded by the prefill deadline", m)
	}
}

// A served window below the declared one caps the sweep, and a model that
// accepts everything up to it is bounded by the served window.
func TestAServedWindowCapsTheSweep(t *testing.T) {
	gw := newFakeGateway(t, 1<<40, http.StatusInternalServerError)
	c := model
	c.Served = 16_384
	src := newFakeSources(gw.srv.URL, c)
	p := probeOf(src, true)
	r := runner(t, newFakePool("org/m"), p)
	r.SetEnabled(true)
	waitFor(t, "a measurement", func() bool { return src.result("org/m") != nil })
	m := src.result("org/m")
	if m.Bound != registry.BoundServedWindow || m.Window > 16_384+512 {
		t.Errorf("measurement = %+v, want bounded by the served window of 16,384", m)
	}
}

// A step whose projected footprint comes within the margin of what the Mac
// has free is skipped, recorded as the memory guard's, with its projection.
func TestAStepTooLargeForThisMacIsSkippedWithItsProjection(t *testing.T) {
	gw := newFakeGateway(t, 1<<40, http.StatusInternalServerError)
	src := newFakeSources(gw.srv.URL, model)
	// The flat charge is 1.2 GiB; 1 KiB a token puts 8,192 tokens at 8 MiB
	// over it. Room for about 6,000 tokens of cache and no more.
	src.available = 1<<30 + 1<<30/5 + 6_000*1024
	p := probeOf(src, true)
	r := runner(t, newFakePool("org/m"), p)
	r.SetEnabled(true)
	waitFor(t, "a measurement", func() bool { return src.result("org/m") != nil })
	m := src.result("org/m")
	if m.Bound != registry.BoundMemoryGuard || m.GuardBytes == 0 {
		t.Errorf("measurement = %+v, want the memory guard with its projection", m)
	}
	if m.Window > 6_100 {
		t.Errorf("window = %d beyond what the guard allowed", m.Window)
	}
}

// A real request during a step: the probe's request is cancelled within the
// poll, the model is unloaded so the abandoned prefill stops, and the step is
// not kept as a reading; the run resumes from its bounds when the Mac is idle
// again.
func TestAProbeYieldsToARealRequestAndResumesFromItsBounds(t *testing.T) {
	gw := newFakeGateway(t, 40_000, http.StatusInternalServerError)
	gw.delay = 30 * time.Millisecond
	src := newFakeSources(gw.srv.URL, model)
	p := probeOf(src, true)
	pool := newFakePool("org/m")
	r := runner(t, pool, p)
	r.SetEnabled(true)
	waitFor(t, "the sweep to reach 4096", func() bool {
		for _, s := range gw.seen() {
			if len(s) > 4096*4 {
				return true
			}
		}
		return false
	})
	before := len(gw.seen())
	pool.mu.Lock()
	pool.inFlight["org/other"] = 1
	pool.mu.Unlock()
	arrived := time.Now()
	waitFor(t, "the probe to yield", func() bool { return r.Status().Job == "" })
	if took := time.Since(arrived); took > 500*time.Millisecond {
		t.Errorf("the probe took %v to yield", took)
	}
	if src.result("org/m") != nil {
		t.Error("a yielded probe wrote a figure")
	}
	if u := src.unloaded(); len(u) == 0 || u[len(u)-1] != "org/m" {
		t.Error("the yielding probe did not unload the model it was driving")
	}
	time.Sleep(30 * time.Millisecond)
	if n := len(gw.seen()); n > before+1 {
		t.Errorf("%d requests while a client was in flight (%d before)", n-before, before)
	}
	// The client leaves; the probe resumes above what it had verified rather
	// than from the floor.
	pool.mu.Lock()
	delete(pool.inFlight, "org/other")
	pool.mu.Unlock()
	waitFor(t, "a measurement after resuming", func() bool { return src.result("org/m") != nil })
	resumed := gw.seen()[before:]
	for _, s := range resumed {
		if len(s) < 2048*4 {
			t.Errorf("after resuming the probe sent a %d-character prompt, below what it had verified", len(s))
		}
	}
}

// An unload refused because a client took the model discards the step: a
// server that kept an abandoned cache measures the cache.
func TestAReadingIsDiscardedWhenTheServerCouldNotBeStopped(t *testing.T) {
	gw := newFakeGateway(t, 40_000, http.StatusInternalServerError)
	gw.delay = 30 * time.Millisecond
	src := newFakeSources(gw.srv.URL, model)
	p := probeOf(src, true)
	pool := newFakePool("org/m")
	r := runner(t, pool, p)
	r.SetEnabled(true)
	waitFor(t, "the first step", func() bool { return len(gw.seen()) >= 1 })
	src.mu.Lock()
	src.unloadErr = runtime.ErrBusy
	src.mu.Unlock()
	pool.mu.Lock()
	pool.inFlight["org/m"] = 2 // the probe's one, and a client's
	pool.mu.Unlock()
	waitFor(t, "the probe to yield", func() bool { return r.Status().Job == "" })
	if src.result("org/m") != nil {
		t.Error("a figure was written from a step whose server could not be stopped")
	}
	p.mu.Lock()
	b := p.bounds["org/m"]
	p.mu.Unlock()
	if b != nil && b.lo > 2048 {
		t.Errorf("bounds advanced to %d on a discarded step", b.lo)
	}
}

// The switch going off, or Dessau quitting, mid-probe: no partial figure,
// the model unloaded, and the model marked incomplete so the next start says
// so rather than retrying.
func TestAnInterruptedProbeWritesNoFigure(t *testing.T) {
	gw := newFakeGateway(t, 40_000, http.StatusInternalServerError)
	gw.delay = 30 * time.Millisecond
	src := newFakeSources(gw.srv.URL, model)
	p := probeOf(src, true)
	r := runner(t, newFakePool("org/m"), p)
	r.SetEnabled(true)
	waitFor(t, "the first step", func() bool { return len(gw.seen()) >= 2 })
	r.SetEnabled(false)
	if src.result("org/m") != nil {
		t.Error("an interrupted probe wrote a figure")
	}
	src.mu.Lock()
	incomplete := src.incomplete["org/m"]
	src.mu.Unlock()
	if !incomplete {
		t.Error("the interrupted probe was not marked incomplete")
	}
	if u := src.unloaded(); len(u) == 0 || u[len(u)-1] != "org/m" {
		t.Error("the model was left loaded")
	}
	// And it is not retried on its own.
	if due := p.Due([]string{"org/m"}, time.Now()); due != "" {
		t.Errorf("an incomplete probe is due again on its own: %q", due)
	}
}

// A probe due does not start while the Mac is busy, and the loop says which
// of the three conditions held it back.
func TestAProbeWaitsForIdleAndNamesWhatHeldItBack(t *testing.T) {
	for _, tt := range []struct {
		name string
		set  func(*fakePool)
		want string
	}{
		{"a request in flight", func(p *fakePool) { p.inFlight["org/other"] = 1 }, selftest.HeldByInFlight},
		{"a waiting caller", func(p *fakePool) { p.waiting = 1 }, selftest.HeldByWaiting},
		{"a download running", func(p *fakePool) { p.downloading = 1 }, selftest.HeldByDownloading},
		{"a recent request", func(p *fakePool) { p.lastUsed["org/other"] = time.Now() }, selftest.HeldByRecent},
	} {
		t.Run(tt.name, func(t *testing.T) {
			gw := newFakeGateway(t, 40_000, http.StatusInternalServerError)
			src := newFakeSources(gw.srv.URL, model)
			p := probeOf(src, true)
			pool := newFakePool("org/m")
			tt.set(pool)
			r := runnerQuiet(t, pool, p, time.Hour)
			r.SetEnabled(true)
			waitFor(t, "the loop to say what held it", func() bool { return r.Status().HeldBy == tt.want })
			if r.Status().Due != "org/m" {
				t.Errorf("the loop does not say which model was due: %+v", r.Status())
			}
			if len(gw.seen()) != 0 {
				t.Errorf("%d requests were sent while %s", len(gw.seen()), tt.name)
			}
		})
	}
}

// A model that would need an eviction to load is not probed: the probe
// acquires only when the room is already free.
func TestAPinnedModelIsNeverEvictedForAProbe(t *testing.T) {
	gw := newFakeGateway(t, 40_000, http.StatusInternalServerError)
	src := newFakeSources(gw.srv.URL, model)
	p := probeOf(src, true)
	pool := newFakePool("org/m")
	pool.fits = false
	r := runner(t, pool, p)
	r.SetEnabled(true)
	time.Sleep(40 * time.Millisecond)
	if len(gw.seen()) != 0 {
		t.Errorf("%d requests were sent for a model that does not fit", len(gw.seen()))
	}
	pool.mu.Lock()
	pool.fits = true
	pool.mu.Unlock()
	waitFor(t, "a measurement once there is room", func() bool { return src.result("org/m") != nil })
}

// A model that does not answer even at the floor is marked incomplete and
// not looped on; "Measure now" is the only retry.
func TestAModelThatFailsAtTheFloorIsLeftAlone(t *testing.T) {
	gw := newFakeGateway(t, 10, http.StatusInternalServerError)
	src := newFakeSources(gw.srv.URL, model)
	p := probeOf(src, true)
	r := runner(t, newFakePool("org/m"), p)
	r.SetEnabled(true)
	waitFor(t, "the floor to fail", func() bool {
		src.mu.Lock()
		defer src.mu.Unlock()
		return src.incomplete["org/m"]
	})
	time.Sleep(30 * time.Millisecond)
	if n := len(gw.seen()); n != 1 {
		t.Errorf("%d requests after the floor failed, want the one", n)
	}
}

// The probe's own timer is the gateway's budget with a margin; if it fires
// first anyway, the step is the deadline's and never the model's, and the
// abandoned prefill is stopped.
func TestTheProbesOwnTimerIsTheDeadlinesNotTheModels(t *testing.T) {
	gw := newFakeGateway(t, 1<<40, http.StatusInternalServerError)
	src := newFakeSources(gw.srv.URL, model)
	p := New(Options{
		Sources: src, Enabled: func() bool { return true }, Log: quiet(),
		StepTimeout: func(body int) time.Duration {
			if body > 8000*4 {
				return 50 * time.Millisecond
			}
			return 2 * time.Second
		},
		UnloadWait: 100 * time.Millisecond,
	})
	gw.mu.Lock()
	gw.delay = 200 * time.Millisecond
	gw.mu.Unlock()
	r := runner(t, newFakePool("org/m"), p)
	r.SetEnabled(true)
	waitFor(t, "a measurement", func() bool { return src.result("org/m") != nil })
	if m := src.result("org/m"); m.Bound != registry.BoundPrefillDeadline || m.IsLimit() {
		t.Errorf("measurement = %+v, want a floor filed as the prefill deadline's", m)
	}
}

// A 503 is the pool without room, not the model: the step is held for a
// later tick and no bound is drawn from it.
func TestAPoolWithoutRoomHoldsTheStepRatherThanBoundingIt(t *testing.T) {
	gw := newFakeGateway(t, 40_000, http.StatusServiceUnavailable)
	src := newFakeSources(gw.srv.URL, model)
	p := probeOf(src, true)
	r := runner(t, newFakePool("org/m"), p)
	r.SetEnabled(true)
	waitFor(t, "the sweep to reach the 503", func() bool { return len(gw.seen()) >= 6 })
	time.Sleep(50 * time.Millisecond)
	if src.result("org/m") != nil {
		t.Errorf("a 503 produced a figure: %+v", src.result("org/m"))
	}
	p.mu.Lock()
	b := p.bounds["org/m"]
	p.mu.Unlock()
	if b == nil || b.bound == registry.BoundModel {
		t.Errorf("a 503 was filed as a bound: %+v", b)
	}
}

// An answer no figure can be drawn from — a key refused — abandons the run:
// marked incomplete, nothing saved, not retried on its own.
func TestARefusedKeyAbandonsTheRun(t *testing.T) {
	gw := newFakeGateway(t, 0, http.StatusUnauthorized)
	src := newFakeSources(gw.srv.URL, model)
	p := probeOf(src, true)
	r := runner(t, newFakePool("org/m"), p)
	r.SetEnabled(true)
	waitFor(t, "the run to be abandoned", func() bool {
		src.mu.Lock()
		defer src.mu.Unlock()
		return src.incomplete["org/m"]
	})
	time.Sleep(30 * time.Millisecond)
	if n := len(gw.seen()); n != 1 || src.result("org/m") != nil {
		t.Errorf("%d requests and result %+v after a 401", n, src.result("org/m"))
	}
}

// A declared window past the probe's ceiling, or a calibration that counts
// oddly, cannot make the probe build a prompt of gigabytes: the sweep is
// capped and the characters-per-token clamped.
func TestAPlantedDeclaredWindowIsCappedAndTheCalibrationClamped(t *testing.T) {
	gw := newFakeGateway(t, 1<<40, http.StatusInternalServerError)
	c := model
	c.Declared, c.Served = 1<<23, 1<<23
	src := newFakeSources(gw.srv.URL, c)
	p := probeOf(src, true)
	r := runner(t, newFakePool("org/m"), p)
	r.SetEnabled(true)
	waitFor(t, "a measurement", func() bool { return src.result("org/m") != nil })
	for _, prompt := range gw.seen() {
		if len(prompt) > MaxProbeWindow*maxCharsPerToken {
			t.Fatalf("a prompt of %d characters was built", len(prompt))
		}
	}
	if m := src.result("org/m"); m.Window > MaxProbeWindow {
		t.Errorf("window = %d beyond the ceiling", m.Window)
	}
}

// A queued model that stops being a candidate — its load failed and the
// record on it stands, it was deleted, it is no longer offered to chat — is
// dropped from the queue rather than kept there forever holding the idle
// loop on; a hand retry queues it afresh (iss-2609211334570516).
func TestAQueuedModelThatIsNoLongerACandidateIsDropped(t *testing.T) {
	src := newFakeSources("http://127.0.0.1:1", model)
	p := probeOf(src, false)
	p.MeasureNow("org/m")
	if due := p.Due([]string{"org/m"}, time.Now()); due != "org/m" {
		t.Fatalf("Due = %q after Measure now", due)
	}
	src.mu.Lock()
	src.cands = nil
	src.mu.Unlock()
	if due := p.Due([]string{"org/m"}, time.Now()); due != "" {
		t.Errorf("Due = %q for a model that is no longer a candidate", due)
	}
	if q := p.Queued(); len(q) != 0 {
		t.Errorf("the queue still holds %v", q)
	}
}
