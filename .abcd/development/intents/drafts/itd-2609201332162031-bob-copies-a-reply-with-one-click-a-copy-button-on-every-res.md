---
id: itd-2609201332162031
slug: bob-copies-a-reply-with-one-click-a-copy-button-on-every-res
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

# Bob copies a reply with one click: a copy button on every response in Dessau Chat

## Press Release

Bob asks a model for a shell command, reads the reply, and moves his pointer
over it: a small copy button appears at the reply's edge. One click puts the
reply's text on the clipboard, the button says "Copied" for a moment, and Bob
pastes it into his terminal. He never has to select the text, never has to
find the right-click menu, and never copies the model's reasoning by
accident, because the button copies what the reply says and nothing else.
On the iPad the button is there without a hover, beside each reply, and a tap
does the same. The context menu keeps its Copy for whoever reaches for it
that way.

## Why This Matters

The chat client is a demonstrator whose replies are meant to be used, and a
reply is used by being taken somewhere else: a terminal, an editor, a
message. Today copying a reply means knowing that a right-click or a long
press offers it, which is the one gesture a person cannot see. A visible
button is the difference between a reply that can be read and a reply that
can be used, and it is the affordance every chat surface Bob already knows
puts on every response.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

Seeded by the filing session, unconfirmed; every bullet is a proposal for the
planning interview:

- Given a reply on screen, when Bob moves the pointer over it on the Mac,
  then a copy button appears at the reply's edge and disappears when the
  pointer leaves; on the iPad the button is visible beside every reply.
- Given a reply with reasoning shown above it, when Bob clicks the button,
  then the clipboard holds the reply's text and none of the reasoning, and
  the same text the context menu's Copy would give.
- Given Bob has clicked the button, when he looks at it, then it says
  "Copied" for a moment and then returns to its resting state, and VoiceOver
  announces the same.
- Given the context menu, when Bob right-clicks or long-presses a reply,
  then Copy is still offered there, unchanged.
- Given a reply still streaming, when the button is used, then what is
  copied is the text received so far, and the button is not disabled.

## Open Questions

- Placement: a hover-revealed button at the reply's edge, or a small toolbar
  under the reply that could later carry other actions.
- What is copied: the rendered text as shown, or the raw markdown the model
  wrote; whether a fenced code block copied alone keeps its fence.
- Confirmation: the button's own "Copied" state only, or a brief
  notification too, and how long it stays.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
