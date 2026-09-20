---
id: itd-2609201332162031
slug: bob-copies-a-reply-with-one-click-a-copy-button-on-every-res
spec_id: spc-2609201442060136
kind: standalone
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
What lands on the clipboard is the reply as Bob reads it, plain text; a
switch in Settings makes it the raw markdown instead, for pasting into an
editor with its fences and headings intact. On the iPad the button is there
without a hover, beside each reply, and a tap does the same. The context
menu keeps its Copy for whoever reaches for it that way.

## Why This Matters

The chat client is a demonstrator whose replies are meant to be used, and a
reply is used by being taken somewhere else: a terminal, an editor, a
message. Today copying a reply means knowing that a right-click or a long
press offers it, which is the one gesture a person cannot see. A visible
button is the difference between a reply that can be read and a reply that
can be used, and it is the affordance every chat surface Bob already knows
puts on every response.

## Mechanism

Confirmed by the maintainer at the 2026-09-20 interview: we expect a
hover-revealed copy button to be used far more than the context-menu Copy
because a visible affordance is discovered and a hidden gesture is not.
What would show this wrong: testers still reaching for the context menu, or
never noticing the button.

## Scope Conditions

Confirmed by the maintainer at the 2026-09-20 interview:

- The chat client only, on the Mac and the iPad; a three-surfaces exception <!-- cond: cond-2609201442065935 -->
  on the precedent of the answer styles (itd-2609200850330402): client-only
  functionality with no server equivalent is not a gap.
- The copied text is the reply's own text, never the reasoning shown above <!-- cond: cond-2609201442060038 -->
  it.
- Rendered plain text or raw markdown is a client setting, plain text by <!-- cond: cond-2609201442060514 -->
  default.
- The context menu's Copy stays exactly as it is. <!-- cond: cond-2609201442066378 -->

## Acceptance Criteria

Confirmed by the maintainer at the 2026-09-20 interview, every bullet walked
and accepted:

- Given a reply on screen, when Bob moves the pointer over it on the Mac,
  then a copy button appears at the reply's edge and disappears when the
  pointer leaves; on the iPad the button is visible beside every reply.
- Given a reply with reasoning shown above it, when Bob clicks the button,
  then the clipboard holds the reply's text, plain by default or the raw
  markdown when the Settings switch says so, and none of the reasoning.
- Given Bob has clicked the button, when he looks at it, then it says
  "Copied" for a moment and then returns to its resting state, and VoiceOver
  announces the same.
- Given the context menu, when Bob right-clicks or long-presses a reply,
  then Copy is still offered there, unchanged.
- Given a reply still streaming, when the button is used, then what is
  copied is the text received so far, and the button is not disabled.

## Open Questions

All three were put to the maintainer at the 2026-09-20 interview and are
decided: the button is hover-revealed at the reply's edge (always visible on
the iPad); it copies the rendered plain text by default, with a Settings
switch for raw markdown; the confirmation is the button's own "Copied"
label, nothing else.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: replies are meant to be taken elsewhere, and a visible copy is the smallest thing that makes a reply usable rather than readable; wrong if testers keep selecting text by hand after it ships
