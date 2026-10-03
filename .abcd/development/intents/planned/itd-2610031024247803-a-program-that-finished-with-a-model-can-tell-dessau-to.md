---
id: itd-2610031024247803
slug: a-program-that-finished-with-a-model-can-tell-dessau-to
spec_id: spc-2610031153301961
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609081259493890, itd-2609061441241254, itd-2609061441285238, itd-2609211335097114]
severity: minor
origin: researcher-authored
production_mode: hand-written
related_issues: [iss-2610031018046897]
impact: additive
---

# A program can ask Dessau to unload a model it has finished with

## Press Release

A program that has finished with a model can give its memory back.

Bob's test script runs a round of requests against one of Alice's models, then
tells Dessau it is done with it, and the memory is free for the next round
without anyone opening the control panel. A model Alice has pinned, or one
someone has just used, stays where it is, and a device Dessau does not trust is
refused. Alice can turn this off in Settings.

## Why This Matters

Today the only ways to free a model's memory are the panel's Unload button and
an undocumented control-panel route that answers only on this Mac. A program
that tests several models in turn waits for someone to click Unload (it
happened on 2026-10-03, when a test tool had to ask the maintainer to), and on
one Mac a model left loaded after a test blocks the next. Seeded 2026-10-03
from the maintainer's request.

## Mechanism

- We expect models to sit loaded for less time after a test round, because the
  program that used a model can say it is done the moment it is. Shown wrong if
  models stay loaded after batches as long as before.
- We expect nobody else to lose a model they are using, because pinned, busy
  and just-used models are refused. Shown wrong if Carol's chat ever has to wait
  for a reload because a program unloaded her model.

## Scope Conditions

- Trusted callers only: programs on this Mac, holders of the API key, and <!-- cond: cond-2610031153302285 -->
  paired clients; other devices on a network without a key, never.
- Only a loaded model that no one is using, that is not pinned and is not in <!-- cond: cond-2610031153304419 -->
  its protected time after use.
- A trusted program may unload any such model, not only ones it used; Dessau <!-- cond: cond-2610031153300110 -->
  does not track who used what.
- Unloading only: programs cannot load, delete or change models this way. <!-- cond: cond-2610031153308005 -->

## Acceptance Criteria

- **Given** an install without a key, **when** a program on this Mac asks to
  unload a loaded model that is idle and not pinned, **then** it is unloaded and
  the answer says so.
- **Given** an install without a key, **when** another device on the network asks
  to unload a model, **then** it is refused whatever the switch says, the model
  stays loaded, and the answer reveals nothing about what is loaded.
- **Given** an install with an API key, **when** a program on another machine asks
  with the key, **then** it may unload; without the key it is refused. A paired
  client may unload.
- **Given** a web page open in a browser on this Mac, **when** it tries to unload a
  model, **then** it is refused and nothing changes.
- **Given** a model Alice has pinned, **when** a trusted program asks to unload it,
  **then** it is refused and the model stays.
- **Given** a model that is answering a request, loading, or in its protected
  time after use, **when** a trusted program asks to unload it, **then** it is
  refused at once and the request finishes.
- **Given** the switch is off, **when** any program asks to unload through the API,
  **then** it is refused, and the Unload button in the control panel still works.
- **Given** the switch, **when** it is read from Go, the panel or `config.json`,
  **then** all three agree; a `config.json` that does not mention it means on, and
  saving an unrelated setting is never refused over it.
- **Given** a request to delete a model through the API (the way OpenAI's API
  deletes models), **when** any client sends it, **then** no file is deleted.
- **Given** a program unloaded a model, **when** Alice reads the statistics or the
  log, **then** they say a program unloaded it and what kind of caller it was
  (this Mac, an API key, a paired client), and never show the key.
- **Given** the documentation, **when** a script author reads it, **then** it names
  the API unload and the control panel's own unload, who can use each, and when
  each is refused.

## Decisions at the planning interview (2026-10-03)

1. No switch on the control panel's own unload route (it is the panel's Unload
   button too, and a script is indistinguishable from it); the route is
   documented, and who can use it follows the panel-access setting of
   itd-2610031004535845. Only the API route has a switch, on by default.
2. Trusted callers only: exactly the callers the gateway already entitles.
3. Pinned models are refused (the pin promise of itd-2609061441241254 holds).
4. Models in their eviction-grace window are refused.
5. Under itd-2610031004535845's "this account only", the API unload keeps the
   API's own rule; other accounts' trusted programs can still unload.
6. Recorded without a question (one defensible answer): the first
   state-changing verb on `/v1` is settled by adr-2610031153127219:
   its route, access rule and refusals.

Typed links: refines itd-2609061441241254 (pins hold), itd-2609061441285238
(grace holds), itd-2609211335097114 (unload behaviour) and itd-2610031004535845
(the API unload stays under API rules); builds on itd-2609081259493890 (three
surfaces); resolves the documentation half of iss-2610031018046897.

## Open Questions

_None open._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: programs that test several models in turn have to wait for someone to click Unload, as happened on 2026-10-03; shown wrong if test tools never use it. And on one Mac a model left loaded after a test blocks the next, so freeing it promptly lets more models be tried; shown wrong if test rounds are no faster with it.
