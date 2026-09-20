package registry

import "fmt"

// ToolCalling is what the tool-call probe found for a model on this Mac:
// whether it answered one fixed question, with one small tool declared, by
// calling the tool (itd-2609201445423499). It lives beside Measured on the
// model's registry entry: a fact about these files under this runtime, and
// nothing else in the provenance matters, since the memory budget and the
// served window cannot change what a chat template does.
type ToolCalling struct {
	// Can says the model answered with a tool call. False is a real
	// "cannot" on this runtime: text, or an empty message, which is what
	// the pinned runtime returns for some families.
	Can bool `json:"can"`
	// At is when the verdict was recorded, Unix seconds UTC.
	At int64 `json:"at"`
	// Runtime is the mlx-lm version the verdict was taken under. A verdict
	// under any other runtime is stale.
	Runtime string `json:"runtime"`
	// Stale names the provenance that has moved since — only StaleRuntime
	// can — or is empty while the verdict is current. Written by
	// RefreshStaleness beside the measurement's own mark, so a reader of the
	// file learns it rather than inferring it.
	Stale string `json:"stale,omitempty"`
}

// Current reports whether the verdict holds under the runtime in force: it
// is what the models list publishes and the probe queues on.
func (tc *ToolCalling) Current() bool { return tc != nil && tc.Stale == "" }

// StaleAgainst names what has moved, or "" when the verdict still holds.
func (tc *ToolCalling) StaleAgainst(runtime string) string {
	if tc.Runtime != runtime {
		return StaleRuntime
	}
	return ""
}

// plausibleToolCalling bounds a verdict read from the file, for the reason
// plausibleMeasurement bounds a measurement: registry.json is, in
// shared-cache mode, a file another local account can write, and a verdict is
// cleared rather than repaired into something that looks right.
func plausibleToolCalling(tc *ToolCalling) bool {
	if tc == nil {
		return false
	}
	return len(tc.Runtime) <= maxProvenanceBytes && tc.At >= 0 && tc.At < 1<<40 &&
		(tc.Stale == "" || tc.Stale == StaleRuntime)
}

// SetToolCalling records the probe's verdict for a model, replacing any
// earlier one. A nil verdict clears.
func (r *Registry) SetToolCalling(repoID string, tc *ToolCalling) error {
	if tc != nil && !plausibleToolCalling(tc) {
		return fmt.Errorf("registry: implausible tool-call verdict for %s", repoID)
	}
	r.mu.Lock()
	existing, ok := r.models[key(repoID)]
	if !ok {
		r.mu.Unlock()
		return fmt.Errorf("registry: %s: %w", repoID, ErrNotFound)
	}
	existing.ToolCalling = tc
	r.models[key(repoID)] = existing
	snapshot := r.listLocked()
	err := r.saveLocked()
	r.mu.Unlock()
	r.broadcast(snapshot)
	return err
}
