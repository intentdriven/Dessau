---
schema_version: 1
id: "iss-2610032241098944"
slug: "the-self-test-unloads-a-model-pinned-during-its-run"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "adversarial review of the probe-pins fix (iss-2609211754251373)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/app/selftest.go"
remedy: "Refuse the self-test's unload of a pinned model the way the probe's adapter now does (ErrPinned), leaving the model resident."
---

The self-test ignores pins as the context probe did: selfTestServer.Unload calls Pool.Unload directly, so a self-test that loaded a model unloads it at the end of its run even when the model was pinned during the run, leaving it pinned and not loaded.
