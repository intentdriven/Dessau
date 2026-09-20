---
id: itd-2609201355512965
slug: bob-keeps-a-conversation-as-a-file-dessau-chat-exports-a-who
spec_id: spc-2609201444537572
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609151701196720]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# Bob keeps a conversation as a file: Dessau Chat exports a whole conversation to wherever he chooses

## Press Release

Bob has spent an afternoon working something out with a model and wants to
keep it. From the conversation's own menu he chooses **Export…**, a save
panel opens with the conversation's title as the file name and a choice of
Markdown or JSON, and one file lands where he put it: every turn in order,
his and the model's, each with the model that answered, the time and the
style it was answered under. Markdown reads in any editor and pastes into a
document; JSON is the client's own shape, and Dessau Chat can read it back
in as the same conversation. A **Detailed export** switch in Settings adds
what a curious person wants and a reader does not: the reasoning shown
above replies, token counts, timings and the statistics the server reports.
On the iPad the same choice opens the share sheet, so the file goes to
Files, Mail or another app. In the app's menu bar, and nowhere in any
window, **Export All…** writes one folder holding every conversation. The
conversation itself stays in Dessau Chat exactly as before; export copies it
out, it never moves it. Nothing is sent anywhere: the file is written by the
client from what it already holds.

## Why This Matters

Dessau Chat keeps every conversation and reopens it after a relaunch, but
what it holds is its own: a conversation cannot leave the app except by
copying replies one at a time. A person who wants to keep, share or quote
a whole conversation has no honest way to do it. Export is the missing
half of "the conversation is yours": it is stored for you, and it is yours
to take.

## Mechanism

Confirmed by the maintainer at the 2026-09-20 interview: we expect a
Markdown export to be the file people keep because a conversation is prose
with code in it and Markdown is what every editor and note app reads. What
would show this wrong: testers asking for a format nothing here writes.

## Scope Conditions

Confirmed by the maintainer at the 2026-09-20 interview:

- The chat client only, on the Mac and the iPad. <!-- cond: cond-2609201444537168 -->
- Markdown and JSON, chosen in the save panel; JSON is the client's own <!-- cond: cond-2609201444537911 -->
  store shape, so it can be re-imported.
- Every export carries the model's name, the time and the style mark per <!-- cond: cond-2609201444539322 -->
  turn; the Detailed export setting adds the reasoning, token counts,
  timings and the statistics the server reports.
- Export All lives in the app's menu bar only, never in a window. <!-- cond: cond-2609201444533682 -->
- Nothing is sent anywhere; the file is written from what the client holds. <!-- cond: cond-2609201444536551 -->

## Acceptance Criteria

Confirmed by the maintainer at the 2026-09-20 interview, every bullet walked
and accepted:

- Given a conversation is open, when Bob chooses Export from its menu on the
  Mac, then a save panel opens named after the conversation and offering
  Markdown or JSON, and the file it writes holds every turn in order with
  who said it.
- Given the same on the iPad, when Bob chooses Export, then the share sheet
  opens with the same file.
- Given the file has been written, when Bob reopens the conversation, then
  it is unchanged, and no server has been sent anything.
- Given a reply, when it is exported, then the turn carries the model's
  name, the time and the style mark; with Detailed export on, the reasoning,
  the token counts, the timings and the server's statistics too.
- Given a JSON export, when it is imported, then it reads as the same
  conversation.
- Given the app's menu bar, when Bob chooses Export All, then one folder is
  written holding every conversation, and no window carries a button for it.
- Given a conversation with a hundred turns, when it is exported, then the
  file is written whole and the app stays responsive.

## Open Questions

All three were put to the maintainer at the 2026-09-20 interview and are
decided: Markdown and JSON chosen in the save panel; the model's name, the
time and the style mark always, and the rest under a Detailed export
setting; Export All as a menu-bar item only.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: the conversation is stored for the person and should be theirs to take; wrong if nobody exports, or if what they want is a sync rather than a file
