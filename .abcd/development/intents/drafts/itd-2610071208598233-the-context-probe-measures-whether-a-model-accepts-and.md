---
id: itd-2610071208598233
slug: the-context-probe-measures-whether-a-model-accepts-and
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2610041945033770]
origin: extracted-from-record
production_mode: hand-written
---

# The context probe measures whether a model accepts and answers a prompt of a given size, not whether it can use that much context. Its measured window says a prompt that long loaded, prefilled and got an answer before a bound stopped the probe. It does not show that the model retrieves or reasons over content near the start of that window. Published long-context evaluations show the two differ widely. RULER (arXiv:2404.06654) found models that score near-perfectly on vanilla needle-in-a-haystack retrieval yet degrade sharply as context grows, with few holding up at 32K despite claiming more. HELMET (2024, updated 2025) reaches the same conclusion with broader tasks. A served window set from the probe can therefore advertise context a model cannot actually use.

## Press Release

> _Seeded by promotion from iss-2610041945033770. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2610041945033770`: The context probe measures whether a model accepts and answers a prompt of a given size, not whether it can use that much context. Its measured window says a prompt that long loaded, prefilled and got an answer before a bound stopped the probe. It does not show that the model retrieves or reasons over content near the start of that window. Published long-context evaluations show the two differ widely. RULER (arXiv:2404.06654) found models that score near-perfectly on vanilla needle-in-a-haystack retrieval yet degrade sharply as context grows, with few holding up at 32K despite claiming more. HELMET (2024, updated 2025) reaches the same conclusion with broader tasks. A served window set from the probe can therefore advertise context a model cannot actually use.. Read that issue record for the source observation.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

_None recorded yet._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
