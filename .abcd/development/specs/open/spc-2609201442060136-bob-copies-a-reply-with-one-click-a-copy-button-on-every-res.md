---
id: spc-2609201442060136
slug: bob-copies-a-reply-with-one-click-a-copy-button-on-every-res
intent: itd-2609201332162031
origin: researcher-authored
production_mode: hand-written
---

# A copy button on every reply in Dessau Chat

## Summary

This spec delivers itd-2609201332162031 in the chat client only: a copy
button at each reply's trailing edge, revealed on hover on the Mac and always
visible on the iPad, which puts the reply's own text on the clipboard — the
rendered plain text by default, the raw markdown when a Settings switch says
so — says "Copied" for a moment, and never copies the reasoning shown above
the reply. The context menu's Copy is untouched. Nothing changes on the
server, the control panel or `config.json`, and no request shape moves.
Impact: additive.

## Scope

In: `client/DessauChat/DessauChat.swift` (the `MessageRow` view and the
Settings pane), a new `client/DessauChat/CopyReply.swift`, a new
`internal/archtest/chat_client_copy_test.go`, and one line in
`client/README.md`. Out: the server, the gateway, the control panel, the
Discord bridge, the transcript store, and the user's turns — the button is
offered on replies only, because a person's own text is already in front of
them.

Client-only functionality with no server equivalent is not a three-surfaces
gap here: cond-2609201442065935 carries the answer-styles precedent
(itd-2609200850330402) forward, and this spec relies on it rather than
re-arguing it.

## Approach

**One place decides what is copied.** A new file
`client/DessauChat/CopyReply.swift` declares `enum CopyPayload` with one pure
function, `static func text(for message: Message, rawMarkdown: Bool) ->
String`. With `rawMarkdown` true it returns `message.text` exactly as the
model wrote it, fences and headings intact. With it false it returns the
reply as Bob reads it: `MarkdownBlocks.parse(message.text)` (the same parser
`MessageRow` draws from, so what is copied and what is on screen cannot
drift) rendered back to characters — a heading's words without its hashes, a
list item behind `- ` or `1. `, a fenced block's lines without the fence, a
`.plain` block (a table) exactly as the model wrote it — blocks joined by a
blank line. The function reads `message.text` and nothing else; it never
names `message.reasoning`, which is how cond-2609201442060038 is held by
construction rather than by care.

**The button is a standard control, placed rather than overlaid.** The same
file declares `struct CopyReplyButton: View`: a bordered-less `Button` with
`Image(systemName: "doc.on.doc")`, `accessibilityLabel("Copy reply")`, and no
styling of its own. `MessageRow` puts it in the existing `HStack` beside the
bubble's trailing edge, not on top of it, because `.overlay(` is one of the
constructs `TestChatClientCarriesNoStylingOfItsOwn` refuses outside its three
named exceptions, and this intent does not earn a fourth. Visibility is
`.opacity`, which that test does not touch: on the Mac the row carries
`@State private var hovering = false` set by `.onHover`, and the button's
opacity is `hovering ? 1 : 0`; under `#if !os(macOS)` the opacity is fixed at
1, so the iPad shows it without a hover. The button keeps its space in the
layout whether it is visible or not, so a reply does not reflow under the
pointer.

**"Copied", and nothing else.** `@State private var copied = false`; a click
writes the clipboard, sets `copied`, posts
`AccessibilityNotification.Announcement("Copied")`, and a `Task` that sleeps
one and a half seconds clears it — cancelled and restarted on a second click,
so two clicks do not leave the label stuck. While `copied` is true the label
is the word "Copied" beside the glyph. There is no toast, no sheet and no
sound: the button's own label is the whole of the confirmation, which is the
maintainer's decision of 2026-09-20.

**The two pasteboards.** The write is the pair `MessageRow` already carries
in its context menu: `NSPasteboard.general.clearContents()` then
`setString(_:forType: .string)` on the Mac, and on iOS
`UIPasteboard.general.setItems(_:options: [.localOnly: true])` so a reply
copied on the iPad does not travel to Bob's other devices over the Universal
Clipboard. That pair moves into `CopyPayload.write(_:)` so there is one copy
of the `localOnly` reasoning rather than two.

**The Settings switch.** `@AppStorage("copyRawMarkdown") private var
copyRawMarkdown = false` in the Settings pane, a `Toggle` reading "Copy
replies as raw markdown" with one line under it saying that plain text is
what a message or a terminal wants and raw markdown is what an editor wants.
Off by default (cond-2609201442060514). It is read at the moment of the
click, so a change applies to the next copy without any reply being redrawn.

**The context menu stays.** cond-2609201442066378 says exactly as it is, and
this spec takes that literally: the existing `Button("Copy")` keeps copying
`message.text` raw, and is not rewired through `CopyPayload`. The two routes
can therefore disagree when the switch is off — the menu gives the markdown,
the button gives the plain text. That is the condition read as written; if
the maintainer would rather the menu followed the switch, it is a one-line
change and a new criterion, not a silent improvement here.

**Streaming.** The button is never disabled and is not conditioned on the
stream being finished. The payload is computed from `message.text` at the
moment of the click, so what lands on the clipboard is what had arrived by
then, which is what the criterion asks for. The bubble already re-renders per
chunk; the button costs one more view in that row and no extra parse, because
the parse is the one `MessageRow` already throttles.

## Acceptance Criteria, and what holds each

The client has no XCUITest target (itd-2609170718438919), so behaviour on
screen is held by a recorded hand check and structure is held by an
architecture test over the Swift source.

- **A hover-revealed button on the Mac, always visible on the iPad**:
  architecture test — `TestChatClientRepliesCarryACopyButton` in
  `internal/archtest/chat_client_copy_test.go` asserts `MessageRow`'s
  assistant branch names `CopyReplyButton`, that the row declares `.onHover`
  and drives `.opacity`, and that the `#if !os(macOS)` arm fixes the opacity
  at 1. Appearance and disappearance under the pointer: a recorded hand check
  on the Mac, and one on the iPad simulator, to the terms of
  itd-2609180943290800.
- **The clipboard holds the reply's text and none of the reasoning, plain or
  raw by the switch**: architecture test —
  `TestTheCopyPayloadNeverNamesTheReasoning` asserts `CopyReply.swift`
  declares `CopyPayload.text(for:rawMarkdown:)`, that the file does not
  contain the string `reasoning`, and that the plain branch goes through
  `MarkdownBlocks.parse`. A recorded hand check copies a reply from a
  thinking model with its Thoughts expanded, in both switch positions, and
  pastes the result.
- **"Copied" for a moment, and VoiceOver says the same**: architecture test —
  the same file asserts `CopyReplyButton` declares a `copied` state, the
  literal `"Copied"`, and an `AccessibilityNotification.Announcement`. The
  timing and the VoiceOver utterance: a recorded hand check with VoiceOver
  on.
- **The context menu's Copy is unchanged**: architecture test — the copy test
  asserts the `.contextMenu` block in `MessageRow` still declares
  `Button("Copy")` and still writes `message.text`, and that it does not call
  `CopyPayload`.
- **A streaming reply copies what has arrived, and the button is not
  disabled**: architecture test — the copy test asserts the button carries no
  `.disabled(` and is not guarded on the model's streaming state. That what
  lands is the partial text is a recorded hand check against a long reply.
- **Nothing is sent anywhere**: architecture test — the copy test asserts
  `CopyReply.swift` names neither `URLSession` nor `URLRequest`, the rule
  `TestChatClientBuiltInBackendTouchesNoNetwork` already applies to the
  answerer seam.
- **No styling of its own**: held by the existing
  `TestChatClientCarriesNoStylingOfItsOwn`, which gains no fourth exception —
  a green run with `CopyReply.swift` present is the evidence.

Beside the criteria, the shipping decision line records the two hand-check
sessions (Mac and iPad) with what was copied and what was pasted, and
`client/README.md` gains one line in its feature list.

## Departures

None at the time of writing.
