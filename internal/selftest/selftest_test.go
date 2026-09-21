package selftest

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/intentdriven/Dessau/internal/config"
	"github.com/intentdriven/Dessau/internal/mlxtest"
)

// fakeServer is a pool with fake model servers behind it. A model is resident
// once acquired until unloaded; every acquisition is one in-flight request
// until released, as in the real pool.
type fakeServer struct {
	t     *testing.T
	fakes map[string]*mlxtest.Server

	mu          sync.Mutex
	ready       []string
	resident    map[string]bool
	inFlight    map[string]int
	lastUsed    map[string]time.Time
	waiting     int
	downloading int
	// foreign is an in-flight request the self-test is not holding, on the
	// model named, standing in for a client's.
	foreign     string
	acquired    []string
	unloaded    []string
	concurrency int
	failLoad    string
	now         func() time.Time
	// acquireDelay is how long a load takes, outside the lock and cancellable,
	// as the pool's is.
	acquireDelay time.Duration
	// noFit names models that would need an eviction to load.
	noFit map[string]bool
	// refusals is the pool's count of loads refused for want of room.
	refusals uint64
	// maxInFlight is the most acquisitions held on any model at once.
	maxInFlight int
	// yield is the last hold's way of being told to let go (YieldFrom), which
	// the real adapter hands to the pool as a soft hold; calling it is the
	// pool preempting the run.
	yield func()
}

// lastYield is how the pool would tell the run to let go of the model, or nil
// before the run has taken a hold.
func (s *fakeServer) lastYield() func() {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.yield
}

func newFakeServer(t *testing.T, models ...string) *fakeServer {
	t.Helper()
	s := &fakeServer{
		t:           t,
		fakes:       map[string]*mlxtest.Server{},
		ready:       models,
		resident:    map[string]bool{},
		inFlight:    map[string]int{},
		lastUsed:    map[string]time.Time{},
		concurrency: 2,
		now:         time.Now,
		noFit:       map[string]bool{},
	}
	for _, m := range models {
		fake := mlxtest.Start(mlxtest.Options{ModelArg: "/models/" + m, Reply: "one two three four five six"})
		t.Cleanup(fake.Close)
		s.fakes[m] = fake
	}
	return s
}

func (s *fakeServer) Ready() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.ready...)
}

func (s *fakeServer) Acquire(ctx context.Context, id string) (Upstream, func(), error) {
	s.mu.Lock()
	s.yield = YieldFrom(ctx)
	// acquired records loads — a model brought in — not every hold: the
	// parallel test takes further holds on a model already resident.
	if !s.resident[id] {
		s.acquired = append(s.acquired, id)
	}
	if id == s.failLoad {
		s.mu.Unlock()
		return Upstream{}, nil, errors.New("simulated launch failure")
	}
	fake, ok := s.fakes[id]
	if !ok {
		s.mu.Unlock()
		return Upstream{}, nil, errors.New("unknown model")
	}
	// A loading entry is in flight from the moment the load starts, as the
	// pool's is, and a cancelled load is torn down.
	s.resident[id] = true
	s.inFlight[id]++
	if s.inFlight[id] > s.maxInFlight {
		s.maxInFlight = s.inFlight[id]
	}
	delay := s.acquireDelay
	s.mu.Unlock()
	if delay > 0 {
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			s.mu.Lock()
			s.inFlight[id]--
			delete(s.resident, id)
			s.mu.Unlock()
			return Upstream{}, nil, ctx.Err()
		}
	}
	s.mu.Lock()
	s.lastUsed[id] = s.now()
	s.mu.Unlock()
	var once sync.Once
	release := func() {
		once.Do(func() {
			s.mu.Lock()
			s.inFlight[id]--
			s.lastUsed[id] = s.now()
			s.mu.Unlock()
		})
	}
	return Upstream{BaseURL: fake.URL(), ModelArg: fake.ModelArg}, release, nil
}

func (s *fakeServer) Activity() Activity {
	s.mu.Lock()
	defer s.mu.Unlock()
	act := Activity{Waiting: s.waiting, Downloading: s.downloading, Refusals: s.refusals}
	for id := range s.resident {
		n := s.inFlight[id]
		if s.foreign == id {
			n++
		}
		act.Models = append(act.Models, ModelActivity{RepoID: id, InFlight: n, LastUsed: s.lastUsed[id]})
	}
	return act
}

func (s *fakeServer) Unload(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.inFlight[id] > 0 {
		return errors.New("busy")
	}
	delete(s.resident, id)
	s.unloaded = append(s.unloaded, id)
	return nil
}

func (s *fakeServer) Concurrency() int { return s.concurrency }

func (s *fakeServer) Fits(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return !s.noFit[id]
}

func (s *fakeServer) snapshot() (acquired, unloaded []string, inFlight map[string]int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	inFlight = map[string]int{}
	for k, v := range s.inFlight {
		inFlight[k] = v
	}
	return append([]string(nil), s.acquired...), append([]string(nil), s.unloaded...), inFlight
}

// fastRunner is a Runner on a test cadence: ticks and polls in milliseconds,
// a quiet period of nothing, so a test waits on results rather than clocks.
func fastRunner(t *testing.T, srv Server, dir string) *Runner {
	t.Helper()
	r := New(Options{
		Server: srv,
		Path:   filepath.Join(dir, FileName),
		Tick:   5 * time.Millisecond,
		Poll:   2 * time.Millisecond,
		Quiet:  time.Nanosecond,
	})
	t.Cleanup(r.Close)
	return r
}

// waitFor polls until the condition holds or the test's patience runs out.
func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(waitBound(t))
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("gave up waiting for %s", what)
}

// waitBound is how long a wait is given.
//
// It is the test binary's own deadline less a margin, rather than a fixed few
// seconds. Every condition waited on here is one the loop reaches in
// milliseconds on an idle Mac, so the bound is not a measurement of anything:
// it exists to turn a wedged loop into a named failure instead of a hung
// binary. A shared build runner under load can take an order of magnitude
// longer than a desktop one to schedule the same goroutine, and a bound set
// for the desktop reports that runner as a broken loop (iss-2609181019447825).
// Tying it to the deadline the binary was given keeps the failure this test's
// message rather than the timeout panic, and gives a loaded runner everything
// the run itself has.
//
// The margin is what leaves room for the failure to be printed and for the
// other tests in the binary to be reached; with no deadline set — go test
// always sets one, but a binary run by hand need not — a generous fixed bound
// stands in.
func waitBound(t *testing.T) time.Duration {
	t.Helper()
	const (
		noDeadline = 60 * time.Second
		margin     = 30 * time.Second
		floor      = 5 * time.Second
	)
	deadline, ok := t.Deadline()
	if !ok {
		return noDeadline
	}
	left := time.Until(deadline)
	if bound := left - margin; bound > floor {
		return bound
	}
	// A binary given a short timeout gets the floor, or half of what is left
	// if that is less: a wait that runs to the deadline turns a named failure
	// back into the timeout panic it exists to avoid, and half leaves the
	// other half for the failure and for the tests after this one
	// (iss-2609181119347301).
	return min(floor, left/2)
}

// stubJob is a job that wants one named model and never gets to run.
type stubJob struct {
	due   string
	runs  atomic.Int64
	parks bool
}

func (j *stubJob) Name() string                       { return "stub" }
func (j *stubJob) Due(_ []string, _ time.Time) string { return j.due }
func (j *stubJob) Run(_ *Session, _ string)           { j.runs.Add(1) }
func (j *stubJob) Parks() bool                        { return j.parks }

// A queued job whose model does not fit says so, rather than going quiet.
//
// The loop asks each job what is due and then asks the pool whether that model
// fits beside what is resident. A model that does not fit used to be passed
// over with nothing written down — no held_by, no due, no line in the log —
// while the panel went on showing it queued for a run that was never going to
// start. The manual check that found this read "job null, held_by null" for
// three hours and forty minutes against a model the queue still held
// (iss-2609161712555136).
func TestAQueuedJobWhoseModelDoesNotFitIsHeldRatherThanDropped(t *testing.T) {
	srv := newFakeServer(t, "org/a")
	srv.mu.Lock()
	srv.noFit["org/a"] = true
	srv.mu.Unlock()
	job := &stubJob{due: "org/a"}
	lines := &recordingHandler{}
	r := New(Options{
		Server: srv, Path: filepath.Join(t.TempDir(), FileName),
		Tick: 5 * time.Millisecond, Poll: 2 * time.Millisecond, Quiet: time.Nanosecond,
		Jobs: []Job{job}, SelfTest: func() bool { return false },
		Log: slog.New(lines),
	})
	t.Cleanup(r.Close)
	r.SetEnabled(true)

	waitFor(t, "the queued model to be reported as held for want of room", func() bool {
		st := r.Status()
		return st.HeldBy == HeldByNoRoom && st.Due == "org/a"
	})
	if got := job.runs.Load(); got != 0 {
		t.Errorf("the job ran %d times on a model that does not fit", got)
	}
	if acquired, _, _ := srv.snapshot(); len(acquired) != 0 {
		t.Errorf("acquired %v; a model that does not fit is not loaded", acquired)
	}
	// Said once, however many ticks pass: the hold is a standing condition,
	// and a line per tick would fill the log for as long as it lasts.
	time.Sleep(50 * time.Millisecond)
	if got := lines.count("does not fit"); got != 1 {
		t.Errorf("the log says the measurement is held %d times, want once: %v", got, lines.lines())
	}
}

// The hold belongs to the queued model, not to the tick.
//
// A Mac with work of its own has a model due on most ticks — after a restart,
// every model is — and the hold used to be cleared by any run that went ahead.
// The queued model was then invisible for as long as there was other work, and
// the log line was never said either (iss-2609181119346098).
func TestAQueuedJobWithNoRoomIsReportedWhileAnotherModelIsMeasured(t *testing.T) {
	srv := newFakeServer(t, "org/a", "org/b")
	srv.mu.Lock()
	srv.noFit["org/a"] = true // the queued model has nowhere to go
	srv.resident["org/b"] = true
	srv.mu.Unlock()
	job := &stubJob{due: "org/a"}
	lines := &recordingHandler{}
	r := New(Options{
		Server: srv, Path: filepath.Join(t.TempDir(), FileName),
		Tick: 5 * time.Millisecond, Poll: 2 * time.Millisecond, Quiet: time.Nanosecond,
		// Measured again at once, so there is a run to go ahead on every tick.
		Retest: time.Millisecond,
		Jobs:   []Job{job},
		Log:    slog.New(lines),
	})
	t.Cleanup(r.Close)
	r.SetEnabled(true)

	waitFor(t, "the other model to be measured", func() bool { return len(runsIn(t, r)) >= 2 })
	waitFor(t, "the queued model to be held for want of room meanwhile", func() bool {
		st := r.Status()
		return st.HeldBy == HeldByNoRoom && st.Due == "org/a"
	})
	if got := lines.count("does not fit"); got != 1 {
		t.Errorf("the log says the measurement is held %d times, want once", got)
	}
}

// recordingHandler keeps what the loop said, so a test can hold the log to one
// line per model rather than one per tick.
type recordingHandler struct {
	mu   sync.Mutex
	said []string
}

func (h *recordingHandler) Enabled(context.Context, slog.Level) bool { return true }
func (h *recordingHandler) WithAttrs([]slog.Attr) slog.Handler       { return h }
func (h *recordingHandler) WithGroup(string) slog.Handler            { return h }

func (h *recordingHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.said = append(h.said, r.Message)
	return nil
}

func (h *recordingHandler) lines() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]string(nil), h.said...)
}

func (h *recordingHandler) count(substr string) int {
	n := 0
	for _, l := range h.lines() {
		if strings.Contains(l, substr) {
			n++
		}
	}
	return n
}

// A wait is as patient as the run it is part of.
//
// The bound this asserts is not about the loop at all: it is about the machine
// the loop is being watched on. A fixed few seconds says a build runner under
// load, which schedules the same goroutine an order of magnitude later than a
// desktop Mac does, is a broken loop (iss-2609181019447825). The binary's own
// deadline is the honest bound, and it is what every wait here is given.
func TestTheWaitBoundFollowsTheTestBinarysOwnDeadline(t *testing.T) {
	deadline, ok := t.Deadline()
	if !ok {
		t.Skip("this binary was given no timeout, so there is no deadline to follow")
	}
	// The bound first, the time left second: both are read from a clock that
	// is moving, and taking them the other way round would compare a bound
	// against a deadline that had not yet run down to it.
	got := waitBound(t)
	left := time.Until(deadline)
	if got >= left {
		t.Errorf("a wait is given %v of the %v this binary has left, which spends the deadline the failure itself needs", got, left)
	}
	if want := min(5*time.Second, left/2); got < want {
		t.Errorf("a wait is given %v, want at least %v — the fixed bound it replaced, or half of what is left of the deadline", got, want)
	}
	// Under the timeout `go test` gives a binary by default this is minutes,
	// not the five seconds a loaded runner outran.
	if left > 2*time.Minute && got <= 5*time.Second {
		t.Errorf("with %v of the deadline left a wait is given %v; a loaded runner is allowed no more than a desktop one was", left, got)
	}
}

func runsIn(t *testing.T, r *Runner) []Run {
	t.Helper()
	runs, err := ReadResults(r.opts.Path)
	if err != nil {
		t.Fatal(err)
	}
	return runs
}

// Off is the default, and off is nothing: no model loaded, no file written,
// no directory created.
func TestOffLoadsNothingAndWritesNothing(t *testing.T) {
	srv := newFakeServer(t, "org/a")
	dir := t.TempDir()
	r := fastRunner(t, srv, dir)
	r.SetEnabled(false) // already off: a no-op, not a hang
	time.Sleep(30 * time.Millisecond)
	if acquired, _, _ := srv.snapshot(); len(acquired) != 0 {
		t.Errorf("the self-test loaded %v while off", acquired)
	}
	if _, err := os.Stat(filepath.Join(dir, FileName)); !os.IsNotExist(err) {
		t.Errorf("a results file exists while off (%v)", err)
	}
	if r.Enabled() {
		t.Error("Enabled() while off")
	}
}

// On, and idle: every ready model is acquired through the pool, gets the
// standard set, is written as one line, and is unloaded again because the
// self-test was what loaded it.
func TestAnIdleMacGetsEveryModelMeasuredAndLeftAsFound(t *testing.T) {
	srv := newFakeServer(t, "org/a", "org/b")
	r := fastRunner(t, srv, t.TempDir())
	r.SetEnabled(true)
	if !r.Enabled() {
		t.Fatal("Enabled() is false after SetEnabled(true)")
	}
	waitFor(t, "two runs", func() bool { return len(runsIn(t, r)) == 2 })

	acquired, unloaded, inFlight := srv.snapshot()
	if len(acquired) != 2 || len(unloaded) != 2 {
		t.Errorf("acquired %v, unloaded %v; want both models once each", acquired, unloaded)
	}
	for id, n := range inFlight {
		if n != 0 {
			t.Errorf("%s still has %d in flight after its run: the hold was not released", id, n)
		}
	}
	for _, run := range runsIn(t, r) {
		if run.Outcome != OutcomeOK || run.Kind != KindRun || !run.ColdLoad {
			t.Errorf("run = %+v; want an ok, cold run", run)
		}
		names := []string{}
		for _, test := range run.Tests {
			names = append(names, test.Name)
			if test.Counted != CountedUsage {
				t.Errorf("%s counted by %q; the fake sends usage", test.Name, test.Counted)
			}
			if test.PromptTokens == 0 || test.CompletionTokens == 0 || test.TotalMs < 0 {
				t.Errorf("%s = %+v; want the server's counts", test.Name, test)
			}
		}
		// Fatal, not an error: the checks below index the set this names, and
		// a run cut short would panic on them rather than say what went wrong.
		if got, want := strings.Join(names, ","), "pp512,tg128,tg128x2"; got != want {
			t.Fatalf("tests = %s, want %s (run = %+v)", got, want, run)
		}
		if run.Tests[2].Parallel != 2 || run.Tests[2].CompletionTokens != 2*run.Tests[1].CompletionTokens {
			t.Errorf("the parallel test = %+v; want two requests' counts added up", run.Tests[2])
		}
	}
	// Once every model has a result, the loop is quiet.
	time.Sleep(30 * time.Millisecond)
	if acquired, _, _ := srv.snapshot(); len(acquired) != 2 {
		t.Errorf("acquired %v; a model with a fresh result was measured again", acquired)
	}
}

// A model that was in memory before the run stays there afterwards: the
// self-test leaves residency as it found it.
func TestAModelThatWasResidentIsLeftResident(t *testing.T) {
	srv := newFakeServer(t, "org/a")
	srv.mu.Lock()
	srv.resident["org/a"] = true
	srv.mu.Unlock()
	r := fastRunner(t, srv, t.TempDir())
	r.SetEnabled(true)
	waitFor(t, "one run", func() bool { return len(runsIn(t, r)) == 1 })
	if _, unloaded, _ := srv.snapshot(); len(unloaded) != 0 {
		t.Errorf("unloaded %v, which was resident before the self-test ran", unloaded)
	}
	if run := runsIn(t, r)[0]; run.ColdLoad {
		t.Error("a resident model was recorded as a cold load")
	}
}

// A client's request during a run ends it: the self-test's own request is
// cancelled, the run is recorded as yielded with what it had, and the model
// is released for the client.
func TestAClientRequestYieldsTheRun(t *testing.T) {
	srv := newFakeServer(t, "org/a", "org/b")
	// Slow enough that a run is in progress when the client arrives.
	for _, fake := range srv.fakes {
		fake.ChunkDelay = 20 * time.Millisecond
		fake.Reply = strings.Repeat("word ", 50)
	}
	r := fastRunner(t, srv, t.TempDir())
	r.SetEnabled(true)
	waitFor(t, "a run to start", func() bool { acquired, _, _ := srv.snapshot(); return len(acquired) == 1 })
	// The client asks for the other model.
	srv.mu.Lock()
	srv.foreign = "org/b"
	srv.resident["org/b"] = true
	srv.mu.Unlock()
	arrived := time.Now()
	waitFor(t, "the yielded run", func() bool { return len(runsIn(t, r)) >= 1 })
	// The stream would take a second to finish; yielding does not wait for it.
	if took := time.Since(arrived); took > 500*time.Millisecond {
		t.Errorf("the run took %v to yield after the client arrived; a real request would have waited that long", took)
	}
	run := runsIn(t, r)[0]
	if run.Outcome != OutcomeYielded {
		t.Errorf("outcome = %q, want %q", run.Outcome, OutcomeYielded)
	}
	if _, _, inFlight := srv.snapshot(); inFlight[run.Model] != 0 {
		t.Errorf("the self-test still holds %s after yielding", run.Model)
	}
	// And it stays out of the way while the client is there.
	time.Sleep(30 * time.Millisecond)
	if acquired, _, _ := srv.snapshot(); len(acquired) != 1 {
		t.Errorf("acquired %v while a client request is in flight", acquired)
	}
}

// Nothing starts while the pool is doing anything else: a request in
// flight, a load waiting for room, a download running, or a request that
// finished only a moment ago.
func TestNothingRunsWhileTheMacIsBusy(t *testing.T) {
	for _, tt := range []struct {
		name string
		set  func(*fakeServer)
	}{
		{"a request in flight", func(s *fakeServer) { s.resident["org/b"] = true; s.foreign = "org/b" }},
		{"a load waiting", func(s *fakeServer) { s.waiting = 1 }},
		{"a download running", func(s *fakeServer) { s.downloading = 1 }},
		{"a recent request", func(s *fakeServer) { s.resident["org/b"] = true; s.lastUsed["org/b"] = time.Now() }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			srv := newFakeServer(t, "org/a", "org/b")
			srv.mu.Lock()
			tt.set(srv)
			srv.mu.Unlock()
			r := New(Options{
				Server: srv, Path: filepath.Join(t.TempDir(), FileName),
				Tick: 5 * time.Millisecond, Poll: 2 * time.Millisecond, Quiet: time.Hour,
			})
			t.Cleanup(r.Close)
			r.SetEnabled(true)
			time.Sleep(40 * time.Millisecond)
			if acquired, _, _ := srv.snapshot(); len(acquired) != 0 {
				t.Errorf("acquired %v with %s", acquired, tt.name)
			}
		})
	}
}

// A result stands for the re-test window, and the model measured longest ago
// goes first. The file is what remembers this across a restart.
func TestAResultStandsForADayAndTheStalestModelGoesFirst(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	earlier := func(d time.Duration) int64 { return now.Add(-d).Unix() }
	seed := []Run{
		{Kind: KindRun, Model: "org/a", At: earlier(time.Hour), Outcome: OutcomeOK},
		{Kind: KindRun, Model: "org/b", At: earlier(30 * time.Hour), Outcome: OutcomeOK},
		{Kind: KindRun, Model: "org/c", At: earlier(40 * time.Hour), Outcome: OutcomeOK},
	}
	f := &file{path: filepath.Join(dir, FileName), maxBytes: DefaultMaxBytes, log: testLogger()}
	for _, run := range seed {
		f.write(run)
	}

	srv := newFakeServer(t, "org/a", "org/b", "org/c")
	r := New(Options{
		Server: srv, Path: filepath.Join(dir, FileName),
		Tick: 5 * time.Millisecond, Poll: 2 * time.Millisecond, Quiet: time.Nanosecond,
		Retest: 24 * time.Hour, Now: func() time.Time { return now },
	})
	t.Cleanup(r.Close)
	r.SetEnabled(true)
	waitFor(t, "two new runs", func() bool { return len(runsIn(t, r)) == 5 })
	acquired, _, _ := srv.snapshot()
	if got, want := strings.Join(acquired, ","), "org/c,org/b"; got != want {
		t.Errorf("acquired %s, want %s: stalest first, and not the one measured an hour ago", got, want)
	}
}

// The switch going off mid-run cancels the request, releases the model, and
// records the run as stopped; SetEnabled returns once the loop is gone.
func TestSwitchingOffStopsARunAndReleasesTheModel(t *testing.T) {
	srv := newFakeServer(t, "org/a")
	srv.fakes["org/a"].ChunkDelay = 20 * time.Millisecond
	srv.fakes["org/a"].Reply = strings.Repeat("word ", 50)
	r := fastRunner(t, srv, t.TempDir())
	r.SetEnabled(true)
	waitFor(t, "a run to start", func() bool { acquired, _, _ := srv.snapshot(); return len(acquired) == 1 })
	r.SetEnabled(false)
	if r.Enabled() {
		t.Fatal("Enabled() after SetEnabled(false)")
	}
	runs := runsIn(t, r)
	if len(runs) != 1 || runs[0].Outcome != OutcomeStopped {
		t.Fatalf("runs = %+v, want one stopped run", runs)
	}
	if _, _, inFlight := srv.snapshot(); inFlight["org/a"] != 0 {
		t.Error("the model is still held after the switch went off")
	}
	// A stopped run is the switch's doing, not the model's: on again, the
	// model is measured, not skipped for a day.
	r.SetEnabled(true)
	waitFor(t, "the run after the stop", func() bool { return len(runsIn(t, r)) == 2 })
	if runs := runsIn(t, r); runs[1].Outcome != OutcomeOK {
		t.Errorf("the run after the stop = %+v, want ok", runs[1])
	}
}

// A model that cannot load is recorded as failed with a class, not with the
// error's text, and the loop moves on to the next model.
func TestALoadThatFailsIsRecordedByClassAndTheLoopMovesOn(t *testing.T) {
	srv := newFakeServer(t, "org/a", "org/b")
	srv.failLoad = "org/a"
	r := fastRunner(t, srv, t.TempDir())
	r.SetEnabled(true)
	waitFor(t, "two runs", func() bool { return len(runsIn(t, r)) == 2 })
	for _, run := range runsIn(t, r) {
		switch run.Model {
		case "org/a":
			if run.Outcome != OutcomeFailed || run.Reason != ReasonLoad {
				t.Errorf("org/a = %+v, want failed/load", run)
			}
		case "org/b":
			if run.Outcome != OutcomeOK {
				t.Errorf("org/b = %+v, want ok", run)
			}
		}
	}
	raw, _ := os.ReadFile(r.opts.Path)
	if strings.Contains(string(raw), "simulated") {
		t.Errorf("the error's text reached the file:\n%s", raw)
	}
}

// The file is this account's own and bounded: 0600 in a 0700 directory,
// started again when the next line would pass the cap, newest line kept.
func TestTheResultsFileIsPrivateAndBounded(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "selftest")
	path := filepath.Join(dir, FileName)
	f := &file{path: path, maxBytes: 300, log: testLogger()}
	for i := 0; i < 10; i++ {
		f.write(Run{Kind: KindRun, Model: "org/a", At: int64(i), Outcome: OutcomeOK})
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Errorf("file mode %o, want 0600", st.Mode().Perm())
	}
	dst, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if dst.Mode().Perm() != 0o700 {
		t.Errorf("directory mode %o, want 0700", dst.Mode().Perm())
	}
	if st.Size() > 300 {
		t.Errorf("file is %d bytes, over its cap of 300", st.Size())
	}
	runs, err := ReadResults(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) == 0 || runs[len(runs)-1].At != 9 {
		t.Errorf("runs = %+v; want the newest kept", runs)
	}
}

// A result is figures. Neither prompt reaches the file, and nothing of an
// answer does either.
func TestAResultCarriesNoPromptAndNoAnswer(t *testing.T) {
	srv := newFakeServer(t, "org/a")
	srv.fakes["org/a"].Reply = "SECRET ANSWER TEXT"
	r := fastRunner(t, srv, t.TempDir())
	r.SetEnabled(true)
	waitFor(t, "one run", func() bool { return len(runsIn(t, r)) == 1 })
	raw, err := os.ReadFile(r.opts.Path)
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"SECRET", "lighthouse", "keeper"} {
		if strings.Contains(string(raw), text) {
			t.Errorf("the file carries %q:\n%s", text, raw)
		}
	}
	var line map[string]any
	if err := json.Unmarshal(raw[:len(raw)-1], &line); err != nil {
		t.Fatal(err)
	}
	for _, field := range RunFields() {
		if field == "reason" {
			continue // omitted from an ok run
		}
		if _, ok := line[field]; !ok {
			t.Errorf("a line does not carry %q", field)
		}
	}
}

// The counts come from the server's usage event when it sends one, and from
// the chunks — one a token on mlx-lm — when it does not, and the result says
// which.
func TestTheStreamIsTimedAndCountedEitherWay(t *testing.T) {
	body := "data: {\"choices\":[{\"delta\":{\"role\":\"assistant\"}}]}\n\n" +
		"data: {\"choices\":[{\"delta\":{\"content\":\"a\"}}]}\n\n" +
		"data: {\"choices\":[{\"delta\":{\"content\":\"b\"}}]}\n\n" +
		"data: {\"choices\":[{\"delta\":{\"content\":\"c\"}}]}\n\n" +
		"data: [DONE]\n\n"
	m, err := readStream(strings.NewReader(body), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if m.CompletionTokens != 3 || m.Counted != CountedChunks {
		t.Errorf("without usage: %+v, want 3 chunks counted", m)
	}
	withUsage := strings.Replace(body, "data: [DONE]",
		"data: {\"choices\":[],\"usage\":{\"prompt_tokens\":40,\"completion_tokens\":3}}\n\ndata: [DONE]", 1)
	m, err = readStream(strings.NewReader(withUsage), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if m.PromptTokens != 40 || m.CompletionTokens != 3 || m.Counted != CountedUsage {
		t.Errorf("with usage: %+v, want the server's counts", m)
	}
	if _, err := readStream(strings.NewReader("data: {\"choices\":[]}\n\n"), time.Now()); err == nil {
		t.Error("a stream that ends without [DONE] was accepted")
	}
}

// The parallel test is the batch's figures: counts added, first token the
// mean, the rate every token over the batch's wall time.
func TestTheParallelTestIsTheBatchsFigures(t *testing.T) {
	ms := []measurement{
		{PromptTokens: 10, CompletionTokens: 100, FirstToken: 100 * time.Millisecond, Total: time.Second, Counted: CountedUsage},
		{PromptTokens: 10, CompletionTokens: 100, FirstToken: 300 * time.Millisecond, Total: time.Second, Counted: CountedUsage},
	}
	got := aggregate("tg128x2", ms, 2*time.Second)
	if got.Parallel != 2 || got.PromptTokens != 20 || got.CompletionTokens != 200 ||
		got.FirstTokenMs != 200 || got.TotalMs != 2000 || got.TokensPerSec != 100 {
		t.Errorf("aggregate = %+v", got)
	}
	single := ms[0].test("tg128", 1)
	if single.PromptTokensPerSec != 100 || single.TokensPerSec != 110 {
		t.Errorf("single = %+v; want 10 prompt tokens over 0.1 s and 99 tokens over 0.9 s", single)
	}
}

// The standard set is the same for every model and its names are
// llama-bench's, with the concurrent test only when there is concurrency.
func TestTheStandardSetIsFixedAndNamedAfterLlamaBench(t *testing.T) {
	names := func(set []testSpec) string {
		out := []string{}
		for _, s := range set {
			out = append(out, s.Name)
		}
		return strings.Join(out, ",")
	}
	if got := names(standardSet(4)); got != "pp512,tg128,tg128x4" {
		t.Errorf("set at concurrency 4 = %s", got)
	}
	if got := names(standardSet(1)); got != "pp512,tg128" {
		t.Errorf("set at concurrency 1 = %s", got)
	}
	if n := len(promptProcessingPrompt()); n < promptProcessingChars || n > promptProcessingChars+len(promptSentence) {
		t.Errorf("the pp512 prompt is %d characters", n)
	}
}

// The fold rule is the repository's: a model's result is found however its
// id was spelled.
func TestResultsAreKeyedOnTheFoldedId(t *testing.T) {
	act := Activity{Models: []ModelActivity{{RepoID: "Org/A", InFlight: 1}}}
	one := hold{model: "org/a", count: 1}
	if busy(act, one) {
		t.Error("the self-test's own hold, spelled differently, counted as a client's")
	}
	act.Models[0].InFlight = 2
	if !busy(act, one) {
		t.Error("a client on the model under test was not seen")
	}
	// The parallel test holds the concurrency's worth; a client beyond that
	// is queued in the pool and seen.
	act.Models[0].InFlight = 4
	if busy(act, hold{model: "org/a", count: 4}) {
		t.Error("the parallel test's own holds counted as clients")
	}
	act.Models[0].InFlight = 5
	if !busy(act, hold{model: "org/a", count: 4}) {
		t.Error("a client queued behind the parallel test was not seen")
	}
	// While the self-test's own load is parked, the waiter the pool counts
	// may be its own, and is not read; a client in flight elsewhere is, and
	// so is a client refused room since the run began. The load's own place in
	// flight is claimed before Acquire is called, so the hold carries it
	// throughout (iss-2609190018027384).
	loading := hold{model: "org/a", count: 1, refusals: 3, parks: true, loading: true}
	if busy(Activity{Waiting: 1}, loading) {
		t.Error("the self-test's own parked load counted as a client's")
	}
	if busy(Activity{Models: []ModelActivity{{RepoID: "org/a", InFlight: 1}}}, loading) {
		t.Error("the self-test's own load in flight counted as a client's")
	}
	if !busy(Activity{Models: []ModelActivity{{RepoID: "org/a", InFlight: 2}}}, loading) {
		t.Error("a client on the model the self-test is loading was not seen")
	}
	// Once that load has returned, a waiter can only be a client's.
	if !busy(Activity{Waiting: 1}, hold{model: "org/a", count: 1, parks: true}) {
		t.Error("a waiter during a run whose own load is done was not seen")
	}
	if !busy(Activity{Models: []ModelActivity{{RepoID: "org/b", InFlight: 1}}}, loading) {
		t.Error("a client in flight during the load was not seen")
	}
	if !busy(Activity{Refusals: 4}, loading) {
		t.Error("a client refused room was not seen")
	}
	if busy(Activity{Refusals: 3}, loading) {
		t.Error("refusals from before the run counted as a client")
	}
	// A job that never parks has no waiter of its own: every waiter is a
	// client's, whether or not the job holds anything.
	if !busy(Activity{Waiting: 1}, hold{model: "org/a"}) {
		t.Error("a waiter was not seen as a client for a job that never parks")
	}
	if config.FoldRepoID("Org/A") != config.FoldRepoID("org/a") {
		t.Fatal("the fold rule changed under this test")
	}
}

func testLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// The load is the longest thing a run does, and a client is not made to wait
// out a cold load either: a request arriving while the self-test's model is
// still loading cancels the load, and the pool tears it down.
func TestAClientRequestDuringTheLoadYieldsToo(t *testing.T) {
	srv := newFakeServer(t, "org/a", "org/b")
	srv.acquireDelay = 2 * time.Second
	r := fastRunner(t, srv, t.TempDir())
	r.SetEnabled(true)
	waitFor(t, "a load to start", func() bool { acquired, _, _ := srv.snapshot(); return len(acquired) == 1 })
	srv.mu.Lock()
	srv.foreign = "org/b"
	srv.resident["org/b"] = true
	srv.mu.Unlock()
	arrived := time.Now()
	waitFor(t, "the yielded run", func() bool { return len(runsIn(t, r)) >= 1 })
	if took := time.Since(arrived); took > 500*time.Millisecond {
		t.Errorf("the load took %v to yield; the client waited that long", took)
	}
	run := runsIn(t, r)[0]
	if run.Outcome != OutcomeYielded || len(run.Tests) != 0 {
		t.Errorf("run = %+v, want yielded with no tests", run)
	}
	if _, _, inFlight := srv.snapshot(); inFlight["org/a"] != 0 {
		t.Error("the abandoned load is still held")
	}
}

// The self-test's own run does not look like a client's afterwards: a model it
// has just released is not what keeps the next tick from starting.
func TestTheSelfTestsOwnRunDoesNotCountAsARecentRequest(t *testing.T) {
	srv := newFakeServer(t, "org/a", "org/b")
	srv.mu.Lock()
	srv.resident["org/a"] = true
	srv.resident["org/b"] = true
	srv.mu.Unlock()
	r := New(Options{
		Server: srv, Path: filepath.Join(t.TempDir(), FileName),
		Tick: 5 * time.Millisecond, Poll: 2 * time.Millisecond, Quiet: time.Hour,
	})
	t.Cleanup(r.Close)
	r.SetEnabled(true)
	waitFor(t, "both resident models measured despite a one-hour quiet period", func() bool { return len(runsIn(t, r)) == 2 })
}

// With eviction grace off, a client whose load finds no room is refused at
// once and waits nowhere the self-test can see; the pool's refusal count is
// the sign it was there, and it ends the run so the client's retry finds the
// room.
func TestAClientRefusedRoomYieldsTheRun(t *testing.T) {
	srv := newFakeServer(t, "org/a")
	srv.fakes["org/a"].ChunkDelay = 20 * time.Millisecond
	srv.fakes["org/a"].Reply = strings.Repeat("word ", 50)
	r := fastRunner(t, srv, t.TempDir())
	r.SetEnabled(true)
	waitFor(t, "a run to start", func() bool { acquired, _, _ := srv.snapshot(); return len(acquired) == 1 })
	srv.mu.Lock()
	srv.refusals++
	srv.mu.Unlock()
	arrived := time.Now()
	waitFor(t, "the yielded run", func() bool { return len(runsIn(t, r)) >= 1 })
	if took := time.Since(arrived); took > 500*time.Millisecond {
		t.Errorf("the run took %v to yield after a refusal", took)
	}
	if run := runsIn(t, r)[0]; run.Outcome != OutcomeYielded {
		t.Errorf("outcome = %q, want %q", run.Outcome, OutcomeYielded)
	}
	if _, _, inFlight := srv.snapshot(); inFlight["org/a"] != 0 {
		t.Error("the model is still held after yielding to a refused client")
	}
}

// The pool may take the run's model back instead of refusing a client, and
// when it does it cancels the hold. The run treats that as the yield it
// already has: recorded as yielded, model released (iss-2609100526194406).
func TestThePoolTakingTheHoldBackYieldsTheRun(t *testing.T) {
	srv := newFakeServer(t, "org/a")
	srv.fakes["org/a"].ChunkDelay = 20 * time.Millisecond
	srv.fakes["org/a"].Reply = strings.Repeat("word ", 50)
	r := fastRunner(t, srv, t.TempDir())
	r.SetEnabled(true)
	waitFor(t, "a run to take a hold the pool can ask for back",
		func() bool { return srv.lastYield() != nil })

	arrived := time.Now()
	srv.lastYield()()

	waitFor(t, "the yielded run", func() bool { return len(runsIn(t, r)) >= 1 })
	if took := time.Since(arrived); took > 500*time.Millisecond {
		t.Errorf("the run took %v to let go after the pool asked", took)
	}
	if run := runsIn(t, r)[0]; run.Outcome != OutcomeYielded {
		t.Errorf("outcome = %q, want %q", run.Outcome, OutcomeYielded)
	}
	if _, _, inFlight := srv.snapshot(); inFlight["org/a"] != 0 {
		t.Error("the model is still held after the pool asked for it back")
	}
}

// The self-test never evicts: a model that would need room made is left for
// a tick when there is room, and a model that fits is measured meanwhile.
func TestAModelThatWouldNeedAnEvictionIsLeftForLater(t *testing.T) {
	srv := newFakeServer(t, "org/a", "org/b")
	srv.noFit["org/a"] = true
	r := fastRunner(t, srv, t.TempDir())
	r.SetEnabled(true)
	waitFor(t, "org/b measured", func() bool { return len(runsIn(t, r)) == 1 })
	time.Sleep(30 * time.Millisecond)
	if acquired, _, _ := srv.snapshot(); len(acquired) != 1 || acquired[0] != "org/b" {
		t.Fatalf("acquired %v; org/a does not fit and must not be loaded", acquired)
	}
	// Room appears; org/a is still due, not charged a day it never had.
	srv.mu.Lock()
	delete(srv.noFit, "org/a")
	srv.mu.Unlock()
	waitFor(t, "org/a measured once there is room", func() bool { return len(runsIn(t, r)) == 2 })
}

// The parallel test takes its places in the pool before it sends: the model
// server never decodes more than the pool allows, and a client arriving on the
// model waits in the pool where the watcher sees it.
func TestTheParallelTestHoldsItsPlacesInThePool(t *testing.T) {
	srv := newFakeServer(t, "org/a")
	srv.concurrency = 3
	r := fastRunner(t, srv, t.TempDir())
	r.SetEnabled(true)
	waitFor(t, "one run", func() bool { return len(runsIn(t, r)) == 1 })
	srv.mu.Lock()
	most := srv.maxInFlight
	srv.mu.Unlock()
	if most != 3 {
		t.Errorf("the most held at once was %d, want the concurrency, 3", most)
	}
	if _, _, inFlight := srv.snapshot(); inFlight["org/a"] != 0 {
		t.Error("holds were left behind after the run")
	}
	run := runsIn(t, r)[0]
	if run.Outcome != OutcomeOK || run.Tests[2].Name != "tg128x3" || run.Tests[2].Parallel != 3 {
		t.Errorf("run = %+v", run)
	}
}

// A planted file where the results go is refused rather than written into or
// waited on: a link, a FIFO, and a directory another account can write.
func TestAPlantedResultsLocationIsRefused(t *testing.T) {
	base := t.TempDir()
	f := func(dir string) *file {
		return &file{path: filepath.Join(dir, FileName), maxBytes: DefaultMaxBytes, log: testLogger()}
	}
	// A link where the directory should be.
	elsewhere := filepath.Join(base, "elsewhere")
	if err := os.Mkdir(elsewhere, 0o700); err != nil {
		t.Fatal(err)
	}
	linked := filepath.Join(base, "linked")
	if err := os.Symlink(elsewhere, linked); err != nil {
		t.Fatal(err)
	}
	f(linked).write(Run{Kind: KindRun, Model: "org/a", Outcome: OutcomeOK})
	if _, err := os.Stat(filepath.Join(elsewhere, FileName)); !os.IsNotExist(err) {
		t.Error("a result was written through a planted link")
	}
	// A directory others can write.
	open := filepath.Join(base, "open")
	if err := os.Mkdir(open, 0o777); err != nil {
		t.Fatal(err)
	}
	os.Chmod(open, 0o777)
	f(open).write(Run{Kind: KindRun, Model: "org/a", Outcome: OutcomeOK})
	if _, err := os.Stat(filepath.Join(open, FileName)); !os.IsNotExist(err) {
		t.Error("a result was written into a directory other accounts can write")
	}
	// A FIFO where the file should be: the write and the read both return.
	fifoDir := filepath.Join(base, "fifo")
	if err := os.Mkdir(fifoDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(filepath.Join(fifoDir, FileName), 0o600); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		f(fifoDir).write(Run{Kind: KindRun, Model: "org/a", Outcome: OutcomeOK})
		if _, err := ReadResults(filepath.Join(fifoDir, FileName)); err == nil {
			t.Error("reading a FIFO was accepted")
		}
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("the writer or the reader blocked on a FIFO")
	}
}

// The loop does not yield to its own load.
//
// The pool counts a load as in flight from the moment Acquire is called, and
// a cold load is the longest thing a run does. A hold raised only once
// Acquire had returned left the loop's own load indistinguishable from a
// client's request on the model under test: the watcher cancelled the run it
// was watching, and every cold-load run was recorded as yielded with no
// tests. The same window reopened at each extra acquisition the parallel test
// takes, which cut runs short at two tests on a loaded build runner
// (iss-2609190018027384).
func TestTheLoopDoesNotYieldToItsOwnAcquisitions(t *testing.T) {
	srv := newFakeServer(t, "org/a")
	// A load many polls long, as a cold load is, and nobody else on the Mac.
	srv.acquireDelay = 150 * time.Millisecond
	r := fastRunner(t, srv, t.TempDir())
	r.SetEnabled(true)
	waitFor(t, "the run", func() bool { return len(runsIn(t, r)) == 1 })
	run := runsIn(t, r)[0]
	if run.Outcome != OutcomeOK || len(run.Tests) != 3 {
		t.Fatalf("run = %+v; want an ok run with the whole set: nobody but the loop asked for the model", run)
	}
}

// blockingJob is a job that runs once, waits until its session is
// cancelled, and says how it ended.
type blockingJob struct {
	due     string
	ran     atomic.Bool
	started chan struct{}
	yielded atomic.Bool
	ended   chan struct{}
}

func (j *blockingJob) Name() string { return "blocking" }
func (j *blockingJob) Due(_ []string, _ time.Time) string {
	if j.ran.Load() {
		return ""
	}
	return j.due
}
func (j *blockingJob) Parks() bool { return false }
func (j *blockingJob) Run(s *Session, _ string) {
	if !j.ran.CompareAndSwap(false, true) {
		return
	}
	close(j.started)
	<-s.Ctx.Done()
	j.yielded.Store(s.Yielded())
	close(j.ended)
}

// Interrupt ends the run in progress on the model named, as a yield, and
// leaves a run on any other model alone; between runs there is nothing to
// interrupt (iss-2609211334576018).
func TestInterruptEndsTheRunOnThatModelAsAYield(t *testing.T) {
	srv := newFakeServer(t, "org/a")
	job := &blockingJob{due: "org/a", started: make(chan struct{}), ended: make(chan struct{})}
	r := New(Options{
		Server: srv, Path: filepath.Join(t.TempDir(), FileName),
		Tick: 5 * time.Millisecond, Poll: 2 * time.Millisecond, Quiet: time.Nanosecond,
		Jobs: []Job{job}, SelfTest: func() bool { return false },
		Log: slog.New(slog.DiscardHandler),
	})
	t.Cleanup(r.Close)
	if r.Interrupt("org/a") {
		t.Error("Interrupt found a run before the loop started")
	}
	r.SetEnabled(true)
	<-job.started
	if r.Interrupt("org/other") {
		t.Error("Interrupt on another model ended this run")
	}
	select {
	case <-job.ended:
		t.Fatal("the run ended without being interrupted")
	case <-time.After(30 * time.Millisecond):
	}
	if !r.Interrupt("Org/A") {
		t.Fatal("Interrupt did not find the run on its model")
	}
	select {
	case <-job.ended:
	case <-time.After(5 * time.Second):
		t.Fatal("the interrupted run did not end")
	}
	if !job.yielded.Load() {
		t.Error("the interrupted run did not read as a yield")
	}
}
