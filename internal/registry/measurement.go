package registry

import (
	"fmt"
)

// Measurement is what the context probe found for a model on this Mac: the
// largest prompt the server verifiably accepted, what stopped the next step,
// and the provenance the figure holds under (itd-2609091301112705). It lives
// on the model's registry entry, so it is a fact about these files on this
// machine, and it is never reused elsewhere.
type Measurement struct {
	// Window is the largest prompt, in the model server's own prompt_tokens,
	// that came back with an answer.
	Window int64 `json:"window"`
	// Bound is what stopped the step above Window: the model itself, or one
	// of the gateway's own limits, or the memory guard. Only BoundModel makes
	// Window the model's limit; every other bound makes it a floor.
	Bound string `json:"bound"`
	// At is when the measurement completed, Unix seconds UTC.
	At int64 `json:"at"`
	// The provenance: what was in force when the figure was taken. Move any
	// of them and the figure is stale.
	Runtime           string `json:"runtime"`
	BudgetBytes       int64  `json:"budget_bytes"`
	DecodeConcurrency int    `json:"decode_concurrency"`
	ServedContext     int64  `json:"served_context"`
	// GuardBytes is the projected peak footprint that the memory guard
	// refused, present only when Bound is BoundMemoryGuard: the step was
	// skipped, and this is the figure that skipped it.
	GuardBytes int64 `json:"guard_bytes,omitempty"`
	// Stale names the provenance that has moved since, or is empty while the
	// figure is current. Written by RefreshStaleness at start and at every
	// save, so a reader of the file learns it rather than inferring it.
	Stale string `json:"stale,omitempty"`
}

// The bounds a step can be stopped by.
const (
	BoundModel           = "model"
	BoundPrefillDeadline = "prefill_deadline"
	BoundServedWindow    = "served_window"
	BoundMemoryGuard     = "memory_guard"
)

// The reasons a measurement goes stale.
const (
	StaleRuntime       = "runtime"
	StaleBudget        = "budget"
	StaleConcurrency   = "decode_concurrency"
	StaleServedContext = "served_context"
)

// maxProvenanceBytes bounds the runtime string a planted file can carry.
const maxProvenanceBytes = 64

// ValidBound reports whether b is one of the four bounds.
func ValidBound(b string) bool {
	switch b {
	case BoundModel, BoundPrefillDeadline, BoundServedWindow, BoundMemoryGuard:
		return true
	}
	return false
}

func validStale(s string) bool {
	switch s {
	case "", StaleRuntime, StaleBudget, StaleConcurrency, StaleServedContext:
		return true
	}
	return false
}

// IsLimit reports whether the window is the model's own limit rather than a
// floor under it.
func (m *Measurement) IsLimit() bool { return m != nil && m.Bound == BoundModel }

// Provenance is what is in force now, to compare a measurement against.
type Provenance struct {
	Runtime           string
	BudgetBytes       int64
	DecodeConcurrency int
	ServedContext     int64
}

// StaleAgainst names the first part of the provenance that has moved, or ""
// when the measurement still holds. The order is the order a reader would
// ask in: the runtime first, since it changes the most about a reading.
func (m *Measurement) StaleAgainst(p Provenance) string {
	switch {
	case m.Runtime != p.Runtime:
		return StaleRuntime
	case m.BudgetBytes != p.BudgetBytes:
		return StaleBudget
	case m.DecodeConcurrency != p.DecodeConcurrency:
		return StaleConcurrency
	case m.ServedContext != p.ServedContext && p.ServedContext != m.Window:
		// A served window set to the measured window itself is adoption,
		// not a move: the floor was verified under a window at least that
		// large, and still stands.
		return StaleServedContext
	}
	return ""
}

// plausibleMeasurement bounds a measurement read from the file, for the
// reason plausibleContextLength bounds the window: registry.json is, in
// shared-cache mode, a file another local account can write, and a figure is
// cleared rather than repaired into something that looks right.
func plausibleMeasurement(m *Measurement) bool {
	if m == nil {
		return false
	}
	const maxBytes = 1 << 50 // a petabyte: past any Mac, still far from overflow
	return plausibleContextLength(m.Window) && ValidBound(m.Bound) && validStale(m.Stale) &&
		len(m.Runtime) <= maxProvenanceBytes && m.At >= 0 && m.At < 1<<40 &&
		m.BudgetBytes >= 0 && m.BudgetBytes <= maxBytes && m.GuardBytes >= 0 && m.GuardBytes <= maxBytes &&
		m.DecodeConcurrency >= 0 && m.DecodeConcurrency <= 1024 &&
		m.ServedContext >= 0 && m.ServedContext <= MaxContextLength
}

// SetMeasurement records the probe's result for a model, replacing any
// earlier one, and clears the incomplete mark. A nil measurement clears.
func (r *Registry) SetMeasurement(repoID string, m *Measurement) error {
	if m != nil && !plausibleMeasurement(m) {
		return fmt.Errorf("registry: implausible measurement for %s", repoID)
	}
	r.mu.Lock()
	existing, ok := r.models[key(repoID)]
	if !ok {
		r.mu.Unlock()
		return fmt.Errorf("registry: %s: %w", repoID, ErrNotFound)
	}
	existing.Measured = m
	existing.ProbeIncomplete = false
	r.models[key(repoID)] = existing
	snapshot := r.listLocked()
	err := r.saveLocked()
	r.mu.Unlock()
	r.broadcast(snapshot)
	return err
}

// SetProbeIncomplete marks a model whose probe was interrupted, so the next
// start can say so rather than silently retrying it.
func (r *Registry) SetProbeIncomplete(repoID string, on bool) error {
	r.mu.Lock()
	existing, ok := r.models[key(repoID)]
	if !ok {
		r.mu.Unlock()
		return fmt.Errorf("registry: %s: %w", repoID, ErrNotFound)
	}
	existing.ProbeIncomplete = on
	r.models[key(repoID)] = existing
	snapshot := r.listLocked()
	err := r.saveLocked()
	r.mu.Unlock()
	r.broadcast(snapshot)
	return err
}

// RefreshStaleness compares every measurement, every tool-call verdict and
// every load failure with what is in force now for its model — the served window is a per-model
// setting, so the provenance is asked per model — writes the verdict onto
// each, and returns the ids of the models whose verdict changed. It is called
// at start and after every save, so staleness is a stored fact rather than an
// assumption.
//
// The registry's lock is not held while inForce runs: the callback reads the
// pool and the configuration, and the pool already reads the registry under
// its own lock (Resolve, under p.mu), so holding r.mu across it would be the
// inversion adr-2609091239058072 exists to prevent. Instead the models are
// snapshotted, judged lock-free, and written back only where the measurement
// or the verdict is still the one that was judged.
func (r *Registry) RefreshStaleness(inForce func(m Model) Provenance) []string {
	type judged struct {
		key string
		// was and stale are the measurement judged and its verdict; tcWas
		// and tcStale the tool-call verdict's. A nil was or tcWas means that
		// side was not judged.
		was     *Measurement
		stale   string
		tcWas   *ToolCalling
		tcStale string
		// lfWas is a load failure whose provenance has moved; it is lifted
		// rather than marked, since a failure under another provenance says
		// nothing about this one.
		lfWas *LoadFailure
	}
	r.mu.RLock()
	var snapshot []Model
	for _, m := range r.models {
		if m.Measured != nil || m.ToolCalling != nil || m.LoadFailure != nil {
			snapshot = append(snapshot, m)
		}
	}
	r.mu.RUnlock()

	var verdicts []judged
	for _, m := range snapshot {
		p := inForce(m)
		j := judged{key: key(m.RepoID)}
		if m.Measured != nil {
			if stale := m.Measured.StaleAgainst(p); stale != m.Measured.Stale {
				j.was, j.stale = m.Measured, stale
			}
		}
		if m.ToolCalling != nil {
			if stale := m.ToolCalling.StaleAgainst(p.Runtime); stale != m.ToolCalling.Stale {
				j.tcWas, j.tcStale = m.ToolCalling, stale
			}
		}
		if m.LoadFailure != nil && m.LoadFailure.StaleAgainst(p) != "" {
			j.lfWas = m.LoadFailure
		}
		if j.was != nil || j.tcWas != nil || j.lfWas != nil {
			verdicts = append(verdicts, j)
		}
	}
	if len(verdicts) == 0 {
		return nil
	}

	r.mu.Lock()
	var changed []string
	for _, v := range verdicts {
		m, ok := r.models[v.key]
		if !ok {
			continue
		}
		moved := false
		if v.was != nil && m.Measured == v.was {
			copied := *m.Measured
			copied.Stale = v.stale
			m.Measured = &copied
			moved = true
		}
		if v.tcWas != nil && m.ToolCalling == v.tcWas {
			copied := *m.ToolCalling
			copied.Stale = v.tcStale
			m.ToolCalling = &copied
			moved = true
		}
		if v.lfWas != nil && m.LoadFailure == v.lfWas {
			m.LoadFailure = nil
			moved = true
		}
		if !moved {
			continue // replaced meanwhile; the next refresh judges the new one
		}
		r.models[v.key] = m
		changed = append(changed, m.RepoID)
	}
	var out []Model
	if len(changed) > 0 {
		out = r.listLocked()
		_ = r.saveLocked()
	}
	r.mu.Unlock()
	if len(changed) > 0 {
		r.broadcast(out)
	}
	return changed
}
