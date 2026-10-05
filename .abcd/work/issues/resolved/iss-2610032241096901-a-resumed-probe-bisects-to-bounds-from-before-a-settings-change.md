---
schema_version: 1
id: "iss-2610032241096901"
slug: "a-resumed-probe-bisects-to-bounds-from-before-a-settings-change"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "adversarial review of the probe-pins fix (iss-2609211754251373)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/contextprobe/probe.go"
remedy: "On resume, clamp b.hi to the candidate's current served window, or drop the bounds when the provenance differs from the one they were made under."
resolution: "A context probe's bounds now record the provenance they were made under; a resume under a different one starts the run over, a run whose provenance moves mid-run saves nothing and is retried, and a saved figure is stamped with the bounds' provenance."
impact: fix
resolved_by:
  commit: "2ae3853d5a6d5ac11da71be1ce451979863f7d77"
---

A resumed context probe bisects to bounds made under the settings in force when the run began: the bounds are kept in memory with no reset on a settings change, and b.hi was taken from the served window then. A run that yields (or, since iss-2609211754251373, meets a pin) and resumes after the served window was lowered can save a window above the new served window while stamping the new provenance.
