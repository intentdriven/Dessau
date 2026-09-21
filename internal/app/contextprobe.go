package app

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/intentdriven/Dessau/internal/config"
	"github.com/intentdriven/Dessau/internal/contextprobe"
	"github.com/intentdriven/Dessau/internal/registry"
	"github.com/intentdriven/Dessau/internal/runtime"
)

// ErrNoMeasurement is returned when a model has nothing to adopt.
var ErrNoMeasurement = errors.New("the model has no current measurement")

// probeSources is what the context probe sees of the app. Like
// selfTestServer it holds none of the app's locks across a call
// (adr-2609091239058072): each method takes the registry's, the pool's or the
// configuration's own lock for one read.
type probeSources struct{ a *App }

// Candidates is every ready model the probe may measure. The probe measures
// through chat completions, so a model the server does not offer to chat —
// the verdict the models list publishes as `chat`, read from its one home,
// registry.Model.CanChat — is not one: a served window means nothing for it,
// and its server never answers the request that would measure it
// (iss-2609211334563318).
func (s probeSources) Candidates() []contextprobe.Candidate {
	models := s.a.Registry.Ready()
	rule := s.a.Config().EffectiveChatRule()
	out := make([]contextprobe.Candidate, 0, len(models))
	for _, m := range models {
		// Nor is a model whose last load failed under the provenance in
		// force: the record on it stands until that moves or a person
		// retries by hand, and idle work is neither (iss-2609211334570516).
		if !m.CanChat(rule) || m.LoadFailed() {
			continue
		}
		served, _ := s.a.ServedWindow(m)
		out = append(out, contextprobe.Candidate{
			RepoID:           m.RepoID,
			Declared:         m.ContextLength,
			Served:           served,
			Bytes:            chargedSize(m),
			KVChargePerToken: m.KVChargePerToken,
			Measured:         m.Measured,
			Incomplete:       m.ProbeIncomplete,
		})
	}
	return out
}

// Provenance is what is in force for a model now: the runtime this build
// runs, the budget and concurrency the pool enforces, and the model's served
// window. It is the one spelling of the provenance, used both to stamp a
// measurement and to judge it stale.
func (s probeSources) Provenance(repoID string) registry.Provenance {
	m, err := s.a.Registry.Get(repoID)
	if err != nil {
		m = registry.Model{RepoID: repoID}
	}
	return s.provenanceFor(m)
}

// provenanceFor is Provenance with the model in hand, for a caller that
// already holds it — the registry's own staleness refresh, which runs under
// the registry's lock and must not read it back.
func (s probeSources) provenanceFor(m registry.Model) registry.Provenance {
	served, _ := s.a.ServedWindow(m)
	return registry.Provenance{
		Runtime:           runtime.MLXLMVersion(),
		BudgetBytes:       s.a.Pool.MemoryBudget(),
		DecodeConcurrency: s.a.Pool.DecodeConcurrency(),
		ServedContext:     served,
	}
}

// Available is what the budget has free: the budget less every resident
// charge and the memory still on its way out.
func (s probeSources) Available() int64 {
	res := s.a.Pool.Residency()
	used := res.ExitingBytes
	for _, m := range res.Models {
		used += m.Charge
	}
	return s.a.Pool.MemoryBudget() - used
}

func (s probeSources) Unload(repoID string) error { return s.a.Pool.Unload(repoID) }

func (s probeSources) Save(repoID string, m *registry.Measurement) error {
	return s.a.Registry.SetMeasurement(repoID, m)
}

func (s probeSources) MarkIncomplete(repoID string, on bool) error {
	return s.a.Registry.SetProbeIncomplete(repoID, on)
}

// Endpoint is this Mac's own OpenAI endpoint on loopback, and the API key
// in force: sent although loopback is admitted without it, so a later
// tightening of that rule does not silently break the probe.
func (s probeSources) Endpoint() (string, string) {
	host := net.JoinHostPort(s.a.bindPlan.Loopback, strconv.Itoa(s.a.BindPort()))
	return "http://" + host, s.a.Config().APIKey
}

// applyIdleJobs puts the idle loop's switches into force: the loop runs when
// either idle job is switched on or a probe is queued, the self-test's own
// set follows its switch alone, and the idle threshold applies to both.
func (a *App) applyIdleJobs(c config.Config) {
	if a.idleQuiet == 0 {
		a.SelfTest.SetQuiet(c.EffectiveIdleThreshold())
	}
	a.SelfTest.SetEnabled(c.SelfTest || c.ContextProbe || len(a.Probe.Queued()) > 0)
}

// refreshStaleness compares every measurement, and every tool-call verdict,
// with what is in force now and records the verdict, at start and after
// every save.
func (a *App) refreshStaleness() {
	src := probeSources{a}
	if changed := a.Registry.RefreshStaleness(src.provenanceFor); len(changed) > 0 {
		a.Log.Info("measurements re-judged against the settings in force", "models", changed)
	}
}

// MeasureNow queues one probe of the model, whatever the switch says, and
// makes sure the idle loop is running to pick it up.
func (a *App) MeasureNow(repoID string) error {
	m, err := a.Registry.Get(repoID)
	if err != nil {
		return err
	}
	if !m.Ready() {
		return fmt.Errorf("%s is not ready", repoID)
	}
	if m.ContextLength <= 0 {
		return fmt.Errorf("%s declares no context window; there is nothing to measure between", repoID)
	}
	if !m.CanChat(a.Config().EffectiveChatRule()) {
		return fmt.Errorf("%s is not offered to chat, and the probe measures through chat completions", repoID)
	}
	// A hand retry: a load failure on the model is lifted, so the probe
	// may load it again.
	a.ForgetLoadFailure(m.RepoID)
	// Under the save lock, so a save that reads the queue empty cannot
	// switch the loop off between the queueing and the start.
	a.saveMu.Lock()
	defer a.saveMu.Unlock()
	a.Probe.MeasureNow(m.RepoID)
	a.SelfTest.SetEnabled(true)
	return nil
}

// AdoptMeasurement makes a model's current measurement its served window:
// the setting that already exists, already charged by the budget and already
// enforced at the gateway. It is a settings save like any other, so every
// check a save makes is made here, and nothing else changes.
func (a *App) AdoptMeasurement(repoID string) error {
	m, err := a.Registry.Get(repoID)
	if err != nil {
		return err
	}
	if m.Measured == nil || m.Measured.Stale != "" {
		return ErrNoMeasurement
	}
	c := a.Config()
	models := make(map[string]config.ModelSettings, len(c.Models)+1)
	for k, v := range c.Models {
		models[k] = v
	}
	key := a.canonicalModelKey(m.RepoID)
	ms := models[key]
	ms.ServedContext = m.Measured.Window
	models[key] = ms
	c.Models = models
	return a.SetConfig(c)
}

// recordLoadFailure is the pool observer's other half for a load that never
// became ready: the pool's reason is written onto the model with the
// provenance in force, where it stands until that moves or a person retries
// the model by hand (registry.LoadFailure). A load another path interrupted
// is not the model's failure and leaves no mark; a load that succeeded lifts
// one. Called off the pool's lock, on the observer's own goroutine.
func (a *App) recordLoadFailure(repoID string, err error) {
	if err == nil {
		a.ForgetLoadFailure(repoID)
		return
	}
	var notReady *runtime.NotReadyError
	if !errors.As(err, &notReady) || notReady.Interrupted {
		return
	}
	// Reports arrive on their own goroutines, in no fixed order. A failure
	// reported after a hand retry has lifted the mark and started a fresh
	// load must not mark the model over that load: the pool holding an
	// entry for the model now is the retry, and its own report decides.
	for _, res := range a.Pool.Residency().Models {
		if config.FoldRepoID(res.RepoID) == config.FoldRepoID(repoID) {
			return
		}
	}
	reason := notReady.Reason
	if reason == "" {
		reason = "did not become ready"
	}
	if len(reason) > registry.MaxLoadFailureReasonBytes {
		reason = reason[:registry.MaxLoadFailureReasonBytes]
	}
	prov := probeSources{a}.Provenance(repoID)
	if err := a.Registry.SetLoadFailure(repoID, &registry.LoadFailure{
		Reason: reason, At: time.Now().Unix(), Transient: notReady.Transient,
		Runtime: prov.Runtime, BudgetBytes: prov.BudgetBytes,
		DecodeConcurrency: prov.DecodeConcurrency, ServedContext: prov.ServedContext,
	}); err != nil {
		a.Log.Warn("could not record the load failure on the model", "model", repoID, "err", err)
	}
}

// ForgetLoadFailure lifts a load failure from a model: the hand retry, which
// Load and Measure now on the card are. A model with none, or one the
// registry does not hold, is left alone.
func (a *App) ForgetLoadFailure(repoID string) {
	m, err := a.Registry.Get(repoID)
	if err != nil || !m.LoadFailed() {
		return
	}
	if err := a.Registry.SetLoadFailure(m.RepoID, nil); err != nil {
		a.Log.Warn("could not lift the load failure from the model", "model", repoID, "err", err)
	}
}
