---
id: itd-2609181104497297
slug: a-model-s-thoughts-are-formatted-like-its-replies-when-a-thi
spec_id: spc-2609181104493254
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609151701196720, itd-2609170836331240]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# A model's thoughts are formatted like its replies

## Press Release

A model's thoughts are formatted like its replies. When a thinking model's
reasoning carries markdown — italics, bold, code, lists — Bob sees it
rendered in the Thoughts row the way the reply below it is, never the raw
marks. Copying the reply still copies what the model wrote.

## Why This Matters

Thinking models write their reasoning in the same markdown as their
answers; a Thoughts row that shows asterisks beside a reply that renders
them is a seam.

## Mechanism

We expect the Thoughts row to render with no new code because the
formatted-replies intent's block parser already turns a reply into blocks
drawn by standard Text, and the reasoning is text of the same kind. What
would show this wrong: reasoning that streams in fragments the parser
mis-renders until it is complete.

## Scope Conditions

- The same parser and the same blocks as the reply (itd-2609170836331240), <!-- cond: cond-2609181104491038 -->
  which this refines; the same throttle while streaming.
- The reasoning stays collapsed by default and is not animated by the <!-- cond: cond-2609181104499446 -->
  effects intent.

## Acceptance Criteria

- Given a reply with reasoning containing `*word*` and `**word**`, when the
  Thoughts row is expanded, then the words are italic and bold and no
  asterisk is shown.
- Given the client's source, when the architecture tests run, then the
  Thoughts row draws its text through the markdown blocks.

## Open Questions

_None recorded yet._

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-4f2011a3e0ab -->
Fidelity review — receipt rcp-4f2011a3e0ab (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:64d984396e7e1d666fc4e38b4f4c0cf5c8399dd9635b804737f8683ea1c3abb7
Input attestations: diff:c2bd0ed..6534be1 -- client internal/archtest (PR 66)@sha256:8360f15991fd63f4f790a03f2425c4a91bb7454074a488ee8fa21a79201551b4;

Acceptance rollup: MET 1 · MET_WITH_CONCERNS 1 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: The Thoughts row now draws reasoningBlocks through the shared blockViews builder, and MarkdownBlocks.parse turns *word*/**word** into AttributedString inline intents that Text draws with the marks consumed, so no asterisk survives; two named concerns: the visual outcome is unverifiable here (the Swift client has no test target and cannot be compiled or run, and the spec's 'checked by hand with a thinking model' is not recorded as performed in .abcd/work/DECISIONS.md), and the whole Thoughts block already carries .italic(), so an emphasised word is italic inside already-italic callout text and is not visually distinguishable from its surroundings.
  evidence: client/GropiusChat/GropiusChat.swift:1356 — "blockViews(reasoningBlocks, animate: false)"
  evidence: client/GropiusChat/GropiusChat.swift:1357 — ".font(.callout).italic()"
  evidence: client/GropiusChat/Markdown.swift:69 — "guard let parsed = try? AttributedString(markdown: text, options: options) else {"
  evidence: client/GropiusChat/GropiusChat.swift:1339 — "Text(s).textSelection(.enabled)"
  evidence: .abcd/work/DECISIONS.md:284 — "Reviews scaled to the blast radius: one architecture test each."
- ac-2 — MET: TestChatClientThoughtsRenderMarkdown reads the reasoningDisclosure block for reasoningBlocks and the source for MarkdownBlocks.parse(displayReasoning), and it passes when run here (go test -count=1 -run TestChatClientThoughtsRenderMarkdown ./internal/archtest/ -> PASS).
  evidence: internal/archtest/chat_client_composer_test.go:143 — "func TestChatClientThoughtsRenderMarkdown(t *testing.T) {"
  evidence: internal/archtest/chat_client_composer_test.go:154 — "if !strings.Contains(block, "reasoningBlocks") {"
  evidence: internal/archtest/chat_client_composer_test.go:157 — "if !strings.Contains(src, "MarkdownBlocks.parse(displayReasoning)") {"

Gap audit:
- honoured:
  - The Thoughts row is rendered the way the reply below it is, through the same blocks and the same block view
    evidence: client/GropiusChat/GropiusChat.swift:1264 — "@ViewBuilder private func blockViews(_ blocks: [MarkdownBlock], animate: Bool) -> some View {"
    evidence: client/GropiusChat/GropiusChat.swift:1251 — "blockViews(blocks, animate: animate)"
    evidence: client/GropiusChat/GropiusChat.swift:1356 — "blockViews(reasoningBlocks, animate: false)"
  - Italics, bold, code and lists all reach the Thoughts row, never the raw marks
    evidence: client/GropiusChat/GropiusChat.swift:1274 — "case .listItem(let ordinal, let s):"
    evidence: client/GropiusChat/GropiusChat.swift:1284 — "case .code(let code):"
    evidence: client/GropiusChat/Markdown.swift:28 — "interpretedSyntax: .full,"
  - Copying the reply still copies what the model wrote
    evidence: client/GropiusChat/GropiusChat.swift:1227 — "NSPasteboard.general.setString(message.text, forType: .string)"
- diverged:
  - Mechanism: 'We expect the Thoughts row to render with no new code because the formatted-replies intent's block parser already turns a reply into blocks drawn by standard Text' — delivered with roughly thirty lines of new code: three new @State properties and a duplicated parse/throttle twin rather than reuse of scheduleParse
    evidence: client/GropiusChat/GropiusChat.swift:1179 — "@State private var reasoningBlocks: [MarkdownBlock] = []"
    evidence: client/GropiusChat/GropiusChat.swift:1301 — "private func scheduleReasoningParse() {"
    evidence: client/GropiusChat/GropiusChat.swift:1319 — "private func scheduleParse() {"
  - Press release: 'Bob sees it rendered ... the way the reply below it is' — the reply may animate its blocks, the Thoughts row is pinned to animate: false, and the Thoughts row is additionally styled .callout/.italic/.secondary, so 'the way the reply is' holds for the parse and the blocks, not for the drawing
    evidence: client/GropiusChat/GropiusChat.swift:1356 — "blockViews(reasoningBlocks, animate: false)"
    evidence: client/GropiusChat/GropiusChat.swift:1250 — "let animate = playing ?? false"
- missing:
  - Mechanism names its own falsifier — 'reasoning that streams in fragments the parser mis-renders until it is complete' — and nothing in the delivery exercises it: the only shipped check is a source-text grep, there is no streaming-fragment test and no recorded hand check with a thinking model
    evidence: internal/archtest/chat_client_composer_test.go:146 — "start := regexp.MustCompile(`private var reasoningDisclosure: some View \{`).FindStringIndex(src)"
    evidence: .abcd/work/DECISIONS.md:284 — "Reviews scaled to the blast radius: one architecture test each."
  - The Thoughts row's own textSelection(.enabled) was removed and selection is now per-block inside blockViews, so a drag across two thought blocks is no longer one selection; the row's doc comment still promises 'Dragging still selects the text' and nothing records the change
    evidence: client/GropiusChat/GropiusChat.swift:1352 — "hidden while it is being read. Dragging still selects the text."
    evidence: client/GropiusChat/GropiusChat.swift:1339 — "Text(s).textSelection(.enabled)"

Scope-condition dispositions:
- cond-2609181104491038 — survived: The thoughts go through the same MarkdownBlocks.parse and the same blockViews the reply uses, and scheduleReasoningParse applies the identical 0.2s throttle value while streaming — a separate twin function over separate state, but the same parser, the same blocks and the same throttle the condition assumed.
  evidence: client/GropiusChat/GropiusChat.swift:1264 — "@ViewBuilder private func blockViews(_ blocks: [MarkdownBlock], animate: Bool) -> some View {"
  evidence: client/GropiusChat/GropiusChat.swift:1302 — "let wait = 0.2 - Date().timeIntervalSince(lastReasoningParse)"
  evidence: client/GropiusChat/GropiusChat.swift:1320 — "let wait = 0.2 - Date().timeIntervalSince(lastParse)"
- cond-2609181104499446 — survived: showReasoning still initialises to false so the disclosure is collapsed by default, and the Thoughts row passes animate: false, which routes every block to plain Text rather than the effects intent's EffectText.
  evidence: client/GropiusChat/GropiusChat.swift:1171 — "@State private var showReasoning = false"
  evidence: client/GropiusChat/GropiusChat.swift:1356 — "blockViews(reasoningBlocks, animate: false)"
  evidence: client/GropiusChat/GropiusChat.swift:1336 — "if animate {"
## Grounds

- pursued: we expect the reply's block parser to render the thoughts unchanged; wrong if streamed reasoning fragments render badly until complete
