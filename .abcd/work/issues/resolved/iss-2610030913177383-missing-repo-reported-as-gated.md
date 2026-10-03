---
schema_version: 1
id: "iss-2610030913177383"
slug: "missing-repo-reported-as-gated"
severity: "minor"
category: "ux"
source: "plan-review"
found_during: "adversarial design review of itd-2610030857275099"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/hub/hub.go"
remedy: "On 401 without a token, say the repository is missing, private or gated, and suggest a token only for the gated case; when a token was sent, say access was refused."
resolution: "A refusal without a token now says the repository is missing, private or gated; one with a token says the token was refused (hub.APIError.TokenSent), so a deleted or renamed model no longer sends the operator to add a token. The update check sends no token unless refused, per adr-2610030857208746."
impact: fix
resolved_by:
  commit: "5edd2a1"
---

A model that no longer exists on HuggingFace is reported as possibly gated. Anonymous requests for a missing repository get HTTP 401, and APIError.Error in internal/hub/hub.go maps 401 and 403 to 'the repo may be gated; add an access token in Settings', so an operator whose model was deleted or renamed upstream is sent to add a token that cannot help. Measured by the design review of itd-2610030857275099 on 2026-10-03.
