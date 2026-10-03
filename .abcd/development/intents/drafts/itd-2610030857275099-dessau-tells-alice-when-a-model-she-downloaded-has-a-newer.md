---
id: itd-2610030857275099
slug: dessau-tells-alice-when-a-model-she-downloaded-has-a-newer
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# Dessau checks downloaded models for newer versions

## Press Release

> Dessau tells Alice when a model she downloaded has a newer version. When Dessau starts, and then at an interval Alice chooses in Settings, it asks Hugging Face whether each model she downloaded through it has changed since she fetched it, and the control panel marks the ones with a newer version, so she is not left serving an old revision without knowing. Bob, who never opens Settings, sees the same mark in the panel and in config.json, because the interval is set in both places and the panel and Go say the same thing.

## Why This Matters

> _Why this matters to the user — replace before planning._

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

- What happens when a newer version is found: a mark only, a one-click update,
  or an automatic update (and whether a loaded model is replaced in place)?
- Off or on by default, and the default interval? Gated by
  adr-2610030857208746 (proposed), which reverses part of
  adr-2609201008476813: Dessau's outbound connections today are only those
  needed to fetch models and provision the runtime, and nothing about the
  Mac's models leaves it.
- How "newer" is judged: the hub downloads `main` and records no revision
  today, so models downloaded before this lands have no known revision.
- Gated or private repositories that need a token; a repository renamed or
  deleted upstream; behaviour when offline.
- Typed links to confirm at the interview: refines itd-2609091301112705 (a newer
  version's measured context no longer applies) and itd-2609201445423499 (nor
  does its tool-calling verdict); refines itd-2610030656210408 (decision models
  stay on their reviewed version and are never offered an update).
- Seeded 2026-10-03 from the maintainer's request; routing confirmed at filing
  (SPLIT: this capability, the ADR above, revision tracking in the spec).

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
