package registry

import (
	"fmt"
	"strings"
)

// LoadFailure records that a model's server started and never became ready
// — it exited during startup, its own log said the model cannot load, or
// the readiness timeout ran out — and the provenance it happened under. It
// lives on the model's registry entry beside Measured: a fact about these
// files on this Mac under this runtime, budget, concurrency and served
// window (iss-2609211334570516).
//
// While it stands, no idle job picks the model — the context probe and the
// self-test skip it — and a request for it is refused with the reason at
// once rather than paying another readiness timeout. It is lifted, not kept
// as stale, the moment any part of the provenance moves (RefreshStaleness):
// a failure under another provenance says nothing about this one. A person
// lifts it by hand with Load or Measure now, and a re-download's fresh
// record carries none.
type LoadFailure struct {
	// Reason is the pool's own text for the failure: the child's terminal
	// traceback line with anything path-shaped stripped, its exit status, or
	// the timeout. Shown on the card and told to an entitled client.
	Reason string `json:"reason"`
	// At is when the load failed, Unix seconds UTC.
	At int64 `json:"at"`
	// The provenance: what was in force when the load failed.
	Runtime           string `json:"runtime"`
	BudgetBytes       int64  `json:"budget_bytes"`
	DecodeConcurrency int    `json:"decode_concurrency"`
	ServedContext     int64  `json:"served_context"`
}

// MaxLoadFailureReasonBytes bounds the reason, which is published on the
// card and to entitled clients: past it a planted file's text is cleared,
// and a caller's is refused.
const MaxLoadFailureReasonBytes = 512

// LoadFailed reports whether a load failure stands against the model.
func (m Model) LoadFailed() bool { return m.LoadFailure != nil }

// StaleAgainst names the first part of the provenance that has moved, or ""
// while the failure still holds. Unlike a measurement's, a served window set
// to any other figure is a move: the charge and the window the model was
// refused under are gone.
func (f *LoadFailure) StaleAgainst(p Provenance) string {
	switch {
	case f.Runtime != p.Runtime:
		return StaleRuntime
	case f.BudgetBytes != p.BudgetBytes:
		return StaleBudget
	case f.DecodeConcurrency != p.DecodeConcurrency:
		return StaleConcurrency
	case f.ServedContext != p.ServedContext:
		return StaleServedContext
	}
	return ""
}

// plausibleLoadFailure bounds a failure read from the file, for the reason
// plausibleMeasurement bounds a measurement: registry.json is, in
// shared-cache mode, a file another local account can write, and the reason
// goes to the card and to entitled clients. A reason with a path separator
// in it is not one the pool wrote, which blanks those.
func plausibleLoadFailure(f *LoadFailure) bool {
	if f == nil {
		return false
	}
	const maxBytes = 1 << 50
	return f.Reason != "" && len(f.Reason) <= MaxLoadFailureReasonBytes && !strings.Contains(f.Reason, "/") &&
		len(f.Runtime) <= maxProvenanceBytes && f.At >= 0 && f.At < 1<<40 &&
		f.BudgetBytes >= 0 && f.BudgetBytes <= maxBytes &&
		f.DecodeConcurrency >= 0 && f.DecodeConcurrency <= 1024 &&
		f.ServedContext >= 0 && f.ServedContext <= MaxContextLength
}

// SetLoadFailure records a load failure on a model, replacing any earlier
// one. A nil failure lifts it.
func (r *Registry) SetLoadFailure(repoID string, f *LoadFailure) error {
	if f != nil && !plausibleLoadFailure(f) {
		return fmt.Errorf("registry: implausible load failure for %s", repoID)
	}
	r.mu.Lock()
	existing, ok := r.models[key(repoID)]
	if !ok {
		r.mu.Unlock()
		return fmt.Errorf("registry: %s: %w", repoID, ErrNotFound)
	}
	existing.LoadFailure = f
	r.models[key(repoID)] = existing
	snapshot := r.listLocked()
	err := r.saveLocked()
	r.mu.Unlock()
	r.broadcast(snapshot)
	return err
}
