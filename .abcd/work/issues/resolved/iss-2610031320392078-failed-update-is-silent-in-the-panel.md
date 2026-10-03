---
schema_version: 1
id: "iss-2610031320392078"
slug: "failed-update-is-silent-in-the-panel"
severity: "minor"
category: "ux"
source: "impl-review"
found_during: "adversarial review of spc-2610030929021692 step 4"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/ui/static/app.js"
remedy: "Record the last update's failure on the model (beside UpdateCheck, cleared by the next success) and show it in the card's version line."
resolution: "The model records a failed update as one of five classes, cleared by the next success, and the card's version line says it."
impact: fix
resolved_by:
  commit: "6015494cf7c97eed3670beeaaa2cf0cef8d2f51d"
---

A failed update is silent in the control panel. When a staged update fails — a hash that does not match, a version the launch checks refuse, a decision model whose commit is not reviewed, a full disk — it is only logged: the progress bar disappears and the card offers Update again with no word of why, and a model of unknown version is offered an Update that may then fail the same way. Found by the adversarial review of spc-2610030929021692 step 4 on 2026-10-03.
