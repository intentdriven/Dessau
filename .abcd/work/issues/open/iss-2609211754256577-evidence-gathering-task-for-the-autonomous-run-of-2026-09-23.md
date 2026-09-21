---
schema_version: 1
id: "iss-2609211754256577"
slug: "evidence-gathering-task-for-the-autonomous-run-of-2026-09-23"
severity: "minor"
category: "process"
source: "plan-review"
found_during: "planning interview 2026-09-21"
origin: researcher-authored
production_mode: hand-written
---

Evidence-gathering task for the autonomous run of 2026-09-23, per the 2026-09-21 decision on itd-2609091712142715: with the live box on v0.9.3 and statistics on, run ONE self-test cycle and read back the served-window statistics onto the record (prompt-size distribution against the served window per model, served-window refusal rate, observed in-flight counts, sampling override rate per parameter) as a dated note under .abcd/development/research/ — the reading the hold on itd-2609091712142715 names. Not a code change; the run's script carries it as a manual/handcheck-class step because it touches the live box, which no session contacts on its own.
