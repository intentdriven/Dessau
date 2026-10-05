---
schema_version: 1
id: "iss-2610041948272032"
slug: "the-idle-threshold-cannot-be-set-above-one-hour-the-context"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "maintainer's request after the self-test loaded models straight after a peer's run, 2026-10-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/config/config.go"
remedy: "Raise MaxIdleThresholdSec to 86400 (24 hours), changing the config validation, the panel's Idle threshold field and docs/context-probe.md together, with the three-surfaces test covering the new upper bound."
---

The idle threshold cannot be set above one hour. The context probe and the self-test share idle_threshold_sec, which is held to 60–3600 seconds (MaxIdleThresholdSec in internal/config/config.go, and max="3600" on the panel's Idle threshold field). The maintainer wants these jobs to run only after a long quiet spell, up to 24 hours, so they never load models during an ordinary break in use. Their threshold is already at the 3600 cap.
