---
schema_version: 1
id: "iss-2610042101436891"
slug: "itd-2610030857275099-s-press-release-says-a-decision-model"
severity: "nitpick"
category: "inconsistency"
source: "impl-review"
found_during: "fidelity review rcp-c60839654fc3 of itd-2610030857275099, 2026-10-04"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped"
remedy: "Maintainer's decision 2026-10-07: the press release stands, so decision models are never marked. Remove the 'awaiting review' mark for decision models from the update check and the panel, with a test that a decision model shows no mark and offers no Update, and note in the intent's audit notes that criterion 9's mark is withdrawn."
---

itd-2610030857275099's press release says a decision model is never marked, but its acceptance criteria give one an 'awaiting review' mark, and the panel code shows it (internal/ui/static/app.js, around line 596). The fidelity review listed this as a promise delivered differently. A shipped intent's press release now contradicts its own criteria.
