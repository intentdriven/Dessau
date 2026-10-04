---
id: itd-2610030857275099
slug: dessau-tells-alice-when-a-model-she-downloaded-has-a-newer
spec_id: spc-2610030929021692
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610030656210408, itd-2609081259493890, itd-2609100519003748]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# Dessau checks downloaded models for newer versions, and updates them with one click

## Press Release

Dessau tells Alice when a model she downloaded has a newer version, and updates
it with one click.

Alice turns on update checks in Settings and picks how often, once a day to
start with. From then on, at every start and every interval, Dessau asks
HuggingFace whether the files of each model she downloaded have changed, and
the control panel marks the ones that have. One click on Update fetches the new
version beside the old one and swaps it in only once every file checks out, so
Carol's chat never meets a half-updated model. Nothing is asked until Alice
turns checks on, and a decision model is never marked.

## Why This Matters

Downloads take a repository's latest files once and are never revisited. Fixes
to chat templates, configs and quantisations land on HuggingFace silently, so
Alice serves a stale build and finds out only when a client misbehaves. And the
one way to update today, downloading again, can leave a model that is half the
old version and half the new (iss-2610030913179523). Seeded 2026-10-03 from the
maintainer's request.

## Mechanism

- We expect Alice to update models she would otherwise leave stale, because the
  mark appears in the panel she already uses. Shown wrong if marked models stay
  un-updated release after release.
- We expect Alice to act on marks because they appear only when a file Dessau
  actually uses has changed, never for a README edit. Shown wrong if a mark
  appears for an upstream change that touches no file Dessau reads.

## Scope Conditions

- Only when Alice has turned checks on; a Mac whose operator never does sends <!-- cond: cond-2610030929026824 -->
  nothing.
- A model is marked only if Dessau recorded which version it downloaded; older <!-- cond: cond-2610030929020636 -->
  downloads show "version unknown" until updated once.
- Models from HuggingFace; public ones are checked without the token, private <!-- cond: cond-2610030929022838 -->
  ones with it only when an anonymous request is refused.
- The Mac is online (offline, marks stay as they were); a decision model is <!-- cond: cond-2610030929025733 -->
  checked, but offered an update only once a Dessau release has reviewed that
  version.

## Acceptance Criteria

- **Given** checks are off, **when** Dessau runs, **then** no check leaves the
  Mac, and saving any other setting never turns checks on.
- **Given** checks are on, **when** Dessau starts (if the last check is older
  than the interval) and each interval passes, **then** every eligible model is
  checked and changed ones are marked.
- **Given** an upstream change that touches only files Dessau does not use (a
  README edit), **when** a check runs, **then** no mark appears.
- **Given** a model whose version Dessau never recorded, **when** Alice opens
  its card, **then** it says "version unknown", shows no mark, and offers
  Update.
- **Given** a token in Settings and a public model, **when** a check runs,
  **then** the request carries no token; a private model gets it only after an
  anonymous request is refused.
- **Given** HuggingFace cannot be reached, **when** a check runs, **then** marks
  stay as they were, one line is logged, and serving is not slowed.
- **Given** a marked model, **when** Alice clicks Update, **then** the new
  version downloads beside the old one at that exact upstream version, every
  file is checked, and it is swapped in only when complete; if anything fails
  the old one keeps serving, so Carol's chat never meets a half-updated model.
- **Given** a newer version that ships its own code, **when** it is checked,
  **then** the mark says Dessau will not run that version and offers no Update.
- **Given** a decision model whose upstream version changed, **when** Alice
  opens its card, **then** it says a newer version exists and will be offered
  once reviewed, with no Update; Update appears when a Dessau release ships the
  reviewed version.
- **Given** the setting (on or off; an interval from 1 hour to 30 days, daily by
  default), **when** it is read from Go, the panel or `config.json`, **then** all
  three agree; an out-of-range value in `config.json` is repaired on load, and
  saving an unrelated setting is never refused over it.
- **Given** a model or file name from HuggingFace containing markup, **when** the
  panel shows a mark, **then** the name appears as plain text.

## Decisions at the planning interview (2026-10-03)

1. On an update: a mark plus a one-click update. The safe update (staged at the
   exact upstream version, every file checked, atomic swap, the old copy kept
   until the new one is in) is part of this intent; the reviews had proposed it
   as a separate one.
2. When: off until the operator turns checks on. Under the 2026-09-08
   precedent this needs no supersession of adr-2609201008476813;
   adr-2610030857208746 records the rule for this check.
3. The existing startup category lookup counts as part of fetching models
   (recorded in adr-2610030857208746); it stays always on.
4. The token: sent only when an anonymous request is refused.
5. Models with no recorded version: "version unknown" plus Update.
6. Interval: daily by default, 1 hour to 30 days.
7. Decision models: checked, with no Update until a Dessau release has
   reviewed the version (the maintainer asked why they should differ; the
   answer is itd-2610030656210408's "reviewed version only").

Typed links: builds on itd-2610030656210408 (one recorded-revision field;
"reviewed version only"), itd-2609081259493890 (three surfaces) and
itd-2609100519003748 (where the panel shows a model's state). Resolves, when
built, iss-2610030913179523 (a re-download can leave a mixed model), since the
update path replaces it.

## Open Questions

_None open._ Settled in spc-2610030929021692: the ordinary re-download moves
onto the same staged path as Update (its step 3), which resolves
iss-2610030913179523.

## Audit Notes

<!-- abcd-review: OWED receipt=rcp-c60839654fc3 -->
Fidelity review OWED (receipt rcp-c60839654fc3).
<!-- abcd-review-end receipt=rcp-c60839654fc3 -->

## Grounds

- pursued: fixes to chat templates, settings and quantisations land on HuggingFace silently, so Alice serves stale builds without knowing; shown wrong if, over a month of daily checks, no model she uses changes in a file Dessau reads. And the only way to update today, downloading again, can leave a half-old, half-new model, which a safe one-click update fixes; shown wrong if no one ever re-downloads a changed model.
