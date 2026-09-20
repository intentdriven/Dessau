---
schema_version: 1
id: "iss-2609202010356702"
slug: "an-operator-unload-that-lands-inside-the-tool-call-probe-s-h"
severity: "minor"
category: "ux"
source: "agent-observation"
found_during: "implementation of the tool-call probe (itd-2609201445423499), pilot run 2026-09-20"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/runtime/pool.go"
---

An operator Unload that lands inside the tool-call probe's hold is refused with 409. Pool.Unload refuses any in-flight hold, and the probe holds the model for the few seconds its one request takes, so an Unload from the panel in that window fails the same way it already does during a self-test run. The exposure is the self-test's, now reachable once more per model per runtime; a capture rather than a change, because the hold is what keeps the model from being evicted under the probe.
