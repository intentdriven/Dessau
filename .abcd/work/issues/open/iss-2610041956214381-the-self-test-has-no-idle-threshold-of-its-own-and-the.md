---
schema_version: 1
id: "iss-2610041956214381"
slug: "the-self-test-has-no-idle-threshold-of-its-own-and-the"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "maintainer's request after the self-test loaded models straight after a peer's run, 2026-10-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/config/config.go"
remedy: "Give the self-test its own idle threshold (a self_test_idle_threshold_sec setting beside idle_threshold_sec, which stays the context probe's), defaulting to 14400 seconds (4 hours), across config, the panel's Self-test box and docs/self-test.md together; it needs the raised cap from iss-2610041948272032."
blocked_by: [iss-2610041948272032]
---

The self-test has no idle threshold of its own, and the shared default is too short for it. It waits for the same idle_threshold_sec as the context probe, whose default is 300 seconds (DefaultIdleThresholdSec in internal/config/config.go). So, out of the box, it loads and benchmarks a model five minutes after the last request: an ordinary pause in use. On 2026-10-04 it started loading models straight after a peer's run. The maintainer wants the two jobs separated, with the self-test waiting four hours by default, while the context probe keeps its own threshold.
