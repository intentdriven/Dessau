---
schema_version: 1
id: "iss-2609202259374942"
slug: "the-hub-completion-job-re-asks-a-repo-the-hub-does-not-answe"
severity: "nitpick"
category: "observation"
source: "impl-review"
found_during: "security review of the adopted-models chat fix (iss-2609202237468921), 2026-09-20"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/app/category.go"
---

The Hub-completion job re-asks a repo the Hub does not answer for at every start, without a cap on how many models one run may ask. In shared-cache mode another local account can plant complete-looking model directories with valid but non-existent repo ids; each costs one outbound Hub request per start, paced at one per second, never blocking startup, a load or a request, and exposing no token or host. A bounded residual of the documented choice that an unreachable Hub is retried at the next start; a per-start cap or a give-up after N silent starts is the shape of a later change if the log noise ever matters.
