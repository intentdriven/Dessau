---
id: itd-2610031004535845
slug: alice-decides-who-on-her-mac-can-open-dessau-s-control-panel
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609081259493890, itd-2609100519003748]
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# Alice decides whether other accounts on her Mac can open the control panel

## Press Release

> Alice decides who on her Mac can open Dessau's control panel. In Settings she chooses between her own account only and anyone on this Mac: with her account only, Bob, who has his own account on the same Mac, can still use the models through the API but cannot open the panel to change settings, start downloads or delete models.

## Why This Matters

> _Why this matters to the user — replace before planning._

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

- FLAGGED: "this account only" needs a way to tell which account a request on
  this Mac comes from. adr-2609091123526871 (section 7) records that no check can
  tell other accounts apart ("from the socket they are the same connection"),
  and adr-2610030906462776 (Alternative 3) counts other accounts opening the
  panel as part of its design. The design review measured a way that works:
  asking the kernel which account owns the connecting socket (libproc), about
  1.3 ms per scan, failing closed because another account's processes are not
  visible. Adopting it means a new ADR refining the first and superseding the
  second in part. Precondition: a spike proving Safari, Chrome and Firefox
  connections can be attributed, or Alice locks herself out.
- Rejected by the review: a cookie set by the menu-bar link (cookies are not
  scoped to a port, so another account listening on another localhost port can
  receive and replay it); a Unix socket (browsers cannot reach it); a macOS
  password prompt (the server cannot tell whose screen to prompt).
- The default: "this account only" (safer; breaking for anyone administering
  from a second account) or "anyone on this Mac" (today's behaviour).
- Lock-out: switching to "this account only" is accepted only from a
  connection already identified as the serving account's; recovery is the menu
  bar or a hand edit of `config.json`, never the panel.
- Who it protects: a standard account; an administrator can act as Alice.
- The model API is unchanged under either choice.
- The network option was declined by the maintainer at routing (2026-10-03),
  the third time it has been set aside (adr-2609091123526871,
  adr-2609081118587999).
- Typed links: builds on itd-2609081259493890 (three surfaces); refines
  itd-2609100519003748 (its condition cond-2609201007364804: "reachable by every
  account" is not a property that intent changes; this one does).
- Seeded 2026-10-03 from the maintainer's request; revised after two
  adversarial reviews the same day.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
