// Package contextprobe measures the largest prompt this Mac will actually
// serve a model: a sweep of growing prompts and a bisection between the last
// that came back and the first that did not, through Dessau's own OpenAI
// endpoint, the way a client reaches the model (itd-2609091301112705).
//
// It is a job of the self-test's idle loop (internal/selftest): that loop owns
// idleness, yielding and the never-evict rule, so there is one idea of "idle"
// on the Mac. This package owns the method, the result and the "Measure now"
// queue. The 2026-09-06 campaign's script is the reference implementation:
// unique prompts from a nonce so no prefix is cached, one output token at
// temperature 0, an unload before each step, the window read from the
// server's own usage.prompt_tokens, and a memory guard that skips a step
// projected too close to what the Mac has.
//
// Because the probe is a client of the gateway rather than an exception
// inside it, the largest step it can take is bounded by the gateway's own
// limits — the prefill deadline and the served-window check. A reading those
// stopped is published as a floor naming the bound, never as the model's
// limit.
package contextprobe

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/intentdriven/Dessau/internal/config"
	"github.com/intentdriven/Dessau/internal/registry"
	"github.com/intentdriven/Dessau/internal/runtime"
	"github.com/intentdriven/Dessau/internal/selftest"
)

// Name is the job's name, as the loop's status and the log carry it.
const Name = "context-probe"

// Candidate is a ready model as the probe sees it.
type Candidate struct {
	RepoID string
	// Declared is the window the model's own configuration declares; zero
	// means none, and there is nothing to bisect between.
	Declared int64
	// Served is the window Dessau serves the model at (the operator's
	// setting, or Declared), which bounds the largest step.
	Served int64
	// Bytes and KVChargePerToken are what the memory guard projects from.
	Bytes, KVChargePerToken int64
	// Measured is the model's current measurement, or nil.
	Measured *registry.Measurement
	// Incomplete says an earlier probe was interrupted; the probe does not
	// retry it on its own.
	Incomplete bool
}

// Sources is what the probe needs from the app.
type Sources interface {
	Candidates() []Candidate
	// Provenance is what is in force now for this model, to record with a
	// measurement.
	Provenance(repoID string) registry.Provenance
	// Available is how many bytes the memory budget has free right now.
	Available() int64
	// Unload stops a model server; runtime.ErrBusy when a request is in
	// flight on it, runtime.ErrNotLoaded when it is not resident.
	Unload(repoID string) error
	// Save records a completed measurement on the model.
	Save(repoID string, m *registry.Measurement) error
	// MarkIncomplete records that a probe of the model was interrupted.
	MarkIncomplete(repoID string, on bool) error
	// Endpoint is this Mac's own OpenAI endpoint, and the API key to send.
	Endpoint() (baseURL, apiKey string)
}

// Options configures a Probe.
type Options struct {
	Sources Sources
	// Enabled is the global switch; nil means off. "Measure now" runs a
	// probe whatever it says.
	Enabled func() bool
	Log     *slog.Logger
	Client  *http.Client
	Now     func() time.Time
	// Floor is the smallest prompt, in tokens, and the calibration size.
	Floor int64
	// MemoryMargin is how close to the budget's free bytes a step may come:
	// a step projected within it is skipped. The campaign used 24 GB on a
	// 128 GB Mac.
	MemoryMargin int64
	// StepTimeout bounds one request, given the request body's size. Nil
	// means the gateway's own prefill budget for that body plus a margin, so
	// the gateway's deadline is what stops a step; should the probe's own
	// timer fire first anyway, the step is filed as the deadline's, never as
	// the model's.
	StepTimeout func(bodyBytes int) time.Duration
	// UnloadWait bounds how long the probe waits for the gateway to release
	// its own abandoned request before treating a refused unload as a
	// client's, and for an unloaded server to hand its memory back before a
	// step. Zero means DefaultUnloadWait.
	UnloadWait time.Duration
}

// MaxProbeWindow caps the sweep whatever a model declares: a million
// tokens is past any window this runtime serves, and with the calibration
// clamped below it bounds the largest prompt the probe will ever build to
// well under the gateway's body cap. A planted declared window is the reason
// it is a constant and not the declared figure.
const MaxProbeWindow = 1 << 20

// The calibration's characters-per-token is clamped to this band: a server
// that counts oddly at the floor must not turn a target into a prompt of
// gigabytes.
const (
	minCharsPerToken = 1
	maxCharsPerToken = 16
)

// DefaultUnloadWait is the bound on the two waits above: the gateway
// releases a cancelled request as soon as its handler notices, and a stopped
// server hands its memory back within the pool's own stop bound.
const DefaultUnloadWait = 15 * time.Second

// Probe is the job.
type Probe struct {
	opts Options

	mu sync.Mutex
	// queue holds the "Measure now" requests, in order; queueGen moves with
	// every addition, so Due can tell a queue it snapshotted the candidates
	// against from one a hand retry reached meanwhile.
	queue    []string
	queueGen uint64
	// bounds holds a model's bisection so far, so a yielded run resumes.
	bounds map[string]*bounds
}

// bounds is where a model's bisection stands.
type bounds struct {
	// lo is the largest target size that came back, and loTokens what the
	// server counted it as; hi is the smallest that did not, and bound what
	// stopped it. cpt is the calibrated characters per token.
	lo, loTokens, hi int64
	bound            string
	guardBytes       int64
	cpt              float64
	// swept says the doubling sweep reached hi or a refusal, so a resumed
	// run bisects rather than starting the sweep over — or, when it did
	// not, resumes the sweep from lo.
	swept bool
}

// New builds a Probe.
func New(opts Options) *Probe {
	if opts.Log == nil {
		opts.Log = slog.Default()
	}
	if opts.Client == nil {
		opts.Client = &http.Client{
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		}
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.Floor <= 0 {
		opts.Floor = 1024
	}
	if opts.StepTimeout == nil {
		opts.StepTimeout = defaultStepTimeout
	}
	if opts.UnloadWait <= 0 {
		opts.UnloadWait = DefaultUnloadWait
	}
	return &Probe{opts: opts, bounds: map[string]*bounds{}}
}

// defaultStepTimeout is the gateway's prefill budget for a body of this
// size — its estimate of a token per four bytes, at 150 tokens a second,
// plus a minute, and at least ten minutes — with a minute's margin, so that
// the gateway's own deadline is what ends a step that outlasts it.
func defaultStepTimeout(bodyBytes int) time.Duration {
	tokens := int64(bodyBytes / 4)
	derived := time.Duration(tokens/150)*time.Second + time.Minute
	if derived < 10*time.Minute {
		derived = 10 * time.Minute
	}
	return derived + time.Minute
}

// Name implements selftest.Job.
func (p *Probe) Name() string { return Name }

// Parks implements selftest.Job: the probe never parks for room. It loads
// only when the room is already free, so every waiter the pool counts is a
// client's.
func (p *Probe) Parks() bool { return false }

// MeasureNow queues one probe of the model, whatever the switch says.
func (p *Probe) MeasureNow(repoID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	key := config.FoldRepoID(repoID)
	for _, q := range p.queue {
		if config.FoldRepoID(q) == key {
			return
		}
	}
	p.queue = append(p.queue, repoID)
	p.queueGen++
}

// Queued lists the models waiting for "Measure now".
func (p *Probe) Queued() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.queue...)
}

// Due implements selftest.Job: a queued model first, then — with the switch
// on — the first ready model with a declared window and no current
// measurement that is not marked incomplete (an interrupted or failed probe
// is retried only by "Measure now"). Both read only the app's candidates,
// which is where a model the server does not offer to chat, and one whose
// last load failed, are left out.
func (p *Probe) Due(ready []string, now time.Time) string {
	p.mu.Lock()
	gen := p.queueGen
	p.mu.Unlock()
	byKey := map[string]Candidate{}
	for _, c := range p.opts.Sources.Candidates() {
		byKey[config.FoldRepoID(c.RepoID)] = c
	}
	isReady := map[string]bool{}
	for _, id := range ready {
		isReady[config.FoldRepoID(id)] = true
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	// A queued model that is no longer a candidate — deleted, no longer
	// offered to chat, or its last load failed and the record on it stands
	// — is dropped rather than kept forever: the queue is what holds the
	// idle loop on, and a hand retry queues the model afresh. Only against
	// the queue the candidates were read for: a hand retry that lifted a
	// model's failure and queued it between the read and here is not
	// pruned on the stale snapshot, and the next tick judges it afresh.
	if gen == p.queueGen {
		kept := p.queue[:0]
		for _, q := range p.queue {
			if _, ok := byKey[config.FoldRepoID(q)]; ok {
				kept = append(kept, q)
			}
		}
		p.queue = kept
	}
	for _, q := range p.queue {
		if c, ok := byKey[config.FoldRepoID(q)]; ok && isReady[config.FoldRepoID(q)] && c.Declared > 0 {
			return c.RepoID
		}
	}
	if p.opts.Enabled == nil || !p.opts.Enabled() {
		return ""
	}
	for _, id := range ready {
		c, ok := byKey[config.FoldRepoID(id)]
		if !ok || c.Declared <= 0 || c.Incomplete {
			continue
		}
		if c.Measured != nil && c.Measured.Stale == "" {
			continue
		}
		return c.RepoID
	}
	return ""
}

// stepOutcome is what one request told the probe.
type stepOutcome int

const (
	stepOK stepOutcome = iota
	stepBounded
	stepYielded
	stepStopped
	stepHeld
	// stepAborted is an answer that is neither the model's nor one of the
	// gateway's size bounds — a key refused, a model not found — from which
	// no figure can be drawn and no retry helps.
	stepAborted
)

// Run implements selftest.Job: one probe of the model, resumed from its
// bounds when it has some.
func (p *Probe) Run(s *selftest.Session, model string) {
	var cand Candidate
	found := false
	for _, c := range p.opts.Sources.Candidates() {
		if config.FoldRepoID(c.RepoID) == config.FoldRepoID(model) {
			cand, found = c, true
			break
		}
	}
	if !found || cand.Declared <= 0 {
		p.dequeue(model)
		return
	}
	key := config.FoldRepoID(model)
	p.mu.Lock()
	b, ok := p.bounds[key]
	if !ok {
		// The gateway's served-window check is in force at the served
		// window whether or not it is below the declared one, so a sweep
		// that reaches the cap without a refusal was stopped by it; the
		// model's own limit is recorded only from a step it actually refused.
		b = &bounds{hi: min(cand.Declared, MaxProbeWindow), bound: registry.BoundServedWindow}
		if cand.Served > 0 && cand.Served < b.hi {
			b.hi = cand.Served
		}
		p.bounds[key] = b
	}
	p.mu.Unlock()

	finish := func(out stepOutcome) {
		switch out {
		case stepYielded, stepHeld:
			// Bounds kept; the loop will bring the model back when the Mac
			// is idle again.
			p.opts.Log.Info("context probe paused", "model", model, "why", map[stepOutcome]string{stepYielded: "a request arrived", stepHeld: "no room without evicting"}[out])
		case stepStopped:
			// No partial figure: the next start says the probe is incomplete.
			p.unloadWaiting(context.Background(), model)
			_ = p.opts.Sources.MarkIncomplete(model, true)
			p.opts.Log.Info("context probe interrupted", "model", model)
		case stepAborted:
			// An answer no figure can be drawn from; not retried on its own.
			p.mu.Lock()
			delete(p.bounds, key)
			p.mu.Unlock()
			p.dequeue(model)
			p.unloadWaiting(s.Ctx, model)
			_ = p.opts.Sources.MarkIncomplete(model, true)
			p.opts.Log.Warn("context probe abandoned: the gateway's answer was neither the model's nor a size bound", "model", model)
		}
	}

	// Calibration, once.
	if b.loTokens == 0 {
		s.Report(fmt.Sprintf("%s: calibrating at %d tokens", model, p.opts.Floor))
		out, res := p.step(s, cand, b, p.opts.Floor)
		if out != stepOK {
			if out == stepBounded {
				// Nothing to bisect between: the model did not answer at
				// the floor. Marked incomplete, so "Measure now" is the
				// only retry.
				p.mu.Lock()
				delete(p.bounds, key)
				p.mu.Unlock()
				p.dequeue(model)
				p.unloadWaiting(s.Ctx, model)
				_ = p.opts.Sources.MarkIncomplete(model, true)
				p.opts.Log.Warn("context probe: the model did not answer at the floor size", "model", model, "bound", res.bound)
				return
			}
			finish(out)
			return
		}
		b.lo, b.loTokens = p.opts.Floor, res.tokens
		b.cpt = min(max(float64(len(res.prompt))/float64(res.tokens), minCharsPerToken), maxCharsPerToken)
	}
	// The doubling sweep, resumed from the largest size verified so far
	// after a yield, up to a refusal or the cap.
	for !b.swept {
		size := b.lo * 2
		if size >= b.hi {
			b.swept = true
			break
		}
		s.Report(fmt.Sprintf("%s: sweeping at %d tokens (verified %d so far)", model, size, b.loTokens))
		out, res := p.step(s, cand, b, size)
		switch out {
		case stepOK:
			b.lo, b.loTokens = size, res.tokens
		case stepBounded:
			b.hi, b.bound, b.guardBytes, b.swept = size, res.bound, res.guardBytes, true
		default:
			finish(out)
			return
		}
	}

	// Bisect to the campaign's tolerance.
	for b.hi-b.lo > max(1024, b.lo/8) {
		mid := (b.lo + b.hi) / 2
		s.Report(fmt.Sprintf("%s: bisecting at %d tokens (between %d and %d)", model, mid, b.lo, b.hi))
		out, res := p.step(s, cand, b, mid)
		switch out {
		case stepOK:
			b.lo, b.loTokens = mid, res.tokens
		case stepBounded:
			b.hi, b.bound, b.guardBytes = mid, res.bound, res.guardBytes
		default:
			finish(out)
			return
		}
	}

	prov := p.opts.Sources.Provenance(model)
	m := &registry.Measurement{
		Window: b.loTokens, Bound: b.bound, At: p.opts.Now().Unix(),
		Runtime: prov.Runtime, BudgetBytes: prov.BudgetBytes, DecodeConcurrency: prov.DecodeConcurrency,
		ServedContext: prov.ServedContext, GuardBytes: b.guardBytes,
	}
	p.unloadWaiting(s.Ctx, model)
	if err := p.opts.Sources.Save(model, m); err != nil {
		// Forty minutes of GPU are in the bounds; keep them for a retry and
		// say what happened, rather than throwing the run away.
		_ = p.opts.Sources.MarkIncomplete(model, true)
		p.opts.Log.Warn("context probe: could not record the measurement; the run's bounds are kept for Measure now", "model", model, "err", err)
		p.dequeue(model)
		return
	}
	p.opts.Log.Info("context probe measured", "model", model, "window", m.Window, "bound", m.Bound)
	p.mu.Lock()
	delete(p.bounds, key)
	p.mu.Unlock()
	p.dequeue(model)
}

func (p *Probe) dequeue(model string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	key := config.FoldRepoID(model)
	for i, q := range p.queue {
		if config.FoldRepoID(q) == key {
			p.queue = append(p.queue[:i], p.queue[i+1:]...)
			return
		}
	}
}

// unloadWaiting stops the model's server, waiting out the gateway's release
// of the probe's own abandoned request first: the gateway gives up a
// cancelled request when its handler notices the client is gone, which is
// after the client's call has returned, so an Unload straight after the
// cancel finds the probe's own request still in flight. It retries for
// UnloadWait; an ErrBusy that outlasts that is a client's request, and the
// model is left to it. It reports whether the model was stopped or was not
// loaded, which is what a step's reading depends on.
func (p *Probe) unloadWaiting(ctx context.Context, model string) bool {
	deadline := time.Now().Add(p.opts.UnloadWait)
	for {
		err := p.opts.Sources.Unload(model)
		switch {
		case err == nil, errors.Is(err, runtime.ErrNotLoaded):
			return true
		case !errors.Is(err, runtime.ErrBusy):
			p.opts.Log.Debug("context probe: could not unload", "model", model, "err", err)
			return false
		}
		if time.Now().After(deadline) {
			p.opts.Log.Debug("context probe: the model stayed busy; a client has it", "model", model)
			return false
		}
		select {
		case <-time.After(50 * time.Millisecond):
		case <-ctx.Done():
			// The stop path passes a background context; a yield's session
			// context is already done, so keep waiting on the clock alone.
			if ctx.Err() != nil && ctx != context.Background() {
				continue
			}
		}
	}
}

// stepResult is what a step produced.
type stepResult struct {
	tokens     int64
	prompt     string
	bound      string
	guardBytes int64
}

// step runs one request at a target size: an unload first so no retained
// cache flatters the reading, the memory guard, the never-evict check, and
// then the request through the gateway with the probe's hold declared.
func (p *Probe) step(s *selftest.Session, c Candidate, b *bounds, size int64) (stepOutcome, stepResult) {
	if s.Ctx.Err() != nil {
		return p.cancelled(s), stepResult{}
	}
	// Unload before every step. A model that stays busy is a client's: that
	// is a yield, and there is no reading to discard yet.
	if !p.unloadWaiting(s.Ctx, c.RepoID) {
		return stepYielded, stepResult{}
	}
	// The stopped server hands its memory back over the pool's stop bound,
	// and until it has, the budget still counts it as exiting: asking Fits at
	// once would refuse any model larger than half the budget. Wait it out.
	if !p.fitsWaiting(s, c.RepoID) {
		return stepHeld, stepResult{}
	}
	// The memory guard: the model's flat charge plus the cache one sequence
	// of this size costs, against what the budget has free, less the margin.
	if c.KVChargePerToken > 0 && c.Bytes > 0 && c.Bytes < 1<<50 && c.KVChargePerToken < 1<<30 {
		projected := c.Bytes + c.Bytes/5 + c.KVChargePerToken*size
		if projected > p.opts.Sources.Available()-p.opts.MemoryMargin {
			return stepBounded, stepResult{bound: registry.BoundMemoryGuard, guardBytes: projected}
		}
	}
	cpt := b.cpt
	if cpt <= 0 {
		cpt = 4
	}
	prompt := filler(int(float64(size) * cpt))
	tokens, verdict, err := p.request(s, c.RepoID, prompt)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) && s.Ctx.Err() == nil {
			// The probe's own timer, which is the gateway's budget plus a
			// margin: a step that outlasted it is the deadline's, and the
			// server is still prefilling the abandoned prompt, so stop it.
			p.unloadWaiting(s.Ctx, c.RepoID)
			return stepBounded, stepResult{bound: registry.BoundPrefillDeadline}
		}
		if s.Ctx.Err() != nil {
			out := p.cancelled(s)
			if out == stepYielded {
				// The gateway gave up on the request; the server has not.
				// Stop it, so the abandoned prefill neither keeps the GPU nor
				// leaves its cache resident. Refused because a client already
				// took the model: nothing of this step is kept, since a server
				// that kept the cache would measure the cache.
				p.unloadWaiting(context.Background(), c.RepoID)
			}
			return out, stepResult{}
		}
		// A transport failure — the server crashed under the prompt, or the
		// gateway answered with an upstream failure — is the model's.
		p.opts.Log.Debug("context probe: a step failed", "model", c.RepoID, "size", size, "err", err)
		return stepBounded, stepResult{bound: registry.BoundModel}
	}
	switch verdict.kind {
	case answerOK:
		return stepOK, stepResult{tokens: tokens, prompt: prompt}
	case answerBounded:
		return stepBounded, stepResult{bound: verdict.bound}
	case answerNoRoom:
		// The pool had no room, or was closing: not the model, not a size
		// bound, and worth a later tick.
		return stepHeld, stepResult{}
	default:
		return stepAborted, stepResult{}
	}
}

// fitsWaiting asks whether the model fits beside what is resident, waiting
// out a server that is still handing its memory back.
func (p *Probe) fitsWaiting(s *selftest.Session, model string) bool {
	deadline := time.Now().Add(p.opts.UnloadWait)
	for {
		if s.Fits(model) {
			return true
		}
		if time.Now().After(deadline) || s.Ctx.Err() != nil {
			return false
		}
		select {
		case <-time.After(100 * time.Millisecond):
		case <-s.Ctx.Done():
			return false
		}
	}
}

// answer classifies the gateway's reply to a step.
type answer struct {
	kind  answerKind
	bound string
}

type answerKind int

const (
	answerOK answerKind = iota
	// answerBounded is one of the gateway's own size bounds: its prefill
	// deadline (504) or its served-window check (400).
	answerBounded
	// answerNoRoom is the pool refusing for want of room or because it is
	// closing (503): no reading, try later.
	answerNoRoom
	// answerAborted is an answer no figure can be drawn from — a key
	// refused, the model unknown to the gateway, the body cap (413, which
	// is not one of the four bounds a measurement may carry).
	answerAborted
)

func (p *Probe) cancelled(s *selftest.Session) stepOutcome {
	if s.Yielded() {
		return stepYielded
	}
	return stepStopped
}

// request sends one non-streaming request through the gateway and reads the
// server's count of the prompt, classifying the gateway's own answers: 504 is
// its prefill deadline and 400 its served-window check (both size bounds),
// 503 is the pool without room (try later), 502 is an upstream failure (the
// model's), and anything else is an answer no figure can be drawn from.
func (p *Probe) request(s *selftest.Session, model, prompt string) (tokens int64, verdict answer, err error) {
	baseURL, apiKey := p.opts.Sources.Endpoint()
	body, err := json.Marshal(map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": "You are a test harness. Reply with exactly: OK"},
			{"role": "user", "content": prompt + "\n\nReply with exactly: OK"},
		},
		"max_tokens": 1, "temperature": 0, "stream": false,
	})
	if err != nil {
		return 0, answer{}, err
	}
	ctx, cancel := context.WithTimeout(s.Ctx, p.opts.StepTimeout(len(body)))
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return 0, answer{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	s.Hold(1)
	defer s.Hold(0)
	resp, err := p.opts.Client.Do(req)
	if err != nil {
		return 0, answer{}, err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusGatewayTimeout:
		return 0, answer{kind: answerBounded, bound: registry.BoundPrefillDeadline}, nil
	case http.StatusBadRequest:
		return 0, answer{kind: answerBounded, bound: registry.BoundServedWindow}, nil
	case http.StatusServiceUnavailable:
		return 0, answer{kind: answerNoRoom}, nil
	case http.StatusBadGateway, http.StatusInternalServerError:
		return 0, answer{}, fmt.Errorf("upstream failure, status %d", resp.StatusCode)
	default:
		p.opts.Log.Warn("context probe: the gateway's answer is not one a step can be read from", "model", model, "status", resp.StatusCode)
		return 0, answer{kind: answerAborted}, nil
	}
	var out struct {
		Usage *struct {
			PromptTokens int64 `json:"prompt_tokens"`
		} `json:"usage"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&out); err != nil {
		return 0, answer{}, err
	}
	if out.Usage == nil || out.Usage.PromptTokens <= 0 {
		return 0, answer{}, errors.New("the answer carried no prompt count")
	}
	return out.Usage.PromptTokens, answer{kind: answerOK}, nil
}
