package app

import (
	"errors"
	"fmt"
	"net"
	"strconv"

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

func (s probeSources) Candidates() []contextprobe.Candidate {
	cfg := s.a.Config()
	models := s.a.Registry.Ready()
	out := make([]contextprobe.Candidate, 0, len(models))
	for _, m := range models {
		out = append(out, contextprobe.Candidate{
			RepoID:           m.RepoID,
			Declared:         m.ContextLength,
			Served:           cfg.ServedContext(m.RepoID, m.ContextLength),
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
	declared := int64(0)
	if m, err := s.a.Registry.Get(repoID); err == nil {
		declared = m.ContextLength
	}
	return s.provenanceFor(repoID, declared)
}

// provenanceFor is Provenance with the declared window in hand, for a caller
// that already holds the model — the registry's own staleness refresh, which
// runs under the registry's lock and must not read it back.
func (s probeSources) provenanceFor(repoID string, declared int64) registry.Provenance {
	return registry.Provenance{
		Runtime:           runtime.MLXLMVersion(),
		BudgetBytes:       s.a.Pool.MemoryBudget(),
		DecodeConcurrency: s.a.Pool.DecodeConcurrency(),
		ServedContext:     s.a.Config().ServedContext(repoID, declared),
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

// refreshStaleness compares every measurement with what is in force now and
// records the verdict, at start and after every save.
func (a *App) refreshStaleness() {
	src := probeSources{a}
	if changed := a.Registry.RefreshStaleness(func(m registry.Model) registry.Provenance {
		return src.provenanceFor(m.RepoID, m.ContextLength)
	}); len(changed) > 0 {
		a.Log.Info("context measurements re-judged against the settings in force", "models", changed)
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
