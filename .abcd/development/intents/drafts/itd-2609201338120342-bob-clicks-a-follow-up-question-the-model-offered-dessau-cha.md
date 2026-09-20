---
id: itd-2609201338120342
slug: bob-clicks-a-follow-up-question-the-model-offered-dessau-cha
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609151701196720, itd-2609200850330402]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# Bob clicks a follow-up question the model offered: Dessau Chat asks the model for optional follow-ups when that setting is on

## Press Release

Bob turns on **Offer follow-up questions** in Dessau Chat's Settings. From
then on, under each reply, the model offers a few short questions it thinks
Bob might ask next: "How does this behave with an empty list?", "Tell me more
about generics." Bob clicks one and it is sent as his next message, in his
own turn of the conversation, exactly as if he had typed it. The questions
are the model's, never the client's: nothing in the reply is turned into a
link by scanning it, so every link Bob sees came from the model that wrote
the reply. With the setting off, nothing is asked of the model and nothing
is shown; a conversation reads exactly as it does today. On the iPad the
questions sit under the reply in the same place, and a tap does the same.

## Why This Matters

A reply often ends where the interesting question begins, and a small local
model is good at naming that question when asked. Offering it as something
to click keeps the conversation moving without Bob composing a follow-up he
half knows the shape of. The decision that every link in a response comes
from the model (2026-09-20, the maintainer, withdrawing an automatic
key-term scan) is what makes this the honest shape: the client asks, the
model offers, Bob chooses.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

Seeded by the filing session, unconfirmed; every bullet is a proposal for the
planning interview:

- Given the setting is off, when Bob sends a message, then the request is
  byte-identical to today's and no follow-up questions are shown.
- Given the setting is on, when Bob sends a message, then the request asks
  the model for a few optional follow-up questions the way the answer styles
  ride the wire (inside the newest user turn, never a system role), and the
  reply's own text is rendered without the block that carries them.
- Given a reply with follow-ups, when Bob looks under it, then up to three
  short questions are shown as buttons, each naming its text to VoiceOver,
  and none of the reply's own words has become a link.
- Given a follow-up button, when Bob clicks or taps it, then that question
  is sent as his next message, appears in the transcript as his own turn,
  and the buttons under the previous reply are no longer offered.
- Given the model returned no follow-ups or a block the client cannot read,
  when the reply is shown, then it is shown as any reply is, with nothing
  under it and no error.
- Given a conversation is stored, when Bob relaunches, then the follow-ups
  under the newest reply are still offered and older ones are not.

## Open Questions

- Wire shape: how the model marks its follow-ups so the client can strip and
  render them (a trailing delimited block, a fixed heading), and how a model
  that ignores the instruction is handled.
- The answer styles: Square says "no questions back"; whether follow-ups are
  offered under every style, or Square offers none.
- Storage: whether the offered questions are kept in the conversation store
  or only shown for the newest reply of the session.
- The Discord bridge and the on-device model: whether either gets follow-ups,
  or only the chat window against a server.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
