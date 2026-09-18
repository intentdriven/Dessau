---
id: spc-2609170842098587
slug: replies-are-shown-formatted-when-a-model-s-reply-carries-mar
intent: itd-2609170836331240
origin: researcher-authored
production_mode: hand-written
---
# Replies are shown formatted

## Summary

A reply is rendered from `AttributedString(markdown:options:)` with the full
interpreted syntax and the partial-parse failure policy, split into blocks on
its presentation intents — paragraph, heading (level), unordered and ordered
list items (one level, with marker), block quote, code block — in a new file
`client/GropiusChat/Markdown.swift` (`MarkdownBlocks`). `MessageRow` draws one
selectable `Text` per block: headings a size up, list items with their marker
in a leading column, block quotes with a leading bar, code blocks from their
raw range in the monospaced face inside a standard `GroupBox`. Constructs the
intents describe that `Text` cannot draw — tables, nested lists — fall back
to the block's plain text. The reply's context menu gains **Copy**, which
copies `message.text`.

## Scope

In scope: `Markdown.swift`; `MessageRow`'s block layout; the parse cache
(finished replies parsed once, keyed by message id and text length; the
streaming reply re-parsed at most four times a second through a debounce on
the row); the Copy item; the client README's sentence on formatting and on
what Copy copies; an architecture test that reads `Markdown.swift` for the
`.full` syntax and the failure policy.

Out of scope: tables, nested lists, images, HTML; syntax highlighting inside
code blocks; rendering the person's own messages (they stay plain, as typed).

## Approach

`MarkdownBlocks.parse(_:)` walks the attributed string's runs by
`presentationIntent`, groups consecutive runs by their outermost block
intent's identity, and yields `[Block]` where a block is `.paragraph(AttributedString)`,
`.heading(level, AttributedString)`, `.listItem(ordinal: Int?, AttributedString)`,
`.quote(AttributedString)` or `.code(String)`. Inline intents (emphasis,
strong, code, link) are already carried in the runs and `Text` draws them;
the presentation intents are stripped from the block's string before it is
handed to `Text`, so no marker or raw mark shows. A code block's text is
taken from the original source range so line breaks survive. While
streaming, an unterminated mark stays a raw mark until it closes, which is
what the parser yields.

The effects intent tags words per block; the Writing Tools intent relies on
each block staying a selectable `Text`.

## How the acceptance criteria are met

1. Inline marks rendered, none visible — the runs' inline intents.
2. Blocks laid out by kind — `MarkdownBlocks` and the row's layout.
3. Copy is the model's text — the context-menu item over `message.text`.
4. A table is shown as text — the fallback.
5. Streaming shows the text so far — the debounced re-parse.

## Verification

Suite green; `client/build.sh` builds; the five constructs checked by hand
with a model on the maintainer's Mac and recorded.
