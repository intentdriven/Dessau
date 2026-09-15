---
schema_version: 1
id: "iss-2609100515475279"
slug: "the-self-test-s-results-have-no-panel-view-the-switch-is-on"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "itd-2609100457007827 build"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/ui/static/app.js"
resolution: "GET /api/selftest on the control plane (latest run per model, fifty most recent, the switch read live) and a Self-test block at the head of the Statistics tab, fetched on the tab's own tick; docs/self-test.md says so"
impact: additive
resolved_by:
  intent: "itd-2609100457007827"
---

The self-test's results have no panel view: the switch is on the Settings pane but the figures are only in selftest/results.jsonl, so the accessible surface shows the operator how to start measuring and nothing of what was measured. A results view on the Statistics or Models tab, reading the file through a loopback endpoint, closes the gap the three-surfaces rule names.

## Grounds

- pursued: we expect the latest run per model, shown beside the request statistics, to be enough for an operator to size a served window and a concurrency without opening the file; shown wrong if readers reach for the file for a figure the block does not carry
