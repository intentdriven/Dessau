---
schema_version: 1
id: "iss-2609161713001858"
slug: "testtheprobemeasuresthroughtherealgatewayandpool-in-internal"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "PR 54 merge queue"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gateway/probe_integration_test.go"
---

TestTheProbeMeasuresThroughTheRealGatewayAndPool in internal/gateway/probe_integration_test.go flaked in the merge queue (PR 54, run 34987185388): 'a job is still reported running' with the step at 'bisecting at 18432 tokens'. The test polls until the measurement is written, then asserts Status().Job is empty at once, but the runner clears the job only after Run returns, so the check races the tail of the run. Wait for the job to clear under the same deadline before asserting.
