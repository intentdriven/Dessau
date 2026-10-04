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
remedy: "Record which promise stands: if decision models are marked 'awaiting review', say so in a successor record or the intent's audit notes (a shipped intent's text is not rewritten), so the press release is not read as the behaviour."
---

itd-2610030857275099's press release says a decision model is never marked, but its acceptance criteria give one an 'awaiting review' mark, and the panel code shows it (internal/ui/static/app.js, around line 596). The fidelity review listed this as a promise delivered differently. A shipped intent's press release now contradicts its own criteria.
