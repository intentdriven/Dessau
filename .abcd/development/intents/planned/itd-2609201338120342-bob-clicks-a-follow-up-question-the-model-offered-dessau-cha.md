---
id: itd-2609201338120342
slug: bob-clicks-a-follow-up-question-the-model-offered-dessau-cha
spec_id: spc-2609201443358645
kind: standalone
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

Bob turns on **Offer follow-up questions** in Dessau Chat's Settings and
chooses the Triangle style, the one that teaches. From then on, under each
reply, the model offers up to three short questions it thinks Bob might ask
next: "How does this behave with an empty list?", "Tell me more about
generics." Bob clicks one and it is sent as his next message, in his own
turn of the conversation, exactly as if he had typed it. The questions are
the model's, never the client's: the client asks for them through a function
tool the model answers with a call, so the reply's own text is never touched
and nothing in it is turned into a link by scanning. With the setting off,
or under the Square and Circle styles, nothing is asked of the model and
nothing is shown; a conversation reads exactly as it does today. The
questions stay with the reply they came with, so after a relaunch the newest
reply still offers them. On the iPad the questions sit under the reply in
the same place, and a tap does the same. The on-device model is asked the
same way.

## Why This Matters

A reply often ends where the interesting question begins, and a small local
model is good at naming that question when asked. Offering it as something
to click keeps the conversation moving without Bob composing a follow-up he
half knows the shape of. The decision that every link in a response comes
from the model (2026-09-20, the maintainer, withdrawing an automatic
key-term scan) is what makes this the honest shape: the client asks, the
model offers, Bob chooses. Triangle is the style that asks what Bob already
knows, and follow-ups are its tool; Square's rule of no questions back
stays whole.

## Mechanism

Confirmed by the maintainer at the 2026-09-20 interview: we expect a small
local model to name a good next question when asked through a function tool
because proposing questions is instruction-following of the kind these
models do reliably, and because a tool call keeps the reply's text
untouched. What would show this wrong: models that emit no tool call for
it, or questions so generic that nobody clicks them.

## Scope Conditions

Confirmed by the maintainer at the 2026-09-20 interview:

- The chat window only, against served models and the on-device model; the <!-- cond: cond-2609201443351335 -->
  Discord bridge gets nothing.
- Only under the Triangle style; Square and Circle send no tool and show <!-- cond: cond-2609201443353681 -->
  nothing.
- The tool rides the request the way the answer styles decided (inside the <!-- cond: cond-2609201443359849 -->
  request, never a system role), and a model that emits no tool call simply
  offers nothing.
- Up to three questions, kept with the reply they came with, offered under <!-- cond: cond-2609201443356413 -->
  the newest reply only.
- Off by default in Settings. <!-- cond: cond-2609201443355340 -->

## Acceptance Criteria

Confirmed by the maintainer at the 2026-09-20 interview, every bullet walked
and accepted:

- Given the setting is off, when Bob sends a message, then the request is
  byte-identical to today's and no follow-up questions are shown.
- Given the setting is on and Triangle is chosen, when Bob sends a message,
  then the request carries a follow-ups function tool, the model's tool call
  yields up to three questions, and the reply's own text is untouched.
- Given a reply with follow-ups, when Bob looks under it, then up to three
  short questions are shown as buttons, each naming its text to VoiceOver,
  and none of the reply's own words has become a link.
- Given a follow-up button, when Bob clicks or taps it, then that question
  is sent as his next message, appears in the transcript as his own turn,
  and the buttons under the previous reply are no longer offered.
- Given the model returned no tool call, or one the client cannot read, when
  the reply is shown, then it is shown as any reply is, with nothing under
  it and no error.
- Given a conversation is stored, when Bob relaunches, then the follow-ups
  under the newest reply are still offered and older ones are not.
- Given Square or Circle is chosen, when Bob sends a message, then no
  follow-ups tool is sent and nothing is shown.
- Given the on-device model, when the setting is on under Triangle, then it
  is asked the same way and its follow-ups are shown the same way.

## Open Questions

All four were put to the maintainer at the 2026-09-20 interview and are
decided: the wire shape is a function tool call the model emits; follow-ups
are offered only under Triangle; they are kept with the reply and shown
under the newest reply only; the chat window only, served models and
on-device, never the bridge.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: a reply often ends where the interesting question begins, and a model asked through a tool can name it; wrong if nobody clicks the offered questions, or if the served models cannot emit the call
