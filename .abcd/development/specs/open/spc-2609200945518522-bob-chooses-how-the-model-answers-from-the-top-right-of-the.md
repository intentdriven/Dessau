---
id: spc-2609200945518522
slug: bob-chooses-how-the-model-answers-from-the-top-right-of-the
intent: itd-2609200850330402
origin: researcher-authored
production_mode: hand-written
---

# Answer styles: Square, Circle and Triangle in the chat client

## Summary

This spec delivers itd-2609200850330402 in the chat client only: three
answer styles chosen from a segmented control in the window's top right,
each a fixed client-authored instruction, applied from the next reply,
stamped on every reply, remembered per conversation, with a default in
Settings. Nothing changes on the server, the control panel or `config.json`;
the wire carries no new role and no new field. Impact: additive.

## Scope

In: `client/DessauChat/` and `internal/archtest/`, the client README and
the user-facing docs page for the chat client. Out: the server, the bridge,
the models list, the pairing and key handling, the transcript recording
draft. The Discord bridge keeps `/model` as its only per-channel choice.

## Approach

**One declaration.** A new file `client/DessauChat/Styles.swift` declares
`enum AnswerStyle: String, CaseIterable, Codable { case square, circle,
triangle }` with, per case: a display name, one line of what it does, the
instruction text, the mark's colour, and a SwiftUI shape for the glyph. The
three texts are the client's own words and are never editable:

- Square: "Answer directly: the shortest complete answer, code before
  explanation when code is asked for, no preamble, no closing offer, and no
  questions back."
- Circle: "Answer thoroughly: build the answer up, explain the trade-offs,
  and keep the thread of the conversation."
- Triangle: "Teach: when it would change the answer, first ask what the
  person already knows; prefer a guiding question where it helps; explain
  the why before the how."

Length lives in these sentences. `max_tokens` stays at its current value as
a safety ceiling and no style touches it, so the operator's per-model
default on the server is never overridden (scope condition 4).

**The control.** A segmented `Picker` in the toolbar's trailing position,
each segment an `AnswerStyle.glyph` view: the chosen style's shape filled in
its colour, the others stroked in the secondary colour. The glyph is drawn
in `Styles.swift` with SwiftUI shapes, which needs `.overlay(` and a filled
shape, so `Styles.swift` becomes the fourth named exception in
`TestChatClientCarriesNoStylingOfItsOwn`, with its reason: the marks are the
product's own forms and no system control draws them (decision 5). On the
Mac each segment carries `.help(name · line)`; on the iPad a long press on a
segment presents a popover with the same name and line (decision 5, iPad);
every segment carries `accessibilityLabel(name)`, and a change posts an
accessibility announcement naming the new style.

**Per conversation, per reply, and the default.** `Conversation` gains
`var style: AnswerStyle? = nil` and `Message` gains `var style: AnswerStyle?
= nil`, both optional so a `conversations.json` written before this change
decodes unchanged, nil reading as Square (scope condition 5, criteria 8 and
9). A new chat takes `AppStorage("defaultStyle")`, default `square`, which
Settings exposes as a pop-up under Models (criterion 8). Picking a style in
a chat sets `conversation.style` and sends nothing (criterion 2). When a
reply is requested, the style in force is written to the assistant message
as it is created, so the mark on a reply is the one it was generated under
however the control is set later (criterion 6).

**The wire, server side (decision 1).** `ServerBackend.reply` keeps mapping
history to `user` and `assistant` roles and adds nothing else. The mapping is
moved into a pure function `ServerBackend.messages(for history: [Message],
style: AnswerStyle) -> [[String: String]]` that returns the history as
stored except that the newest user turn's content is the style's text, a
blank line, then Bob's text. Earlier user turns travel as Bob wrote them,
because the style is restated beside the newest turn only (decision 2), so
the cached prefix survives a switch and the stored transcript never carries
a style text. No `system` role is ever sent, so a chat template without one
answers as normal (criteria 3 and 4).

**The wire, on-device (decision 1, criterion 5).** `BuiltInBackend.trimmed`
keeps building one instructions entry per turn; its content becomes
`BuiltInBackend.instructions` followed by a blank line and the style's text,
so the client's language and markdown rules are unchanged and the style is
appended, never substituted (scope condition 3). The budget already counts
the instructions entry, so a longer entry is paid for before the history is
trimmed.

**The reply label (criterion 6).** The speaker string that
`MessageView.speaker` builds gains the style's name in front of the model's
name, and the row draws the glyph before the text; the accessibility label
already reads the speaker, so VoiceOver hears "Square · Qwen… said: …".

**The docs.** The client's docs page gains a section on the three styles
that says plainly that the on-device model follows Triangle less well than
a served model (decision 4), and the client README's feature list gains one
line. Present tense, British English.

## How each criterion is held

1. Control, colour, names, default: the control is a standard segmented
   picker; the glyph exception is named in the architecture test; the
   default is a hand check recorded in the shipping decision line.
2. The Square test: a recorded hand check on a served model and on the
   on-device model, the prompts and replies kept in the shipping line.
3. Request body: `messages(for:style:)` is the only builder of the request's
   messages, held by an architecture test that the body is built through it
   and that the file contains no `"system"` role string; the client has no
   test target of its own (itd-2609170718438919), so the exact prefix is
   also a recorded hand check against the server's request log.
4. No system role: follows from 3; a hand check on a template-strict model
   is recorded.
5. On-device composition: the architecture test holds that
   `BuiltInBackend.instructions` is still declared and still names markdown
   and code blocks; the fenced-code check is a hand check.
6. Mark on each reply: the style is written at message creation; a hand
   check with two styles in one chat.
7. Relaunch: a hand check.
8. Settings default and existing chats: a hand check.
9. Old conversations file: a hand check with the file the maintainer's
   client holds before the change.
10. Architecture tests: `TestChatClientDeclaresThreeAnswerStyles` reads
    `Styles.swift` and asserts exactly the three cases, three non-empty
    texts, no banned name in any text, and the fourth exception's reason;
    the existing instructions test is extended as in 5.
11. iPad: the simulator, a hand check of the long press, recorded to the
    same terms as itd-2609180943290800.

## What is deliberately not built

- No transcript row at the switch point: the mock draws one, the intent
  promises only the mark on each reply.
- No per-style `max_tokens`, no style on the bridge, no server field.
- No migration of `conversations.json`; the two new fields are optional.

## Falsifiers, from the Mechanism claim

Square and Circle replies indistinguishable in length on the held-out
prompts; Triangle not held across turns on the on-device model beyond the
caveat the docs state; a reply's mark lost after relaunch. Any of these
reopens the intent rather than being patched.
