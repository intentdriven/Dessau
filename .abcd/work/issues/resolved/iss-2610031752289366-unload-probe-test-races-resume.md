---
schema_version: 1
id: "iss-2610031752289366"
slug: "unload-probe-test-races-resume"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "driving PR 168 to green"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gateway/probe_integration_test.go"
remedy: "Decide whether an operator's Unload of a model a probe holds should cancel the queued measurement rather than yield it; either way, make the test assert on what Unload promises (the job let go and the model was unloaded) without racing the loop's next tick, for example by waiting for the probe's run to end before reading residency or by giving the stack a quiet window longer than the read."
resolution: "The test now holds Unload to what it promises: the instance the probe held is gone, and a model found afterwards must be a fresh load from the probe's resume. The product question it raised is captured as iss-2610031818057157."
impact: internal
resolved_by:
  commit: "9319e11"
---

TestUnloadFromThePanelTakesTheModelBackFromTheProbe fails about one run in a few hundred, on main as on branches: right after the panel's Unload returns 200, the residency read finds org/m loading again with a request in flight. The context probe the unload interrupted is a yield, and a yielded probe resumes; under the test's one-nanosecond quiet cadence it resumes on the next tick and loads the model back before the test reads residency. Seen on the macOS runner on PR 168's merged head and reproduced locally on origin/main (about 1 in 100 under -race).
