---
id: itd-2609211335097114
slug: a-stuck-model-never-holds-the-server-hostage-alice-asks-her
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# A stuck model never holds the server hostage. Alice asks her Dessau Server a question and gets an answer, even while the server is busy with work it started on its own: an idle job such as the context probe yields the moment a real request needs the memory it holds — the step is abandoned, the model it was measuring is released, and the request loads its model as if nothing had been in the way. When something does hold the memory, Alice can see it: the control panel names the model, the job holding it and for how long, and offers one click to free it; a client refused for lack of memory is told what holds it and that it is the server's own work, not the size of the model asked for. A job that finds its model unresponsive gives up in seconds, not after a timeout sized for a hundred thousand tokens, and does not pick the same model again until something about it has changed.

## Press Release

> _Seeded from a quoted-text intent capture. Expand into the full press-release narrative before planning._

## Why This Matters

A stuck model never holds the server hostage. Alice asks her Dessau Server a question and gets an answer, even while the server is busy with work it started on its own: an idle job such as the context probe yields the moment a real request needs the memory it holds — the step is abandoned, the model it was measuring is released, and the request loads its model as if nothing had been in the way. When something does hold the memory, Alice can see it: the control panel names the model, the job holding it and for how long, and offers one click to free it; a client refused for lack of memory is told what holds it and that it is the server's own work, not the size of the model asked for. A job that finds its model unresponsive gives up in seconds, not after a timeout sized for a hundred thousand tokens, and does not pick the same model again until something about it has changed.

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
