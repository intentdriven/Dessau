---
schema_version: 1
id: "iss-2609200823571893"
slug: "the-ci-workflow-runs-on-every-branch-push-and-every-pull-req"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "2026-09-19 autonomous sweep, recorded 2026-09-20"
origin: researcher-authored
production_mode: hand-written
found_at: ".github/workflows/ci.yml"
---

The CI workflow runs on every branch push AND every pull request, so a burst of lane branches doubles the load on the five macOS runners and the merge queue starves; the push trigger should be restricted to main, since pull requests already cover every branch.
