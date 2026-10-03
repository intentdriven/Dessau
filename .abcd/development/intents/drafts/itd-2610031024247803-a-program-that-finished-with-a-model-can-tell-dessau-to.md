---
id: itd-2610031024247803
slug: a-program-that-finished-with-a-model-can-tell-dessau-to
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609081259493890, itd-2609061441241254, itd-2609061441285238, itd-2609211335097114]
severity: minor
origin: researcher-authored
production_mode: hand-written
related_issues: [iss-2610031018046897]
---

# A program can ask Dessau to unload a model it has finished with

## Press Release

> A program that finished with a model can tell Dessau to unload it. Bob's test script runs a round of requests against one of Alice's models and then asks Dessau, through the same API it used for the requests, to unload that model, so the memory is free for the next round without anyone clicking Unload in the control panel; a script on Alice's own Mac can use the control panel's documented unload as well. Alice can switch either way off in Settings; both are on until she does.

## Why This Matters

> _Why this matters to the user — replace before planning._

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

- FLAGGED: a switch on the control panel's existing unload route cannot be
  enforced; the panel's own Unload button calls the same route and a script on
  this Mac is indistinguishable from it. Dropping that switch leaves the docs
  half as documentation (iss-2610031018046897).
- FLAGGED (reversal): the API route would let a client unload a pinned model,
  reversing itd-2609061441241254 ("nothing Bob or Carol requests can push them
  out of memory") and the 2026-09-21 "the pin wins" decision, unless it refuses
  pinned models.
- Who may use the API route: on a keyless install every network client and any
  web page could reach it; the reviews propose admitting exactly the callers the
  gateway already entitles (this Mac on a keyless install; key holders and this
  Mac on a keyed one; paired clients), and requiring a JSON content type.
- Eviction grace: a client unloading between Carol's turns forces a cold reload
  each turn; refuse inside the grace window?
- The planned panel limit (itd-2610031004535845) promises the model API is
  unchanged; does "this account only" also close the API unload to other
  accounts?
- The route: `POST /v1/dessau/unload` with the model in the body (model ids
  contain a slash; `DELETE /v1/models/{id}` is OpenAI's delete and is never
  registered); the API unload does not interrupt idle work and answers 409 at
  once; logs and statistics name the kind of caller, never the key, with a new
  stop reason.
- A state-changing verb on `/v1` is architecture: the reviews propose an ADR
  settling the route, the access rule and the pin rule.
- Seeded 2026-10-03 from the maintainer's request; revised after two
  adversarial reviews the same day.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
