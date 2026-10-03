---
schema_version: 1
id: "iss-2610031757379759"
slug: "measure-now-shown-for-non-chat"
severity: "nitpick"
category: "ux"
source: "agent-finding"
found_during: "adversarial review of the fix for iss-2610031010371709"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/ui/static/app.js"
remedy: "Show Measure now only for a model the models list marks chat."
resolution: "Measure now shows only when m.chat."
impact: fix
resolved_by:
  commit: "a0264236"
---

The panel showed Measure now for a model that cannot chat, which MeasureNow then refuses, so the operator saw an error alert for a button that could never work.
