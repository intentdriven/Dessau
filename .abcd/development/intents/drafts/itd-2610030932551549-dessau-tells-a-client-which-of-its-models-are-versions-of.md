---
id: itd-2610030932551549
slug: dessau-tells-a-client-which-of-its-models-are-versions-of
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609091129451578, itd-2609061431463108, itd-2609081259493890, itd-2609100519003748]
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# Dessau's models list groups versions of the same model and describes each

## Press Release

> Dessau tells a client which of its models are versions of the same model, and how they differ. When Alice has downloaded two builds of one model, a 4-bit and an 8-bit or a Flash and a full one, Dessau's models list groups them as versions of that model and gives each a short description in plain words, such as 'Faster, lighter' or 'More capable, needs more memory', so a tool Bob writes can choose between them without decoding repository names.

## Why This Matters

> _Why this matters to the user — replace before planning._

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

- What counts as builds of the same model. HuggingFace's
  `base_model:quantized:<base>` tag marks most quantised builds; bf16 copies are
  tagged like fine-tunes, older repositories carry no tag, and adopted models
  have no Hub data. A fallback by name (the repository name minus a precision
  suffix, with the same model shape in `config.json`) would be Dessau's own
  rule: FLAGGED, it reverses "Gropius does not invent a taxonomy" in
  itd-2609091129451578 unless the maintainer rules otherwise.
- Flash and full are different models on the Hub (different bases and sizes),
  so no derived rule can pair them; only a family the operator declares could.
- Where descriptions come from: facts Dessau holds (precision, size on disk,
  served context) published as fields for clients to word, text the operator
  writes per model (a new setting on all three surfaces), or both. Dessau
  measures no "capability".
- Whether this and itd-2610030932556747 are planned as one bundle or as two
  intents, the client blocked by this one.
- Typed links: refines itd-2609091129451578 (categories; clients pick by them;
  every model stays callable by name, which holds); builds on
  itd-2609061431463108 (the models-list field convention),
  itd-2609081259493890 (three surfaces) and itd-2609100519003748 (the panel's
  model cards); decision models (itd-2610030656210408) never share a group with
  chat models.
- Seeded 2026-10-03 from the maintainer's request (split into server and
  client at filing); revised after two adversarial reviews the same day.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
