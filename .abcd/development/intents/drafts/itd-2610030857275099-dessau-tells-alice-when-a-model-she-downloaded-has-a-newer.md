---
id: itd-2610030857275099
slug: dessau-tells-alice-when-a-model-she-downloaded-has-a-newer
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610030656210408, itd-2609081259493890]
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

- What happens when a newer version is found: a mark only, a mark with a
  one-click update, or an automatic update? Acting on an update safely needs a
  staged download at a pinned commit, a hash check of every file and an atomic
  swap (today a re-download over a changed repository can leave a mixed model:
  iss-2610030913179523), which the reviews route to its own intent.
- How the check runs, and whether it is on by default: see
  adr-2610030857208746 (proposed), which lays out six options. Under the
  2026-09-08 precedent an opt-in, off-by-default check needs no supersession of
  adr-2609201008476813; a check on by default does.
- The existing startup category job already asks the Hub about some
  repositories unasked (2026-09-21): was that within "needed to fetch models",
  or a reversal to ratify by name?
- What "newer" means: a change to a file Dessau downloads and reads, not a new
  commit (a README edit is a commit). Needs the commit and per-file hashes
  recorded at download, shared with the revision field spc-2610030846273729
  plans for decision models; models downloaded before that show "version
  unknown", never a mark.
- The token: checks send no HuggingFace token unless a repository refuses
  anonymous access (a repository's commit is readable anonymously, even when
  gated).
- Typed links: builds on itd-2610030656210408 (one revision field; decision
  models stay on their reviewed version and are never marked) and
  itd-2609081259493890 (the setting on all three surfaces); refines
  itd-2609100519003748 (where the panel shows it). The draft's earlier
  "refines itd-2609091301112705 / itd-2609201445423499" are withdrawn: a
  re-download already clears the measured context and the tool-call verdict.
- Seeded 2026-10-03 from the maintainer's request; routing confirmed at filing
  (SPLIT) and corrected after the two adversarial reviews the same day.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
