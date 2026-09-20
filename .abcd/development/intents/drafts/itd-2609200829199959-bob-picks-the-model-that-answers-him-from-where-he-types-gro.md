---
id: itd-2609200829199959
slug: bob-picks-the-model-that-answers-him-from-where-he-types-gro
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609170718438919, itd-2609170718430553]
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# Bob picks the model that answers him from where he types. DessauChat opens on the Mac's own model and answers at once. A control at the start of the message pill, which floats over the transcript, carries the name of the model in use and opens a menu: On this Mac first, then one section per server found on the network with that server's chat models under its name, a lock where a key is needed, a greyed row saying No response where a server has gone quiet, and a server Bob has paired with shown as paired; a server appears once however many names the network gives it. The first time servers appear on the network, a line above the composer offers one of them in one click and can be dismissed, so Alice's server is found without a tour. Settings holds two choices rather than a list: the default model for a new chat, and whether a chat falls back to the Mac's own model when its server cannot be reached, on by default; starred models sit at the top of the menu. A new chat starts on the default, changing the model in a chat changes that chat only, and Set as Default promotes it. A fallback happens only between turns, never mid-reply, and is written into the transcript as a row Bob can read and retry from; the client never moves a conversation from the Mac to a server without asking him. Where the Mac's own model is unavailable, its row says which of the three reasons applies, not eligible, not switched on, or not ready yet, and offers the fix where there is one. The same menu is in the menu bar under Model.

## Press Release

> _Seeded from a quoted-text intent capture. Expand into the full press-release narrative before planning._

## Why This Matters

Bob picks the model that answers him from where he types. DessauChat opens on the Mac's own model and answers at once. A pop-up button beside Send carries the name of the model in use and opens a menu: On this Mac first, then one section per server found on the network with that server's chat models under its name, a lock where a key is needed, a greyed row saying No response where a server has gone quiet, and a server Bob has paired with shown as paired; a server appears once however many names the network gives it. The first time servers appear on the network, a line above the composer offers one of them in one click and can be dismissed, so Alice's server is found without a tour. Settings holds two choices rather than a list: the default model for a new chat, and whether a chat falls back to the Mac's own model when its server cannot be reached, on by default; starred models sit at the top of the menu. A new chat starts on the default, changing the model in a chat changes that chat only, and Set as Default promotes it. A fallback happens only between turns, never mid-reply, and is written into the transcript as a row Bob can read and retry from; the client never moves a conversation from the Mac to a server without asking him. Where the Mac's own model is unavailable, its row says which of the three reasons applies, not eligible, not switched on, or not ready yet, and offers the fix where there is one. The same menu is in the menu bar under Model.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

- **The maintainer proposed a welcome sheet in which Bob must choose a model before his first message.** The SOTA note (`research/notes/2026-09-20-model-picker-sota.md`, findings 1 and 8) argues against on measured evidence and on the HIG, and this draft is written without it; the on-network offer line is the proposed replacement for the demonstration moment. For the maintainer to decide at the interview.
- **The maintainer proposed ranked preferences across servers.** Finding 5 narrows that to one default, one fixed fallback and favourites; the draft is written in that shape. For the maintainer to decide.
- **Where the picker lives — decided 2026-09-20.** The composer is a pill floating over the transcript; the picker sits at its start, labelled with the model and where it runs, and collapses on a narrow window to a small server glyph (a chip for a server, the Apple mark for this Mac) and chevron with the name in the tooltip; not a dot, which the styles intent itd-2609200850330402 reserves for the Circle mark (maintainer's decision at that intent's interview, 2026-09-20). No toolbar picker; a Model menu in the menu bar mirrors it. The maintainer chose this over the toolbar corner on the mock (DECISIONS.md, 2026-09-20). The top right of the window carries the answer styles instead, filed as their own draft.
- **Server identity for deduplication.** Finding 4 says never by display name. The server advertises no stable identity in its TXT record today; that server-side change is captured as iss-2609200830185068 (`internal/discovery`) so a server lane can take it independently of this interview. The client-side symptom is iss-2609200822241060. Until the id exists, the client deduplicates by the spki fingerprint where TLS is on and by instance name otherwise.
- **The on-device availability message.** This draft has the client read the three availability reasons and say so. That is the side already decided on iss-2609181116081228 at the 2026-09-20 interview (DECISIONS.md, 2026-09-20: the client keeps reading availability and shows its own message; the text-intelligence intent's promise is corrected). This draft cites that decision and does not reopen it; it refines the message into three states.
- **The API key for a newly found server.** iss-2609181116082211 records that a stored key is sent only to the origin it was saved for; the offer line and the per-server sections inherit that rule unless the interview changes it.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
