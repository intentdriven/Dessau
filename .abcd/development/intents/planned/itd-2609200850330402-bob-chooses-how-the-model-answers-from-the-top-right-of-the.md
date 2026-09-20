---
id: itd-2609200850330402
slug: bob-chooses-how-the-model-answers-from-the-top-right-of-the
spec_id: spc-2609200945518522
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609200827202340, itd-2609200829199959]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# Bob chooses how the model answers, from the top right of the chat window

## Press Release

Bob opens a chat in the Dessau chat client and, in the top right of the
window, finds three small marks: a square, a circle and a triangle. They are
the three forms of the product's own mark, and each one is a way of
answering. Square is direct, and the one a new chat starts on: short answers, code
first, no preamble and no questions back. Circle is thorough: it
builds the answer up, explains the trade-offs and keeps the thread. Triangle
is teaching: it asks what Bob already knows, answers with a question where
that helps, and explains the why before the how. Bob picks one and his next
reply comes in that style; the replies already on screen stay as they were,
and each reply's label carries the mark it was answered under, so a
conversation shows where the stance changed. A style is a way of answering,
never a person: no style has a name, and the model never claims to be
anyone. The chat remembers its style, Settings holds the default for a new
chat, and the iPad client shows the same three in the same place.

## Why This Matters

Today the only way to get a short answer is to type "be brief" into every
message, and the only way to be taught rather than told is to argue with the
model about it turn by turn. Both are habits, not settings, and a person who
wants the model to ask what they already know has no way to say so once. The
naming decision (adr-2609200729102059, decision 5) settled that a mode
carries a shape from the mark and never a person's name; this intent is the
capability that rule was written for, and the first thing to make the mark
do work inside the product.

## Mechanism

Confirmed by the maintainer at the 2026-09-20 interview: we expect
three named styles to change how a small local model answers because length
and whether to ask a question back are instruction-following behaviours that
instruction-tuned MLX models and the on-device model obey reliably, where
voice and character are not, which is why there is no playful style; and
because the client already rebuilds the on-device instructions entry on every
turn, so a style change takes effect on the next reply without discarding the
conversation. What would show this wrong: a served model whose Square and
Circle replies are indistinguishable in length on a held-out set of prompts;
the on-device model failing to hold the Triangle stance across turns; or a
style whose mark on a reply label cannot be reconstructed after relaunch.

## Scope Conditions

Confirmed by the maintainer at the 2026-09-20 interview, all six:

- The chat client only, on the Mac and the iPad; the server, the control <!-- cond: cond-2609200945519359 -->
  panel and `config.json` gain nothing. A three-surfaces exception on the
  precedent of itd-2609170718438919 (`cond-2609170842086229`): client-only
  functionality with no server equivalent is not a gap. The Discord bridge
  (itd-2609180959397172) does not gain styles; `/model` stays its only
  per-channel choice.
- Three styles and no more: Square, Circle, Triangle, from <!-- cond: cond-2609200945512268 -->
  adr-2609200729102059 decision 5, drawn with the mark's three forms
  (itd-2609200827202340). No style carries a person's name and the assistant
  never claims to be a person.
- A style is a fixed, client-authored text that Bob cannot edit, so the <!-- cond: cond-2609200945517188 -->
  on-device rule that instructions win over a prompt holds; it is appended to
  the client's own instructions (language, markdown), never a replacement for
  them.
- Length is part of the style's text, never a token cap: the client's <!-- cond: cond-2609200945514804 -->
  existing `max_tokens` stays a safety ceiling and no style overrides the
  per-model default Alice set on the server.
- Per-conversation and per-reply state live in the conversation store as <!-- cond: cond-2609200945513896 -->
  optional fields, nil meaning Square, so a conversation written before this
  change loads and reads as Square with no migration (pre-1.0, and in any
  case none is needed).
- The iPad claim holds to the same terms as itd-2609180943290800: the <!-- cond: cond-2609200945516495 -->
  maintainer's own device on a seven-day profile, the on-device path in the
  simulator.

## Acceptance Criteria

Confirmed by the maintainer at the 2026-09-20 interview, every bullet walked and accepted:

- Given the chat window is open, when Bob looks at the top right of the
  toolbar, then three marks, square, circle and triangle, sit as one
  segmented control, the chosen one filled in its colour from the mark, each
  naming its style on hover, on long press on the iPad, and to VoiceOver, and
  Square is chosen in a new chat unless Settings says otherwise.
- Given Circle is chosen and Bob has asked "how do I reverse a list in
  Python", when he picks Square and asks the same again, then the Square
  reply opens with a code block and contains no sentence ending in a question
  mark, the earlier reply on screen is unchanged, and no request was sent at
  the moment he picked.
- Given a server backend, when the client builds a request under a style,
  then the style's text is prepended to the newest user turn and nothing
  else in the body changes; held by a test over the request body.
- Given a model whose chat template has no system role, when Bob sends under
  any style, then the reply arrives as normal, since no system role is used.
- Given the on-device backend, when a style is chosen, then its text is
  appended to the client's own instructions, the composed instructions are
  rebuilt on every turn, and a Square reply still renders a fenced code block
  with a language.
- Given a reply has arrived, when Bob reads its label, then the style's mark
  sits beside the model's name and its name is in the accessibility label,
  and the mark is the one in force when that reply was generated, not the
  one selected now.
- Given a conversation with replies in two styles, when Bob quits and
  relaunches, then each reply still carries the mark it was generated under
  and the conversation reopens on the style last chosen in it.
- Given Settings, when Bob sets the default style, then a new chat opens on
  it and an existing conversation keeps its own; a conversation stored
  before this change reads as Square.
- Given a conversations file written before this change, when the client
  launches, then every conversation loads and each renders as Square.
- Given the client's source, when the architecture tests run, then exactly
  three styles are declared in one file, each with a non-empty text, the
  client's fixed instructions are still present and still name markdown, and
  no style text contains a person's name from the banned list.
- Given the iPad client in the simulator, when the same window is opened,
  then the same three marks sit in the same place and a long press on a mark
  shows its name and one line of what it does, since the iPad has no hover.

## Open Questions

Decided at the 2026-09-20 interview (maintainer's answers, one line each in
DECISIONS.md):

1. **How the style travels to a server:** inside the newest user turn, the
   style's text prepended to Bob's message. Nothing new on the wire, every
   chat template, no gateway involvement. Declined: a system message, the
   client's first, and a per-model choice needing a models-list field.
2. **A mid-conversation switch:** the style is stamped per reply and restated
   beside the newest turn, so the model sees the change and the cached
   prefix survives; follows from keeping the per-reply mark in the release.
3. **The default:** Square everywhere. Declined: Circle everywhere, and a
   default that follows the backend.
4. **Triangle on the on-device model:** ship three and say so in the docs.
   Declined: hiding Triangle on this Mac, a check-first gate, dropping it.
5. **The control:** filled colour, with a fourth named file added to the
   client's styling exception. The iPad names each mark on long press. The
   model control at the composer collapses to a server glyph rather than a
   dot, so the dot never collides with the Circle mark (recorded on
   itd-2609200829199959).
6. **Impact:** additive, stamped at the maintainer's sign-off.

Nothing open: the eleven acceptance criteria were walked and accepted at
the same interview, and the plan step was run as the maintainer's sign-off.

Decided or provenance before the interview: the three names and the
no-person rule (adr-2609200729102059, decision 5, which this intent
refines); the marks are the forms of itd-2609200827202340; the top-right
placement follows the picker decision of 2026-09-20 (DECISIONS.md), on which
this draft builds. Mock reviewed 2026-09-20 on boards 7 and 8 of the picker
mock; the reply label carrying the mark and the switch point are drawn there.

Adversarial reviews 2026-09-20, design/feasibility and record-discipline:
applied above except two findings rejected: the seeded criterion that the
style is sent as a system message (superseded by decision 1), and the
suggestion to leave the picker's collapsed dot in place (decision 5 changed
it instead).

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: we expect the three forms to mean something inside the product once they do work as controls, which makes the logo a system rather than decoration; wrong if testers still read them as ornament and never touch the control.
