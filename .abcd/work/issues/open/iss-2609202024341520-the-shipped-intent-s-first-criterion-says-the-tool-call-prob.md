---
schema_version: 1
id: "iss-2609202024341520"
slug: "the-shipped-intent-s-first-criterion-says-the-tool-call-prob"
severity: "nitpick"
category: "inconsistency"
source: "review-followup"
found_during: "fidelity audit of itd-2609201445423499 (receipt rcp-89e6e831c5c0), pilot run 2026-09-20"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-2609201445423499-alice-s-server-knows-which-models-can-call-tools-a-one-time.md"
---

The shipped intent's first criterion says the tool-call probe runs when the model's context has been measured, while the spec and the code fire it when the model reaches loaded after the loading request is served. The fidelity audit judged ac-1 MET_WITH_CONCERNS on exactly this: the probe ran before any context measurement in the hand check, as the spec's Approach designed it. The behaviour is the spec's and is what the maintainer accepted at the interview; the criterion's wording is the artefact to reconcile, by the maintainer's decision, not the code.
