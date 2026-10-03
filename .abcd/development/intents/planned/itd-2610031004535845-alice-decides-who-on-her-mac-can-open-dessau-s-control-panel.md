---
id: itd-2610031004535845
slug: alice-decides-who-on-her-mac-can-open-dessau-s-control-panel
spec_id: spc-2610031016319710
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609081259493890, itd-2609100519003748]
severity: minor
origin: researcher-authored
production_mode: hand-written
impact: additive
---

# Alice decides whether other accounts on her Mac can open the control panel

## Press Release

Alice decides whether other accounts on her Mac can open Dessau's control
panel.

On the Mac she shares with Bob, Alice switches "Who can open the control panel"
from anyone on this Mac to her own account only. From then on, when Bob opens
the panel from his own account, the page tells him it belongs to the account
that runs Dessau and nothing changes, while his editor goes on getting answers
from Dessau as before. Alice opens the panel from the menu bar, a bookmark or
the address typed by hand, as she always has.

## Why This Matters

Today every account that can sign into the Mac can change settings, start
downloads and delete models through the panel, because it asks no one for
anything (documented on 2026-10-03 when Dessau became a one-account server).
People Alice trusts to use models are not necessarily people she wants running
the server. Seeded 2026-10-03 from the maintainer's request; the network option
it named was declined (decision line of the same day).

## Mechanism

- We expect the limit to hold because macOS itself reports which account owns
  the program that connected, and another account's programs are invisible to
  Dessau. Shown wrong if Bob, from his own account, reads or changes anything in
  the panel, or if Alice is ever refused from her own browser.

## Scope Conditions

- One Mac and no network: both choices are about accounts on this Mac, and the <!-- cond: cond-2610031016316867 -->
  panel stays unreachable from other machines under either.
- Standard accounts: the limit protects against standard accounts like Bob's; <!-- cond: cond-2610031016313952 -->
  an administrator account can act as Alice and is not kept out.
- Accounts, not programs: anything running in Alice's own account is treated as <!-- cond: cond-2610031016313865 -->
  Alice.
- The model API is unchanged: who can use the models through the API is not <!-- cond: cond-2610031016316262 -->
  changed by this setting.

## Acceptance Criteria

- **Given** "this account only", **when** Bob's browser or any program in his
  account on the same Mac opens the panel or any of its actions, **then** he is
  refused with a reason that names no account, and nothing changes.
- **Given** "this account only", **when** Alice opens the panel from the menu
  bar, a bookmark, or the address typed into Safari, Chrome or Firefox, or runs
  the `dessau` command, **then** each works with no extra step.
- **Given** either choice, **when** Bob uses the models through the API from his
  account, **then** it works exactly as before.
- **Given** "anyone on this Mac" (the default), **when** Bob opens the panel,
  **then** it behaves as it does today; a save that leaves the access setting
  untouched succeeds, and a change to it from Bob's account is refused, with the
  reason.
- **Given** "this account only", **when** a connection reaches the panel whose
  account Dessau cannot identify, **then** it is refused.
- **Given** Bob has the panel open, **when** Alice switches to "this account
  only", **then** Bob's open page stops updating and his next action is refused.
- **Given** the setting, **when** it is read from Go, the panel or
  `config.json`, **then** all three agree, and a value Dessau does not recognise
  counts as "this account only".
- **Given** Alice cannot open the panel, **when** she uses the menu bar or edits
  `config.json` by hand, **then** her access is restored; recovering never needs
  the panel.
- **Given** either choice, **when** a request for the panel comes from another
  machine, **then** it is refused, as today.

## Decisions at the planning interview (2026-10-03)

1. Two choices, this account only and anyone on this Mac; no network option
   (declined at routing; dated decision line).
2. A new ADR records that Dessau can tell the serving account's connections
   apart by asking the kernel which account owns the connecting socket,
   refining adr-2609091123526871 and superseding in part the panel clause of
   adr-2610030906462776.
3. The default is anyone on this Mac (today's behaviour), so the impact is
   additive; only Alice's choice narrows access.
4. Mechanism chosen on the design review's measurement: the kernel's account
   for the connecting process, not a cookie (cookies are not scoped to a port)
   and not a password prompt; a spike across Safari, Chrome and Firefox comes
   first.

Typed links: builds on itd-2609081259493890 (three surfaces); refines
itd-2609100519003748 (cond-2609201007364804).

## Open Questions

_None open._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: Dessau now serves from one account on Macs that others also sign into, and they should use models without being able to change settings or delete models; shown wrong if no one who shares a Mac ever turns the limit on. And today any account on the Mac can administer the server with no password, a documented gap; shown wrong if the limit, once on, still lets another account change anything.
