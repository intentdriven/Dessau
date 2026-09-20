---
id: spc-2609201443358645
slug: bob-clicks-a-follow-up-question-the-model-offered-dessau-cha
intent: itd-2609201338120342
origin: researcher-authored
production_mode: hand-written
---

# Follow-up questions the model offers, through a function tool, under Triangle

## Summary

This spec delivers itd-2609201338120342 in the chat client only: with
**Offer follow-up questions** on in Settings and the Triangle answer style
chosen, each request carries one function tool, and the questions the model
returns in its tool call are shown as buttons under the newest reply. A click
sends the question as Bob's own next turn. With the setting off, or under
Square or Circle, the request is what it is today and nothing is shown.
Nothing changes on the server, the control panel or `config.json`; no system
role is sent, and no word of a reply is scanned or turned into a link.
Impact: additive.

## Scope

In: `client/DessauChat/` — a new `FollowUps.swift`, `Backends.swift`,
`DessauChat.swift` (the `Message` type, `MessageRow`, the Settings pane) —
`internal/archtest/chat_client_followups_test.go`, and `client/README.md`.
Out: the server and the gateway (cond-2609201443359849 keeps the tool inside
the request, which the gateway already relays untouched), the Discord bridge
(cond-2609201443351335), and the answer styles themselves, which arrive with
spc-2609200945518522 and are a dependency of this spec rather than a subject
of it.

## Approach

**The style gate.** This spec is written against `Styles.swift` as
spc-2609200945518522 declares it: `enum AnswerStyle { case square, circle,
triangle }`, the style in force held on `Conversation.style` and stamped on
each `Message.style`. The gate is `offerFollowUps && style == .triangle`,
evaluated once where a reply is requested and passed into the backend, so
there is one reading of it per request (cond-2609201443353681). Square's rule
of no questions back is untouched, because under Square no tool is declared
at all.

**One declaration of the tool.** A new file
`client/DessauChat/FollowUps.swift` declares `enum FollowUps` with:

- `static let toolName = "offer_follow_ups"`.
- `static let tool: [String: Any]`, the OpenAI function-tool shape: a
  function named `offer_follow_ups`, described as offering up to three short
  questions the person might ask next, with parameters `{"type": "object",
  "properties": {"questions": {"type": "array", "items": {"type": "string"},
  "maxItems": 3}}, "required": ["questions"]}`.
- `static func parse(_ arguments: String) -> [String]`, which decodes the
  arguments JSON, takes `questions`, trims each, drops the empty ones, cuts
  anything past a length bound, and returns at most three. Anything it cannot
  read — malformed JSON, a different shape, a different tool name — returns
  an empty array, which is how "a tool call the client cannot read" becomes
  "nothing under the reply" with no error path (criterion 5).

**The served wire.** `ServerBackend.reply` adds one key to the body it
already builds — `"tools": [FollowUps.tool]` — and only inside the gate. The
messages array is untouched: it is still the history mapped to `user` and
`assistant`, with the style's text on the newest user turn as
spc-2609200945518522 composes it, and no `system` role is ever sent
(cond-2609201443359849). With the setting off the body is the dictionary
built today, key for key.

Reading the answer: `StreamChunk.Choice.Delta` gains an optional `tool_calls`
array of `{index, function: {name, arguments}}`. The stream loop accumulates
each index's `arguments` fragments into a buffer under the same byte caps the
loop already applies to a line and to the whole response, and on the terminal
event — `finish_reason == "tool_calls"`, or the end of the stream, whichever
comes first — hands the buffer for the call named `offer_follow_ups` to
`FollowUps.parse`. The reply's own `content` deltas are appended exactly as
today, so a model that emits both text and a tool call has its text shown
untouched (criterion 2). `ReplyEvent` gains `case followUps([String])`,
delivered once, after the last text chunk.

**The on-device wire.** `BuiltInBackend` is asked the same way, through the
framework's own tool seam rather than a hand-rolled prompt: `FollowUps.swift`
declares `struct FollowUpsTool: Tool` with the same name, the same
description and a `@Generable` arguments type holding `questions: [String]`,
and `BuiltInBackend` gives it to the session only inside the gate. Its `call`
hands the questions to the same sink the served path feeds and returns an
empty output, so nothing the tool produces re-enters the transcript and the
instructions entry keeps the budget it already counts. A model that never
calls it yields nothing, which is the same no-op as the served path.

**Storage and where they are shown.** `Message` gains `var followUps:
[String]? = nil` — optional for the reason its `model` field is optional:
Codable's synthesised decoder falls back for a missing key only on an
optional property, so a `conversations.json` written before this change
loads unchanged, and pre-1.0 means no migration code. The questions are
written onto the assistant message as the `followUps` event arrives, so they
travel with the reply they came with (cond-2609201443356413). `MessageRow`
draws them only when the message is the conversation's last: a `Bool` passed
in by the transcript list, not read from global state, so a row cannot offer
buttons for a reply that is no longer the newest. Older replies keep their
stored questions and show none, which is what makes the relaunch criterion
and the "previous buttons are no longer offered" criterion the same rule.

**The buttons.** Under the bubble, a wrapping row of bordered `Button`s, each
with the question as its title and `accessibilityLabel` (criterion 3), each
carrying no styling of its own so
`TestChatClientCarriesNoStylingOfItsOwn` gains no exception. A click calls
the same `send()` path a typed message takes, with the question as the text:
it appears in the transcript as Bob's own user turn, and the new assistant
message that follows becomes the newest, which retires the previous row of
buttons by the rule above (criterion 4).

**Settings.** `@AppStorage("offerFollowUps") private var offerFollowUps =
false` — off by default (cond-2609201443355340) — a `Toggle` in the Settings
pane reading "Offer follow-up questions", with one line saying the questions
are the model's own, that they are asked for only under the Triangle style,
and that a model which does not answer with a tool call simply offers none.
`client/README.md` gains one line; the client's docs section on the answer
styles gains a sentence saying the same.

## Acceptance Criteria, and what holds each

The client has no XCUITest target (itd-2609170718438919), so the wire shape
and the gating are held by architecture tests over the Swift source, and what
appears on screen by recorded hand checks.

- **Setting off: the request is byte-identical to today's and nothing is
  shown**: architecture test —
  `TestFollowUpsRideOnlyTheGuardedRequest` in
  `internal/archtest/chat_client_followups_test.go` asserts that `"tools"`
  appears in `Backends.swift` only inside the branch guarded by the
  follow-ups gate, and that `FollowUps.swift` and `Backends.swift` contain no
  `"system"` role literal. Byte identity itself is a recorded hand check
  against the server's request log with the setting off and on.
- **Setting on under Triangle: the tool rides, up to three questions come
  back, the reply's text is untouched**: architecture test — the same file
  asserts the tool declaration names `offer_follow_ups` with `maxItems` 3,
  that `parse` caps its result at three, and that the text path in the stream
  loop is not conditioned on the tool call. A recorded hand check against a
  served model captures the request body, the tool call and the reply.
- **Up to three buttons under the reply, each named to VoiceOver, no word of
  the reply turned into a link**: architecture test — the test asserts
  `MessageRow` draws the buttons from `message.followUps` and that no file
  under `client/DessauChat/` scans reply text for key terms (no new matcher
  beside `Effects.swift`'s, which matches for animation and links nothing). A
  recorded hand check with VoiceOver on.
- **A click sends the question as Bob's turn and retires the previous
  buttons**: architecture test — the test asserts the button's action calls
  the same send path a typed message takes, and that the row draws the
  buttons only when it is told it is the last message. The behaviour on
  screen: a recorded hand check over two turns.
- **No tool call, or one that cannot be read: the reply is shown as any reply
  is, with no error**: architecture test — the test asserts `parse` returns
  an empty array on every failure and throws nothing, and that no error path
  in the stream loop is reached from a tool-call branch. A recorded hand
  check against a model the server records as unable to call tools.
- **Relaunch keeps the newest reply's follow-ups and offers no older ones**:
  architecture test — the test asserts `Message.followUps` is an optional
  stored property on the Codable type. The relaunch itself: a recorded hand
  check, including one against a `conversations.json` written before this
  change.
- **Square or Circle: no tool, nothing shown**: architecture test — the gate
  named above is the only place the tool is added, and the test asserts the
  gate names `.triangle`.
- **The on-device model is asked the same way**: architecture test — the test
  asserts `FollowUpsTool` conforms to `Tool`, carries the same tool name, and
  is attached to the session only inside the gate; and that
  `BuiltInBackend` still names no `URLSession`. That its questions show the
  same way: a recorded hand check on the Mac's own model.

## Departures

None at the time of writing.
