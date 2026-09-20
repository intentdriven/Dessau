---
schema_version: 1
id: "iss-2609202010356183"
slug: "a-transient-probe-failure-is-retried-only-when-the-model-is"
severity: "nitpick"
category: "documentation"
source: "impl-review"
found_during: "adjudication of the three reviews of the tool-call probe (itd-2609201445423499), pilot run 2026-09-20"
origin: researcher-authored
production_mode: hand-written
found_at: "docs/context-probe.md"
---

A transient probe failure is retried only when the model is next loaded, while the measurements page says the model is asked again at its next serve. A transport error, a non-200 or a timeout records nothing, as the spec requires, and the probe is queued again only when the model next reaches loaded; a model that stays resident is not re-asked on its next request. Spec-consistent (cond-2609201451063083 re-runs on model or runtime change), but the page's wording reads as sooner than the code delivers; noted by the Opus 4.8 adjudicator as missed by all three reviews.
