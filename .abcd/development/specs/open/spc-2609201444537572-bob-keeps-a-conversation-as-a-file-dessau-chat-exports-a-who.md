---
id: spc-2609201444537572
slug: bob-keeps-a-conversation-as-a-file-dessau-chat-exports-a-who
intent: itd-2609201355512965
origin: researcher-authored
production_mode: hand-written
---

# Exporting a conversation: Markdown or JSON, written by the client

## Summary

This spec delivers itd-2609201355512965 in the chat client only: **Export…**
in a conversation's own menu writes the whole conversation to a file the
person chooses — Markdown for reading, JSON in the client's own store shape
so it can be read back in — with the model's name, the time and the style
mark on every turn, and, under a **Detailed export** switch in Settings, the
reasoning, the token counts, the timings and the statistics the server
reports. **Export All…** lives in the app's menu bar and writes one folder
holding every conversation. Nothing is sent anywhere and the stored
conversation is untouched. Impact: additive.

## Scope

In: `client/DessauChat/` — a new `Export.swift`, `DessauChat.swift` (the
`Message` type, the conversation menu, the app's `Commands`, the Settings
pane), `Backends.swift` (the reply statistics the detailed export needs) —
`internal/archtest/chat_client_export_test.go`, and `client/README.md`. Out:
the server, the control panel, `config.json`, the Discord bridge, any sync or
upload path, and any change to how `conversations.json` is written — export
copies out of the store, it never moves or rewrites it
(cond-2609201444537168, cond-2609201444536551).

## Approach

**One renderer, two formats, no view code.** A new file
`client/DessauChat/Export.swift` declares `enum ExportFormat: String {
case markdown, json }` and `enum Export` with four pure functions:
`markdown(_ conversation: Conversation, detailed: Bool) -> String`,
`json(_ conversation: Conversation, detailed: Bool) throws -> Data`,
`filename(for: Conversation, _ format: ExportFormat) -> String`, and
`folderName(at: Date) -> String` for Export All. They take a value and return
bytes: no `View`, no `AppModel`, no file system, which is what makes the
hundred-turn criterion a matter of where they are called from rather than of
what they do.

**Markdown.** `# <title>`, then the conversation's creation date on its own
line, then one section per turn in order: `## You` for Bob's turns and
`## <model name> · <style mark> · <time>` for a reply, where the model name is
the last path component of `message.model` (the same shortening
`MessageRow.speaker` does), the style mark is `AnswerStyle`'s display name as
`Styles.swift` declares it, and the time is the turn's timestamp in the
person's locale. The body is the text as the model wrote it — raw markdown,
fences and all, because the file is markdown. With `detailed` true a reply
also carries a `### Thoughts` block quote holding `message.reasoning` and one
line of figures: prompt and completion tokens, time to first token, total
time, and the server's own state and queue time.

**JSON is the store shape.** `Export.json` encodes the `Conversation` value
with the same `JSONEncoder` configuration `conversations.json` is written
with, so an exported file is a document the client already knows how to
decode and import is `JSONDecoder().decode(Conversation.self, from:)` with a
fresh `id` and, on a title clash, a suffixed title (cond-2609201444537911).
With `detailed` false the encoder is handed a copy of the conversation whose
`reasoning` is blanked and whose statistics are dropped; every such property
is optional or defaulted, so a plain export still decodes as the same
conversation with less in it. There is no second schema and no version field:
pre-1.0 means no migration code, and the store shape is the contract.

**Where the figures come from.** `Message` gains `var stats: ReplyStats? =
nil`, a small Codable struct declared in `Export.swift`: prompt and
completion tokens, first-token milliseconds, total milliseconds, and the
server's `X-Dessau-State` and `X-Dessau-Queue-Time` header values.
`ServerBackend` fills it: the body gains `"stream_options": {"include_usage":
true}` — the key is always written when the object is present, because a
`stream_options` object without it raises inside mlx-lm (`internal/mlxtest`)
— the final usage event supplies the counts, the loop already knows when the
first chunk arrived, and the two headers are read off the response.
`BuiltInBackend` fills the timings and leaves the counts and headers nil,
which the renderer prints as absent rather than as zero. The field is
optional, so an older `conversations.json` decodes unchanged.

**The Mac's save panel.** The conversation's own menu — the one
`ConversationCard`'s context menu and the window's conversation menu share —
gains **Export…**. Under `#if os(macOS)` it drives SwiftUI's
`.fileExporter(isPresented:document:contentTypes:defaultFilename:)` with a
`ConversationDocument: FileDocument` whose `writableContentTypes` are
`[.markdown, .json]` and whose `fileWrapper(configuration:)` renders through
`Export` for the type the panel returns, so the format is chosen in the panel
rather than in a dialog of our own. `defaultFilename` is
`Export.filename(for:_:)`: the title, reduced to a file-safe name, with the
extension the type carries.

**The iPad's share sheet.** Under `#else` the same menu item renders the file
into the app's temporary directory and presents a `ShareLink` over that URL,
so it goes to Files, Mail or another app (criterion 2). The rendered bytes
are identical — the same three functions produce them — so the two platforms
cannot drift.

**Export All, in the menu bar only.** `DessauChat.swift` already declares a
`Commands` builder; it gains `CommandGroup(after: .saveItem) { Button("Export
All…") }`. The action opens a folder chooser, then writes one folder named
`Dessau Chat export <date>` holding one file per conversation in the format
the Settings default names, skipping nothing and overwriting nothing (a name
that exists gains a numeric suffix). No window carries a button, a toolbar
item or a menu entry for it (cond-2609201444533682), and an architecture test
holds that.

**Staying responsive.** Rendering and writing happen off the main actor: the
menu item captures the conversation value, hands it to a detached task, and
publishes only a completion or a failure back. Export All iterates the same
way, one conversation at a time, so a hundred-turn conversation or fifty
conversations cost memory proportional to one file rather than to the store
(criterion 7).

**Settings.** `@AppStorage("detailedExport") private var detailedExport =
false`, a `Toggle` reading "Detailed export" with one line saying it adds the
reasoning, the token counts, the timings and the server's statistics — and
that a detailed file therefore carries what a reader does not need
(cond-2609201444539322). `client/README.md` gains one line.

## Acceptance Criteria, and what holds each

The client has no XCUITest target (itd-2609170718438919), so the renderers
and the placement of the controls are held by architecture tests over the
Swift source, and the panels, the sheet and the round trip by recorded hand
checks.

- **Export from the conversation's menu on the Mac opens a save panel named
  after the conversation, offering Markdown or JSON, and the file holds every
  turn in order with who said it**: architecture test —
  `TestConversationExportRendersBothFormats` in
  `internal/archtest/chat_client_export_test.go` asserts `Export.swift`
  declares the two format cases and the four functions, that the markdown
  renderer walks `conversation.messages` in order and writes a heading per
  turn, and that `ConversationDocument.writableContentTypes` names
  `.markdown` and `.json`. The panel itself and its default name: a recorded
  hand check on the Mac.
- **The same on the iPad opens the share sheet with the same file**:
  architecture test — the test asserts the `#else` arm presents a `ShareLink`
  over a URL produced by the same `Export` functions. The sheet: a recorded
  hand check on the iPad simulator, to the terms of itd-2609180943290800.
- **The conversation is unchanged and no server is sent anything**:
  architecture test — the test asserts `Export.swift` names neither
  `URLSession`, `URLRequest` nor `Network`, and that no export path calls the
  store's save. A recorded hand check reopens the conversation after an
  export and confirms the server's request log is silent.
- **Model name, time and style mark per turn; detailed adds reasoning,
  counts, timings and the server's statistics**: architecture test — the test
  asserts the markdown renderer names `message.model`, the style's display
  name and the timestamp unconditionally, and that `reasoning` and
  `ReplyStats` are named only inside a `detailed` branch. A recorded hand
  check compares a plain and a detailed export of the same conversation.
- **A JSON export imports as the same conversation**: architecture test — the
  test asserts the encoder and decoder are the pair `conversations.json` uses
  and that the import path decodes `Conversation`. The round trip itself: a
  recorded hand check, exporting, importing and comparing every turn.
- **Export All writes one folder holding every conversation, and no window
  carries a button for it**: architecture test — the test asserts `"Export
  All"` appears inside the `Commands` builder and nowhere else in
  `client/DessauChat/`. The folder's contents: a recorded hand check with
  several conversations, including one with a name that already exists.
- **A hundred-turn conversation is written whole and the app stays
  responsive**: architecture test — the test asserts the export action runs
  in a detached task and that the renderers take values rather than the
  observable model. Responsiveness: a recorded hand check against a
  conversation of a hundred turns, with the window scrolled while it writes.

## Departures

None at the time of writing.
