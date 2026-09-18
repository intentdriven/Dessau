---
id: itd-2609170836331240
slug: replies-are-shown-formatted-when-a-model-s-reply-carries-mar
spec_id: spc-2609170842098587
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609151701196720]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# Replies are shown formatted

## Press Release

Replies are shown formatted. When a model's reply carries markdown — a word
between asterisks, a code span, a list, a heading, a fenced code block, a
link — Bob sees it rendered the way a Mac app shows rich text: emphasis in
italics, code in the system's monospaced face, lists with their markers,
headings a size up, code blocks set apart and links he can click. The raw
marks never show. What the model wrote is what is saved, and a reply's context
menu offers **Copy** which copies the model's own text, marks and all;
dragging over rendered text copies what is shown. Alice, running the
server, sees no difference: the client renders what it already receives.

## Why This Matters

Every chat model writes markdown, and a client that shows the asterisks is
showing its seams. Rendering it is what every chat app a Mac user knows does,
and the system parses markdown itself, so the client has nothing to bring in.

## Mechanism

We expect formatted replies without a third-party renderer because the
system's `AttributedString(markdown:)` parses the full syntax into
presentation intents when parsed with the full interpreted syntax and a
partial-parse failure policy, and a reply split into blocks on those intents
— paragraph, heading, list item, block quote, code block — draws each with
the standard `Text`, which already renders inline emphasis, code and links;
a code block is drawn from its raw range, so its line breaks survive the
whitespace folding the parser applies to prose. What would
show this wrong: a construct the intents describe that `Text` cannot draw
(a table, nested lists beyond one level), in which case that construct is
shown as the model's text and named in the docs as not rendered.

## Scope Conditions

- Rendered: inline emphasis, strong, code spans and links; block-level <!-- cond: cond-2609170842099020 -->
  paragraphs, headings, bulleted and numbered lists (one level), block
  quotes and fenced code blocks. Tables and nested lists are shown as the
  model's text. The maintainer's request, 2026-09-17.
- Rendering happens on the finished reply and, while streaming, on the text <!-- cond: cond-2609170842090302 -->
  so far; an unterminated mark while streaming shows as the raw mark until
  it closes.
- The 27 client only. <!-- cond: cond-2609170842095759 -->
- Cost: a reply that has finished is parsed once and the blocks are kept; <!-- cond: cond-2609170842096701 -->
  the reply that is streaming is re-parsed at most a few times a second, not
  on every delta.
- The code block's "set apart" is a standard grouped container, not a <!-- cond: cond-2609170842097453 -->
  client-drawn background; if a container is not standard, the no-styling
  test names the exception with its reason.
- The reply view is shared with the Writing Tools and text-effects intents: <!-- cond: cond-2609170842097888 -->
  every block stays a selectable `Text`.

## Acceptance Criteria

- Given a reply containing `*word*`, `**word**`, `` `code` `` and a link,
  when it is shown, then the word is italic, the second bold, the code in
  the monospaced face, the link clickable, and no asterisk or backtick is
  visible.
- Given a reply with a heading, a bulleted list, a numbered list and a
  fenced code block, when it is shown, then each is laid out as its kind:
  the heading larger, the lists with markers, the code block set apart in
  the monospaced face with its line breaks kept.
- Given a formatted reply, when Bob chooses Copy from its context menu or
  saves the chat, then the text is the model's text with its marks
  unchanged.
- Given a reply with a markdown table, when it is shown, then the table's
  text is shown as the model wrote it and nothing is dropped.
- Given a reply is streaming, when a mark has opened and not yet closed,
  then the text so far is shown and the client neither hides nor
  mis-renders it; when the mark closes, the rendered form appears.

## Open Questions

_None recorded yet._

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-e0b9d86d3751 -->
Fidelity review — receipt rcp-e0b9d86d3751 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:de06b547e8444f8fe3b5e5bd843b910d3a1c636a9522943d1fc0cd5d550f237c
Input attestations: diff:acacd07..210ce66 -- client internal/archtest install.sh .github README.md docs CHANGELOG.md (origin/main at 400ccd7)@sha256:da0c18bfb59c2c67e4e35a363d5d29a445eb86969c738f5c943f26fd3167cbdb;

Acceptance rollup: MET 1 · MET_WITH_CONCERNS 4 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: the parser consumes the marks and the block is handed to Text with only its inline intents left, so no asterisk or backtick can reach the screen; but the italic/bold/monospaced/clickable appearance is delegated wholly to SwiftUI's Text(AttributedString) and nothing committed exercises it — the Swift client cannot be built or run here and the architecture test reads only two option strings
  evidence: client/GropiusChat/Markdown.swift:80 — "s.presentationIntent = nil"
  evidence: client/GropiusChat/GropiusChat.swift:1300 — "Text(s).textSelection(.enabled)"
  evidence: internal/archtest/chat_client_native_test.go:211 — "for _, want := range []string{"interpretedSyntax: .full", "failurePolicy: .returnPartiallyParsedIfPossible"} {"
- ac-2 — MET_WITH_CONCERNS: headings take a larger font, list items draw an ordinal or bullet in a leading column and the code block is set in a standard GroupBox in the monospaced face — but the promised line-break preservation is not delivered the way the mechanism said: the code block is emitted from the parsed attributed string's characters, not from its raw source range, so its line breaks rest on undemonstrated Foundation parser behaviour
  evidence: client/GropiusChat/GropiusChat.swift:1241 — ".font(level <= 1 ? .title2 : level == 2 ? .title3 : .headline)"
  evidence: client/GropiusChat/GropiusChat.swift:1244 — "Text(ordinal.map { "\($0)." } ?? "•").foregroundStyle(.secondary)"
  evidence: client/GropiusChat/GropiusChat.swift:1256 — ".font(.body.monospaced())"
  evidence: client/GropiusChat/Markdown.swift:86 — "case .code: blocks.append(.code(String(s.characters)))"
- ac-3 — MET: the reply's context menu writes message.text — the model's own unrendered text — to the pasteboard, and the saved chat is a JSON encoding of the same Message.text, so neither path touches the rendered form
  evidence: client/GropiusChat/GropiusChat.swift:1219 — "Button("Copy") {"
  evidence: client/GropiusChat/GropiusChat.swift:1221 — "NSPasteboard.general.setString(message.text, forType: .string)"
  evidence: client/GropiusChat/GropiusChat.swift:596 — "if let data = try? JSONEncoder().encode(conversations) {"
- ac-4 — MET_WITH_CONCERNS: runs of lines beginning with a pipe are cut out before parsing and emitted verbatim as a .plain block drawn in the monospaced face, which satisfies the criterion for the conventional table; the concern is the second path — a table whose rows carry no leading pipe reaches parseProse, and the .table intents there fall back to the PARSED cell text, from which the pipe separators have already been dropped
  evidence: client/GropiusChat/Markdown.swift:38 — "blocks.append(.plain(segment.trimmingCharacters(in: .newlines)))"
  evidence: client/GropiusChat/Markdown.swift:119 — "case .table, .tableRow, .tableCell, .tableHeaderRow: return .plain("")"
  evidence: client/GropiusChat/Markdown.swift:87 — "case .plain: blocks.append(.plain(String(s.characters)))"
  evidence: client/GropiusChat/GropiusChat.swift:1262 — ".font(.body.monospaced())"
- ac-5 — MET_WITH_CONCERNS: the streaming text is re-parsed on every change behind a 0.2s debounce, the partial-parse failure policy keeps an unterminated mark from losing the text, and a parse that throws falls back to the whole text as a plain block so nothing is hidden; the concern is that no committed test exercises a mark that has opened and not closed — the only record of it is the ad-hoc harness noted in the decision line, and the client cannot be run here
  evidence: client/GropiusChat/GropiusChat.swift:1280 — "private func scheduleParse() {"
  evidence: client/GropiusChat/Markdown.swift:30 — "failurePolicy: .returnPartiallyParsedIfPossible)"
  evidence: client/GropiusChat/Markdown.swift:70 — "guard let parsed = try? AttributedString(markdown: text, options: options) else {"
  evidence: .abcd/work/DECISIONS.md:277 — "a harness on this Mac streamed nine snapshots from the on-device model"

Gap audit:
- honoured:
  - replies are rendered from the system's markdown parser with no third-party renderer, split into blocks on their presentation intents
    evidence: client/GropiusChat/Markdown.swift:27 — "static let options = AttributedString.MarkdownParsingOptions("
    evidence: client/GropiusChat/Markdown.swift:94 — "let key = intent?.components.map(\.identity) ?? []"
  - a reply's context menu offers Copy and it copies the model's own text, marks and all
    evidence: client/GropiusChat/GropiusChat.swift:1221 — "NSPasteboard.general.setString(message.text, forType: .string)"
  - the code block is set apart in a standard grouped container rather than a client-drawn background
    evidence: client/GropiusChat/GropiusChat.swift:1254 — "GroupBox {"
  - the client README states what is rendered and what Copy copies
    evidence: client/README.md:62 — "Replies render their markdown — emphasis, code, lists, headings, block quotes,"
  - an architecture test pins the full interpreted syntax and the partial-parse failure policy
    evidence: internal/archtest/chat_client_native_test.go:205 — "func TestChatClientRendersMarkdownWithTheFullSyntax(t *testing.T) {"
- diverged:
  - the mechanism promised a code block drawn from its raw range so its line breaks survive; it is delivered from the parsed attributed string's characters instead
    evidence: client/GropiusChat/Markdown.swift:86 — "case .code: blocks.append(.code(String(s.characters)))"
  - the spec's parse cache keyed by message id and text length is delivered as per-row @State, so a row recreated on scroll re-parses a finished reply
    evidence: client/GropiusChat/GropiusChat.swift:1172 — "@State private var blocks: [MarkdownBlock] = []"
    evidence: client/GropiusChat/GropiusChat.swift:1273 — "if blocks.isEmpty { blocks = MarkdownBlocks.parse(displayText); lastParse = Date() }"
  - the spec said the streaming reply is re-parsed at most four times a second; the debounce admits five
    evidence: client/GropiusChat/GropiusChat.swift:1281 — "let wait = 0.2 - Date().timeIntervalSince(lastParse)"
  - the promise that nested lists are shown as the model's text is not kept — a nested list is flattened into one-level list items, only tables fall back
    evidence: client/GropiusChat/Markdown.swift:126 — "return .listItem(ordinal: ordered ? ordinal : nil, AttributedString())"
    evidence: client/GropiusChat/GropiusChat.swift:1247 — ".padding(.leading, 8)"
  - every block stays a selectable Text, except the effect-animating branch, which draws a Text carrying no textSelection for the ~1.3s the effect plays
    evidence: client/GropiusChat/Effects.swift:124 — "tagged"
- missing:
  - the spec's verification — the five constructs checked by hand with a model and recorded — is not evidenced for the RENDERED appearance; the decision line records only the harness's block split, and no committed test asserts italic, bold, monospaced, a clickable link or a kept line break
    evidence: .abcd/work/DECISIONS.md:277 — "split every markdown construct into its block"
    evidence: internal/archtest/chat_client_native_test.go:211 — "for _, want := range []string{"interpretedSyntax: .full", "failurePolicy: .returnPartiallyParsedIfPossible"} {"
  - the docs name nothing as not rendered — the mechanism said an undrawable construct is 'named in the docs as not rendered', but the client README lists only what IS rendered, with no mention of tables or nested lists
    evidence: client/README.md:62 — "Replies render their markdown — emphasis, code, lists, headings, block quotes,"

Scope-condition dispositions:
- cond-2609170842099020 — narrowed: every promised inline and block kind has a case and a layout, and tables do fall back to the model's text, but the nested-list half of the fallback was never built
  narrowing: holds for the rendered set and for tables only; a nested list is not shown as the model's text but flattened into one-level list items, since kind(of:) maps any listItem component to .listItem with no depth check and the row draws a fixed leading inset
  evidence: client/GropiusChat/Markdown.swift:119 — "case .table, .tableRow, .tableCell, .tableHeaderRow: return .plain("")"
  evidence: client/GropiusChat/Markdown.swift:116 — "case .listItem(let n): ordinal = n"
  evidence: client/GropiusChat/GropiusChat.swift:1247 — ".padding(.leading, 8)"
- cond-2609170842090302 — survived: the row parses on appearance and again on every text change behind a debounce, and the partial-parse policy is what leaves an unterminated mark standing as raw text until it closes
  evidence: client/GropiusChat/GropiusChat.swift:1275 — ".onChange(of: displayText) { _, _ in scheduleParse() }"
  evidence: client/GropiusChat/Markdown.swift:30 — "failurePolicy: .returnPartiallyParsedIfPossible)"
- cond-2609170842095759 — survived: the rendering work of this intent lands only in the Swift client and its architecture test — a new client file, the client's MessageRow and the client README — and the client's bundle floor is macOS 27
  evidence: client/GropiusChat/Markdown.swift:1 — "// A reply's markdown, rendered: the system's parser turns the text into an"
  evidence: client/Info.plist:22 — "< string>27.0< /string>"
- cond-2609170842096701 — narrowed: the streaming ceiling is enforced by a debounce and a finished reply is not re-parsed while its row lives, but the blocks are kept on the view rather than in the cache the spec described
  narrowing: holds per live row, not per message: the blocks are @State on MessageRow rather than a cache keyed by message id and text length, so a row recreated (on scroll, for instance) re-parses a finished reply; and the streaming ceiling delivered is the 0.2s debounce, up to five parses a second rather than the spec's four
  evidence: client/GropiusChat/GropiusChat.swift:1172 — "@State private var blocks: [MarkdownBlock] = []"
  evidence: client/GropiusChat/GropiusChat.swift:1273 — "if blocks.isEmpty { blocks = MarkdownBlocks.parse(displayText); lastParse = Date() }"
  evidence: client/GropiusChat/GropiusChat.swift:1281 — "let wait = 0.2 - Date().timeIntervalSince(lastParse)"
- cond-2609170842097453 — survived: the code block is drawn inside a standard GroupBox with no client-drawn background, so the no-styling test needs no exception for it — and that test would reject one, since it bans .background( outright
  evidence: client/GropiusChat/GropiusChat.swift:1254 — "GroupBox {"
  evidence: internal/archtest/chat_client_native_test.go:45 — "".buttonStyle(.glass", ".background(", ".overlay(", ".shadow(","
- cond-2609170842097888 — narrowed: every block the reply draws is a Text and every non-animating one enables text selection, but the branch the text-effects intent takes is a concatenated Text with no textSelection modifier
  narrowing: holds for every block except one that is animating: while the effect plays (about 1.3 seconds, on a reply that earned it) the block is drawn as `tagged` with a text renderer and no .textSelection(.enabled), so it is a Text but not a selectable one until the animation finishes
  evidence: client/GropiusChat/GropiusChat.swift:1300 — "Text(s).textSelection(.enabled)"
  evidence: client/GropiusChat/Effects.swift:122 — "Text(text).textSelection(.enabled)"
  evidence: client/GropiusChat/Effects.swift:125 — ".textRenderer(EffectRenderer(progress: progress))"
## Grounds

- pursued: we expect the system's markdown parser plus a block split on presentation intents to render what chat models write without a dependency, because Text already draws inline styles and the intents name the blocks; wrong if the constructs Text cannot draw (tables, nested lists) turn out to be common in replies
