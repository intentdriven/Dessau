---
id: itd-2609201355512965
slug: bob-keeps-a-conversation-as-a-file-dessau-chat-exports-a-who
spec_id: null
kind: null
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
panel opens with the conversation's title as the file name, and one file
lands where he put it: every turn in order, his and the model's, readable
in any editor and pasteable into a document. On the iPad the same choice
opens the share sheet, so the file goes to Files, Mail or another app. The
conversation itself stays in Dessau Chat exactly as before; export copies
it out, it never moves it. Nothing is sent anywhere: the file is written by
the client from what it already holds.

## Why This Matters

Dessau Chat keeps every conversation and reopens it after a relaunch, but
what it holds is its own: a conversation cannot leave the app except by
copying replies one at a time. A person who wants to keep, share or quote
a whole conversation has no honest way to do it. Export is the missing
half of "the conversation is yours": it is stored for you, and it is yours
to take.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

Seeded by the filing session, unconfirmed; every bullet is a proposal for the
planning interview:

- Given a conversation is open, when Bob chooses Export from its menu on the
  Mac, then a save panel opens named after the conversation, and the file it
  writes holds every turn in order with who said it.
- Given the same on the iPad, when Bob chooses Export, then the share sheet
  opens with the same file.
- Given the file has been written, when Bob reopens the conversation, then
  it is unchanged, and no server has been sent anything.
- Given a reply that carried reasoning, when it is exported, then the file
  keeps what the interview decides about reasoning and says which model
  answered.
- Given a conversation with a hundred turns, when it is exported, then the
  file is written whole and the app stays responsive.

## Open Questions

- Format: Markdown, plain text, JSON, or a choice among them; whether the
  file can be re-imported later.
- What travels with the turns: the model's name, the answer-style mark, the
  time of each turn, the reasoning shown above a reply.
- Whether all conversations can be exported at once as one archive.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
