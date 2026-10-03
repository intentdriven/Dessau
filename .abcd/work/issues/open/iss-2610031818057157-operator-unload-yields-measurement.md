---
schema_version: 1
id: "iss-2610031818057157"
slug: "operator-unload-yields-measurement"
severity: "minor"
category: "architectural-insight"
source: "agent-finding"
found_during: "fixing the unload test's race (iss-2610031752289366)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/selftest/selftest.go"
remedy: "Decide with the maintainer: keep the yield (the measurement was asked for), or have the panel's Unload cancel a queued measurement of that model, and say which in docs/context-probe.md."
---

An operator's Unload of a model the context probe holds is a yield: the probe lets go, the model is unloaded, and the measurement the operator asked for resumes at the next idle tick, loading the model straight back. Whether the person who pressed Unload wanted the measurement abandoned, or only the memory back for now, is not decided anywhere; the test only now stops racing it.
