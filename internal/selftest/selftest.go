// Package selftest measures Gropius's own models while nobody is using them.
//
// When the operator switches it on (config.Config.SelfTest), a loop wakes
// once a minute and asks whether the Mac is idle: no request in flight on any
// model, nobody waiting for a load, no download running, and the last request
// older than a quiet period. When it is, the loop picks the ready model that
// was tested longest ago — never one tested within the last day — acquires it
// through the pool's ordinary path, runs the same short set of tests against
// it, appends one line of figures to a file, and unloads the model if the
// self-test was what brought it in. Then it goes back to sleep.
//
// The set is llama-bench's pair plus what the pool can tell: how long the load
// took, how fast a long prompt is read (pp512), how fast tokens come out
// (tg128), and what that becomes when the decode concurrency's worth of
// requests arrive at once. The figures are the ones the 2026-09-06 model-bench
// campaign decided the memory budget and the served window from, measured on
// this Mac rather than on somebody else's (itd-2609100457007827).
//
// A real request always wins. The pool has no preemption, so the self-test
// yields by watching the pool while a request of its own is in flight and
// cancelling that request the moment anyone else's appears; the run is
// recorded as yielded and the model is released. Nothing in a result is a
// prompt or an answer: the prompts are the constants in request.go, and the
// file holds counts and durations (adr-2609061503319212).
package selftest

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/intentdriven/Gropius/internal/config"
)

// Server is what the self-test needs from the app: the models it may test,
// the pool's ordinary way of loading one, the pool's view of what is going on,
// and a way to unload what the self-test itself loaded.
//
// It is an interface rather than the App so the loop can be tested against a
// fake pool with a fake model server behind it, and so that the package
// imports nothing that imports it back.
type Server interface {
	// Ready lists the models that can be loaded, by repo id.
	Ready() []string
	// Acquire loads the model if it is not resident and holds it until the
	// release function is called, exactly as a request does.
	Acquire(ctx context.Context, repoID string) (Upstream, func(), error)
	// Activity is what the pool is doing right now.
	Activity() Activity
	// Unload stops a model server. The pool refuses one that is serving.
	Unload(repoID string) error
	// Concurrency is the decode concurrency each model server runs with,
	// which is how many requests the concurrent test sends at once.
	Concurrency() int
	// Fits reports whether the model can be loaded beside what is resident
	// without evicting anything. The self-test never evicts: a Mac whose
	// working model stays warm all day would otherwise find it gone, and
	// pay a cold load, once a day per model the self-test measured.
	Fits(repoID string) bool
}

// Upstream is a ready model server: where it answers, and the exact string
// its "model" field must carry (runtime.Upstream.ModelArg).
type Upstream struct {
	BaseURL  string
	ModelArg string
}

// Activity is the pool's view of the moment: what is resident and how busy
// each model is, how many loads are waiting for room, and how many downloads
// are running.
type Activity struct {
	Models      []ModelActivity
	Waiting     int
	Downloading int
	// Refusals is how many loads the pool has refused for want of room, ever
	// (runtime.Pool.Refusals). With eviction grace off a client refused room
	// waits nowhere the self-test can see, so the count moving during a run
	// is what says a client wanted the memory the run is holding.
	Refusals uint64
}

// ModelActivity is one resident model's share of that view.
type ModelActivity struct {
	RepoID   string
	InFlight int
	LastUsed time.Time
}

// Job is a second kind of run the loop schedules at idle, beside the
// self-test's own: the context probe is one (internal/contextprobe). The loop
// owns idleness, yielding and the never-evict rule for every job, so there is
// one idea of "idle" on the Mac rather than one per feature.
type Job interface {
	// Name is the kind of run, for the log and the panel.
	Name() string
	// Due returns the model the job wants to run on next, given the ready
	// models, or "" when it wants nothing.
	Due(ready []string, now time.Time) string
	// Run performs one run on the model. It returns when done, or when the
	// session's context is cancelled: by a client arriving (Yielded reports
	// true) or by the switch going off.
	Run(s *Session, model string)
	// Parks reports whether the job's own loads may wait in the pool's
	// queue for room. A job that never parks — one that loads only when the
	// room is already free — lets the loop read the waiter count as a client
	// unconditionally, since none of the waiters can be its own.
	Parks() bool
}

// Session is what the loop hands a job for one run.
type Session struct {
	// Ctx is cancelled the moment a client appears or the loop stops.
	Ctx context.Context
	// Hold tells the loop how many in-flight requests on the model are the
	// job's own right now, so they are not mistaken for a client's. A job
	// that drives the model through the gateway holds one while a request is
	// in flight and none between requests.
	Hold func(n int)
	// Yielded reports whether Ctx was cancelled because a client appeared,
	// as opposed to the switch going off.
	Yielded func() bool
	// Report sets the words the panel shows for the run in progress.
	Report func(step string)
	// Fits reports whether the model can be loaded now without evicting
	// anything; a job that reloads between steps asks before each one.
	Fits func(model string) bool
}

// Status is what the loop is doing, for the panel.
type Status struct {
	// Job and Model name the run in progress; Step is the job's own words
	// for where it is. All empty when nothing is running.
	Job   string `json:"job,omitempty"`
	Model string `json:"model,omitempty"`
	Step  string `json:"step,omitempty"`
	// HeldBy names what kept a due run from starting at the last tick:
	// "in_flight", "waiting", "downloading", "recent" or "no_room"; empty when
	// nothing did, or nothing was due. Due names the model that run would be
	// on.
	HeldBy string `json:"held_by,omitempty"`
	Due    string `json:"due,omitempty"`
}

// What HeldBy can say.
const (
	HeldByInFlight    = "in_flight"
	HeldByWaiting     = "waiting"
	HeldByDownloading = "downloading"
	HeldByRecent      = "recent"
	// HeldByNoRoom is a model that is due and does not fit in the memory
	// budget beside what is already resident. The self-test never evicts, so
	// the run waits for room — and says it is waiting, rather than being
	// passed over in silence while the panel shows it queued
	// (iss-2609161712555136).
	HeldByNoRoom = "no_room"
)

// The cadence, and the bounds a run is held to.
const (
	// DefaultTick is how often the loop asks whether the Mac is idle.
	DefaultTick = time.Minute
	// DefaultQuiet is how long the last request must be in the past before
	// the Mac counts as idle.
	DefaultQuiet = 5 * time.Minute
	// DefaultRetest is how long a model's result stands before it is measured
	// again.
	DefaultRetest = 24 * time.Hour
	// DefaultPoll is how often a run in progress looks for a real request.
	DefaultPoll = 250 * time.Millisecond
	// DefaultRequestTimeout bounds one load and one request each, so a model
	// server that never answers cannot wedge the loop. It is the pool's own
	// readiness bound (runtime.ProbeTimeout).
	DefaultRequestTimeout = 10 * time.Minute
)

// Options configures a Runner. Only Server and Path are required; every
// duration left zero takes its default above.
type Options struct {
	Server Server
	// Path is the results file (config.Paths.SelfTest). Its directory is
	// created, closed, on the first write.
	Path string
	Log  *slog.Logger

	Tick, Quiet, Retest, Poll, RequestTimeout time.Duration
	// MaxBytes is the results file's cap; zero means DefaultMaxBytes.
	MaxBytes int64
	// Jobs are the other kinds of run the loop schedules, asked in order at
	// every idle tick before the self-test's own set; the first that is due
	// runs. A job that runs whether or not the self-test switch is on — the
	// probe's "Measure now" — is why the loop runs while the switch is off,
	// asking the jobs only.
	Jobs []Job
	// SelfTest is the self-test's own switch; the loop itself runs whenever
	// a job might want it. Nil means on.
	SelfTest func() bool
	// Client sends the test requests. Nil means a plain client; the timeout
	// is applied per request through its context.
	Client *http.Client
	// Now is the clock; nil means time.Now. It is what a result's "at" carries
	// and what the quiet and re-test windows are measured against, so a test
	// can place a result in the past without waiting.
	Now func() time.Time
}

// Runner is the loop. It does nothing until SetEnabled(true), and one Runner
// serves the life of the process: the switch turns the loop on and off.
type Runner struct {
	opts Options

	// switchMu serialises SetEnabled end to end, so an off that is draining a
	// run and an on that arrives meanwhile cannot leave two loops running.
	switchMu sync.Mutex

	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
	// tested is when each model (folded id) was last measured, seeded from the
	// results file when the loop starts so a restart does not measure
	// everything again.
	tested map[string]time.Time
	// touched is when the loop last released each model. The pool stamps a
	// model's last use at release, so without this the self-test's own run
	// would look like a client's for the whole quiet period afterwards.
	touched map[string]time.Time
	// noRoomSaid is which models (folded id) the log has already said do not
	// fit, so a hold that lasts hours costs one line rather than one a tick.
	noRoomSaid map[string]bool
	file       *file
	// status is what the panel reads; see Status.
	status Status
}

// New builds a Runner. It opens nothing and starts nothing.
func New(opts Options) *Runner {
	if opts.Log == nil {
		opts.Log = slog.Default()
	}
	if opts.Tick <= 0 {
		opts.Tick = DefaultTick
	}
	if opts.Quiet <= 0 {
		opts.Quiet = DefaultQuiet
	}
	if opts.Retest <= 0 {
		opts.Retest = DefaultRetest
	}
	if opts.Poll <= 0 {
		opts.Poll = DefaultPoll
	}
	if opts.RequestTimeout <= 0 {
		opts.RequestTimeout = DefaultRequestTimeout
	}
	if opts.MaxBytes <= 0 {
		opts.MaxBytes = DefaultMaxBytes
	}
	if opts.Client == nil {
		// No redirects: the one place a request may go is the model server
		// the pool named (adr-2609061503319212).
		opts.Client = &http.Client{
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		}
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	return &Runner{
		opts:    opts,
		tested:  map[string]time.Time{},
		touched: map[string]time.Time{},
		file:    &file{path: opts.Path, maxBytes: opts.MaxBytes, log: opts.Log},
	}
}

// SetEnabled turns the loop on or off. Off cancels a run in progress — the
// model is released and the run recorded as stopped — and returns once the
// loop has stopped. On while already on, and off while already off, do
// nothing.
func (r *Runner) SetEnabled(on bool) {
	r.switchMu.Lock()
	defer r.switchMu.Unlock()
	r.mu.Lock()
	if on == (r.cancel != nil) {
		r.mu.Unlock()
		return
	}
	if !on {
		cancel, done := r.cancel, r.done
		r.cancel, r.done = nil, nil
		r.mu.Unlock()
		cancel()
		<-done
		return
	}
	r.seedLocked()
	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel
	r.done = make(chan struct{})
	go r.loop(ctx, r.done)
	r.mu.Unlock()
}

// Enabled reports whether the loop is on.
func (r *Runner) Enabled() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.cancel != nil
}

// Close is SetEnabled(false) for the end of the process, and it releases the
// results file's handles. A Runner closed and switched on again opens them
// afresh on its next result.
func (r *Runner) Close() {
	r.SetEnabled(false)
	r.file.close()
}

// SetQuiet changes the idle threshold in force, so a saved setting applies
// to the next tick rather than the next start.
func (r *Runner) SetQuiet(d time.Duration) {
	if d <= 0 {
		return
	}
	r.mu.Lock()
	r.opts.Quiet = d
	r.mu.Unlock()
}

// Status reports what the loop is doing.
func (r *Runner) Status() Status {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.status
}

func (r *Runner) setStatus(f func(*Status)) {
	r.mu.Lock()
	f(&r.status)
	r.mu.Unlock()
}

// seedLocked reads when each model was last measured from the results file,
// so that a restart does not start the cycle over. Callers hold r.mu.
func (r *Runner) seedLocked() {
	runs, err := ReadResults(r.opts.Path)
	if err != nil {
		// The error names the path; the path is for the detailed level.
		r.opts.Log.Warn("self-test: could not read earlier results; every model counts as untested")
		r.opts.Log.Debug("self-test: could not read earlier results", "err", err)
		return
	}
	for _, run := range runs {
		// A stopped run is the switch's or the process's doing, not the
		// model's: it does not use up the model's day.
		if run.Outcome == OutcomeStopped {
			continue
		}
		at := time.Unix(run.At, 0)
		key := config.FoldRepoID(run.Model)
		if at.After(r.tested[key]) {
			r.tested[key] = at
		}
	}
}

func (r *Runner) loop(ctx context.Context, done chan struct{}) {
	defer close(done)
	tick := time.NewTicker(r.opts.Tick)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			// Both channels can be ready at once, and select picks at random:
			// a tick after the switch went off must not start a run with a
			// context that is already dead.
			if ctx.Err() != nil {
				return
			}
			r.tick(ctx)
		}
	}
}

// tick is one wake-up: is the Mac idle, is there a model due, and if both,
// one run.
func (r *Runner) tick(ctx context.Context) {
	now := r.opts.Now()
	ready := r.opts.Server.Ready()
	act := r.opts.Server.Activity()
	// What is due, before asking whether the Mac is idle, so that a due run
	// held back can say what held it.
	var job Job
	model := ""
	var noRoom []string
	// Every due job is asked, not only the ones up to the first that can run:
	// a model that does not fit is reported whether or not something else is
	// measured this tick. It used to be passed over in silence, and a Mac
	// with work of its own to do has a model due on most ticks, so the
	// queued one could stay invisible for as long as that lasted
	// (iss-2609161712555136, iss-2609181119346098).
	for _, j := range r.opts.Jobs {
		m := j.Due(ready, now)
		if m == "" {
			continue
		}
		if !r.opts.Server.Fits(m) {
			// The self-test never evicts, so a run with no room waits for
			// room rather than making it.
			noRoom = append(noRoom, m)
			continue
		}
		if job == nil {
			job, model = j, m
		}
	}
	if job == nil && (r.opts.SelfTest == nil || r.opts.SelfTest()) {
		model = r.next(now)
	}
	for _, m := range noRoom {
		r.sayNoRoom(m)
	}
	if model == "" {
		if len(noRoom) > 0 {
			r.setStatus(func(st *Status) { st.HeldBy, st.Due = HeldByNoRoom, noRoom[0] })
			return
		}
		r.setStatus(func(st *Status) { st.HeldBy, st.Due = "", "" })
		return
	}
	r.roomFound(model)
	if held := r.heldBy(act, now); held != "" {
		r.setStatus(func(st *Status) { st.HeldBy, st.Due = held, model })
		return
	}
	// A run that can go ahead does, and a model waiting for room is still
	// reported while it does: the hold belongs to the queued model, not to the
	// tick, and clearing it here is what used to make it disappear whenever
	// there was other work.
	if len(noRoom) > 0 {
		r.setStatus(func(st *Status) { st.HeldBy, st.Due = HeldByNoRoom, noRoom[0] })
	} else {
		r.setStatus(func(st *Status) { st.HeldBy, st.Due = "", "" })
	}
	if job != nil {
		r.runJob(ctx, job, model)
		return
	}
	r.run(ctx, model, resident(act, model))
}

// sayNoRoom writes the line about a model that is due and does not fit, once
// per model rather than once per tick: the condition persists — it is resolved
// by memory falling free, which can be hours away — and a line a tick would
// bury the log for as long as it lasts. The model's repository id is the whole
// of what is said; nothing about this Mac goes in it.
func (r *Runner) sayNoRoom(model string) {
	key := config.FoldRepoID(model)
	r.mu.Lock()
	said := r.noRoomSaid[key]
	if !said {
		if r.noRoomSaid == nil {
			r.noRoomSaid = map[string]bool{}
		}
		r.noRoomSaid[key] = true
	}
	r.mu.Unlock()
	if said {
		return
	}
	r.opts.Log.Info("the measurement is held because the model does not fit the memory budget beside what is loaded; it stays queued", "model", model)
}

// roomFound forgets that a model was ever short of room, so an occasion months
// later is reported as its own rather than swallowed by the first one.
func (r *Runner) roomFound(model string) {
	key := config.FoldRepoID(model)
	r.mu.Lock()
	delete(r.noRoomSaid, key)
	r.mu.Unlock()
}

// heldBy names what keeps a due run from starting, or "" when the Mac is
// idle. It is quiet's answer with its reason.
func (r *Runner) heldBy(act Activity, now time.Time) string {
	if act.Waiting > 0 {
		return HeldByWaiting
	}
	if act.Downloading > 0 {
		return HeldByDownloading
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, m := range act.Models {
		if m.InFlight > 0 {
			return HeldByInFlight
		}
		if m.LastUsed.IsZero() || now.Sub(m.LastUsed) >= r.opts.Quiet {
			continue
		}
		if own, ok := r.touched[config.FoldRepoID(m.RepoID)]; ok && !m.LastUsed.After(own) {
			continue
		}
		return HeldByRecent
	}
	return ""
}

// quiet reports whether the Mac is idle enough to start a run: nothing in
// flight, nothing waiting, nothing downloading, and the last request older
// than the quiet period. A model never used counts as long idle, and a last
// use that is the self-test's own release is not a request.
func (r *Runner) quiet(act Activity, now time.Time) bool { return r.heldBy(act, now) == "" }

// hold is what the loop itself has taken from the pool right now: the model,
// how many acquisitions it holds on it (one for the single tests, the decode
// concurrency's worth for the parallel one, so that the pool's own semaphore
// bounds the batch and a client arriving on the model waits in it, visibly),
// and how many refusals the pool had made when the run began.
type hold struct {
	model string
	// count is how many of the pool's in-flight entries on model are the
	// loop's own. It counts an acquisition still in progress as well as one
	// that has returned: the pool counts a load as in flight from the moment
	// Acquire is called, so a place is claimed here before it is asked for.
	count    int
	refusals uint64
	// parks says the holder's own load may be among the pool's waiters; see
	// Job.Parks.
	parks bool
	// loading says one of those claimed places is an Acquire still in
	// progress, which is what makes a waiter possibly the holder's own. It is
	// its own field rather than count == 0 because the place is claimed
	// before the load starts (iss-2609190018027384).
	loading bool
}

// busy reports whether anyone but the self-test wants the pool: a request in
// flight the loop is not holding itself, a load waiting for room, or a load
// refused room since the run began.
//
// While the loop's own Acquire is still parked, the waiter the pool counts
// may be the self-test's own, so waiters are not read then: a client that
// arrives during the load shows up as a request in flight on its model, or as
// a refusal, or is parked behind the self-test's load and cannot be told from
// the self-test's own place in that queue. The load's own place in flight is
// h.count's, claimed before Acquire is called, so it is never read as a
// client's either.
func busy(act Activity, h hold) bool {
	if act.Refusals > h.refusals {
		return true
	}
	// A waiter is a client unless it might be the holder's own parked load,
	// which is only possible for a job that parks and only while its load is
	// in progress.
	if act.Waiting > 0 && (!h.parks || !h.loading) {
		return true
	}
	held := config.FoldRepoID(h.model)
	for _, m := range act.Models {
		inFlight := m.InFlight
		if h.count > 0 && config.FoldRepoID(m.RepoID) == held {
			inFlight -= h.count
		}
		if inFlight > 0 {
			return true
		}
	}
	return false
}

func resident(act Activity, repoID string) bool {
	key := config.FoldRepoID(repoID)
	for _, m := range act.Models {
		if config.FoldRepoID(m.RepoID) == key {
			return true
		}
	}
	return false
}

// next is the ready model measured longest ago, or "" when every ready model
// has a result younger than the re-test window. A model with no result at
// all comes first.
func (r *Runner) next(now time.Time) string {
	ready := r.opts.Server.Ready()
	r.mu.Lock()
	defer r.mu.Unlock()
	var due []string
	for _, id := range ready {
		if at, ok := r.tested[config.FoldRepoID(id)]; ok && now.Sub(at) < r.opts.Retest {
			continue
		}
		// A model that would need something evicted is left for a tick when
		// there is room; it stays due, and is not charged a day.
		if !r.opts.Server.Fits(id) {
			continue
		}
		due = append(due, id)
	}
	sort.SliceStable(due, func(i, j int) bool {
		return r.tested[config.FoldRepoID(due[i])].Before(r.tested[config.FoldRepoID(due[j])])
	})
	if len(due) == 0 {
		return ""
	}
	return due[0]
}

// run measures one model and writes the result. wasResident says whether the
// model was in memory before the run, which decides whether the run unloads
// it afterwards: the self-test leaves residency as it found it.
func (r *Runner) run(ctx context.Context, model string, wasResident bool) {
	now := r.opts.Now()
	key := config.FoldRepoID(model)
	res := Run{Kind: KindRun, Model: model, At: now.Unix(), ColdLoad: !wasResident}
	defer func() {
		// A stopped run does not use up the model's day; see seedLocked.
		if res.Outcome != OutcomeStopped {
			r.mu.Lock()
			r.tested[key] = now
			r.mu.Unlock()
		}
		r.file.write(res)
		r.opts.Log.Info("self-test run", "model", model, "outcome", res.Outcome, "tests", len(res.Tests))
	}()

	// The watcher cancels the run's context the moment a real request
	// appears — during the load as much as during a test, since a cold load
	// is the longest thing a run does and the pool tears down an abandoned
	// load on cancel. yielded says that is what happened, as opposed to the
	// switch going off or a request failing on its own. holding is "" until
	// Acquire returns; see busy for why the watcher needs to know.
	// The self-test's own Acquire may park for room, so it is a parking job.
	w := r.startWatch(ctx, model, true)
	runCtx := w.ctx
	claim := w.claim
	r.setStatus(func(st *Status) { st.Job, st.Model, st.Step = "self-test", model, "" })
	defer r.setStatus(func(st *Status) { st.Job, st.Model, st.Step = "", "", "" })
	// ended stops the watcher and names the outcome of a run cut short.
	ended := func() {
		w.stop()
		switch {
		case w.didYield():
			res.Outcome, res.Reason = OutcomeYielded, ""
		case ctx.Err() != nil:
			res.Outcome, res.Reason = OutcomeStopped, ""
		}
	}

	loadCtx, cancelLoad := context.WithTimeout(runCtx, r.opts.RequestTimeout)
	defer cancelLoad()
	started := time.Now()
	// The place is claimed before it is asked for; see watch.claim.
	claim(1, true)
	up, release, err := r.opts.Server.Acquire(loadCtx, model)
	if err != nil {
		claim(0, false)
		res.Outcome, res.Reason = OutcomeFailed, ReasonLoad
		ended()
		return
	}
	res.LoadMs = time.Since(started).Milliseconds()
	claim(1, false)
	defer func() {
		release()
		r.mu.Lock()
		r.touched[key] = time.Now()
		r.mu.Unlock()
		// A model the self-test brought in goes back out — unless a client
		// has arrived, in which case the client's request decides what stays
		// resident, and an unload that then finds the model busy or already
		// evicted is the designed outcome rather than a warning.
		if wasResident || res.Outcome == OutcomeYielded {
			return
		}
		if err := r.opts.Server.Unload(model); err != nil {
			r.opts.Log.Debug("self-test: could not unload the model it loaded", "model", model, "err", err)
		}
	}()

	res.Outcome = OutcomeOK
	for _, spec := range standardSet(r.opts.Server.Concurrency()) {
		// The parallel test takes the rest of its requests' places in the
		// pool before it sends them, so the model server never decodes more
		// than the pool allows and a client arriving meanwhile queues in the
		// pool, where the watcher sees it, rather than beside the batch.
		// Claimed before asked for, as the load was: each of these
		// acquisitions is in flight in the pool before it returns, and a claim
		// raised only afterwards is the same false client as the load's.
		if spec.Parallel > 1 {
			claim(spec.Parallel, false)
		}
		extra, releaseExtra, err := r.holdMore(runCtx, model, spec.Parallel-1)
		if err != nil {
			releaseExtra()
			claim(1, false)
			res.Outcome, res.Reason = OutcomeFailed, ReasonLoad
			ended()
			return
		}
		claim(1+extra, false)
		t, err := r.runTest(runCtx, up, spec)
		releaseExtra()
		claim(1, false)
		if err != nil {
			res.Outcome, res.Reason = OutcomeFailed, ReasonRequest
			ended()
			return
		}
		res.Tests = append(res.Tests, t)
		r.setStatus(func(st *Status) { st.Step = spec.Name + " done" })
	}
	w.stop()
}

// holdMore acquires n further places on a resident model and returns how
// many it holds and a function that releases them all; n of zero or less
// holds nothing.
func (r *Runner) holdMore(ctx context.Context, model string, n int) (int, func(), error) {
	var releases []func()
	releaseAll := func() {
		for _, rel := range releases {
			rel()
		}
	}
	for i := 0; i < n; i++ {
		_, rel, err := r.opts.Server.Acquire(ctx, model)
		if err != nil {
			return len(releases), releaseAll, err
		}
		releases = append(releases, rel)
	}
	return len(releases), releaseAll, nil
}

// watch is the loop's eye on the pool for one run: it cancels the run's
// context the moment a client appears, and knows the difference between that
// and the switch going off.
type watch struct {
	ctx     context.Context
	cancel  context.CancelFunc
	done    chan struct{}
	mu      sync.Mutex
	held    hold
	yielded bool
}

// startWatch begins watching for a client on behalf of a run on model. The
// caller stops it with stop, which returns once the watcher is gone.
func (r *Runner) startWatch(ctx context.Context, model string, parks bool) *watch {
	runCtx, cancel := context.WithCancel(ctx)
	w := &watch{ctx: runCtx, cancel: cancel, done: make(chan struct{}),
		// A parking holder is taken to be loading until it says otherwise:
		// its load is the first thing it does, and the pool's waiter count may
		// be its own for as long as that lasts.
		held: hold{model: model, refusals: r.opts.Server.Activity().Refusals, parks: parks, loading: parks}}
	go func() {
		defer close(w.done)
		poll := time.NewTicker(r.opts.Poll)
		defer poll.Stop()
		for {
			select {
			case <-runCtx.Done():
				return
			case <-poll.C:
				w.mu.Lock()
				h := w.held
				w.mu.Unlock()
				if busy(r.opts.Server.Activity(), h) {
					w.mu.Lock()
					w.yielded = true
					w.mu.Unlock()
					cancel()
					return
				}
			}
		}
	}()
	return w
}

// setHold is what a job reports through Session.Hold: n requests of its own in
// flight. A job that holds nothing is between requests, which for a job that
// parks is where its own load can be.
func (w *watch) setHold(n int) { w.claim(n, n == 0) }

// claim records how many of the pool's in-flight entries on the model are the
// loop's own, and whether one of them is an Acquire still in progress. The
// loop claims a place before it asks for it and drops the claim only after
// the place is released: the pool counts a load as in flight from the moment
// Acquire is called, so a claim raised afterwards leaves the loop's own load
// looking like a client's for the whole of it (iss-2609190018027384).
func (w *watch) claim(n int, loading bool) {
	w.mu.Lock()
	w.held.count, w.held.loading = n, loading
	w.mu.Unlock()
}

func (w *watch) didYield() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.yielded
}

func (w *watch) stop() {
	w.cancel()
	<-w.done
}

// runJob gives a job one run on a model under the loop's watch.
func (r *Runner) runJob(ctx context.Context, job Job, model string) {
	w := r.startWatch(ctx, model, job.Parks())
	r.setStatus(func(st *Status) { st.Job, st.Model, st.Step = job.Name(), model, "" })
	defer func() {
		w.stop()
		r.mu.Lock()
		r.touched[config.FoldRepoID(model)] = time.Now()
		r.mu.Unlock()
		r.setStatus(func(st *Status) { st.Job, st.Model, st.Step = "", "", "" })
	}()
	job.Run(&Session{
		Ctx:     w.ctx,
		Hold:    w.setHold,
		Yielded: w.didYield,
		Report:  func(step string) { r.setStatus(func(st *Status) { st.Step = step }) },
		Fits:    r.opts.Server.Fits,
	}, model)
}

// runTest runs one test of the set: one request, or spec.Parallel of them
// at once, against the upstream.
func (r *Runner) runTest(ctx context.Context, up Upstream, spec testSpec) (Test, error) {
	if spec.Parallel <= 1 {
		m, err := r.measure(ctx, up, spec)
		if err != nil {
			return Test{}, err
		}
		return m.test(spec.Name, 1), nil
	}
	// One request failing cancels its siblings: the figure is the batch's,
	// and a batch with a hole in it is not worth finishing.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	results := make([]measurement, spec.Parallel)
	errs := make([]error, spec.Parallel)
	var wg sync.WaitGroup
	wall := time.Now()
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i], errs[i] = r.measure(ctx, up, spec)
			if errs[i] != nil {
				cancel()
			}
		}(i)
	}
	wg.Wait()
	if err := errors.Join(errs...); err != nil {
		return Test{}, err
	}
	return aggregate(spec.Name, results, time.Since(wall)), nil
}
