---
id: itd-2610030932556747
slug: dessau-chat-offers-the-versions-of-a-model-as-choices-each
spec_id: spc-2610030950592909
kind: standalone
suggested_kind: null
reclassification_history: []
blocked_by: [itd-2610030932551549]
builds_on: [itd-2610030932551549, itd-2609170718430553, itd-2609200829199959]
severity: minor
origin: researcher-authored
production_mode: hand-written
impact: additive
---

# Dessau Chat offers a model's builds as described choices

## Press Release

Dessau Chat shows a model's builds as described choices.

When Carol opens the model picker and Alice's server holds two builds of one
model, she sees that model once with its builds beneath it, each described in
plain words from its facts, such as "4-bit · 4.6 GB · quicker to load". She
picks one for this chat, can make it her default on this server without
changing her default on any other, and is never moved to another build without
her say.

## Why This Matters

Carol sees near-duplicate repository names in the picker with no way to tell
which to choose. Seeded 2026-10-03 from the maintainer's request; the server
half is itd-2610030932551549.

## Mechanism

- We expect Carol to pick the build she meant more often because the builds sit
  together, described. Shown wrong if testers still pick by repository name.

## Scope Conditions

None stated.

## Acceptance Criteria

- **Given** a server publishing builds, **when** Carol opens the picker,
  **then** each model with several chat builds appears once with its builds
  beneath it, each described from its facts (for example "4-bit · 4.6 GB ·
  quicker to load").
- **Given** a model with only one chat build on the server (including a group
  where only one build is a chat model), **when** Carol opens the picker,
  **then** it appears as a plain row.
- **Given** Carol picks a build, **when** she sends, **then** it changes this
  chat only; "Set as Default" makes it her default on this server, and her
  default on any other server is unchanged.
- **Given** Carol's default build was deleted from the server, **when** she
  opens Dessau Chat, **then** the picker says it is gone and offers that model's
  other builds; nothing changes until she picks.
- **Given** a server without the new build information, **when** Carol opens
  the picker, **then** she sees today's plain list, unchanged.
- **Given** the Model menu, **when** Carol opens it, **then** it offers the same
  builds and descriptions as the picker.

## Decisions at the planning interview (2026-10-03)

1. A pick changes this chat only; "Set as Default" sets the default per server
   (today's one global default becomes one per server).
2. A deleted default is shown as gone with its siblings offered; Dessau Chat
   never switches model by itself.
3. Descriptions are worded by Dessau Chat from the server's facts.
4. Waits on itd-2610030932551549 (blocked by it); planned separately.
5. The maintainer chose to record no scope condition for this intent.

Typed links: builds on itd-2610030932551549 and itd-2609170718430553; refines
itd-2609200829199959 (the picker, the Model menu, "Set as Default").

## Open Questions

_None open._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: servers now hold several builds of one model, and clients see near-identical names with no way to choose; shown wrong if, once grouped, nobody chooses between builds differently than before. And programs that call Dessau need to pick the build that fits their job without hard-coding repository names; shown wrong if such tools keep hard-coding names anyway.
