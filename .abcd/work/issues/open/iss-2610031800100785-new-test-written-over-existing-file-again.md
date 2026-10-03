---
schema_version: 1
id: "iss-2610031800100785"
slug: "new-test-written-over-existing-file-again"
severity: "minor"
category: "lapse"
source: "agent-finding"
found_during: "writing the gateway refusals for iss-2610031444343397"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gateway/sampling_test.go"
remedy: "Before any whole-file write of a new test file, test that the name is free (ls or git ls-files) and pick a name of its own; restore the overwritten file from HEAD."
---

A new test was written to internal/gateway/sampling_test.go with a whole-file write, over the existing file of that name and its eleven sampling tests; the build caught it before any commit. The second time in this session after iss-2610031757095502 (internal/runtime/drain_test.go): a whole-file write to a test file's name is not checked against what is there.
