---
id: itd-2610030932556747
slug: dessau-chat-offers-the-versions-of-a-model-as-choices-each
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610030932551549, itd-2609170718430553, itd-2609200829199959]
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# Dessau Chat offers a model's versions as described choices

## Press Release

> Dessau Chat offers the versions of a model as choices, each with its description. When Carol picks a model in Dessau Chat and Alice's server holds more than one version of it, Carol sees one model with its versions listed beneath it, each described in plain words, such as 'Faster, lighter' or 'More capable', instead of several cryptic repository names, and her choice is remembered for her next conversation.

## Why This Matters

> _Why this matters to the user — replace before planning._

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

- Remembering a choice: today Dessau Chat remembers one model globally, and a
  model change inside a chat changes that chat only ("Set as Default"
  promotes it; itd-2609200829199959). The draft's "remembered for her next
  conversation" is FLAGGED: as written it reverses that rule.
- What happens when the remembered build has been deleted: the client never
  switches model by itself today; offer the siblings and let the user pick?
- A server without the new fields must show today's flat list unchanged.
- Typed links: builds on itd-2610030932551549 (the server's fields) and
  itd-2609170718430553 (the client offers the server's models); refines
  itd-2609200829199959 (the picker and its Model-menu mirror).
- Seeded 2026-10-03 from the maintainer's request; revised after two
  adversarial reviews the same day.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
