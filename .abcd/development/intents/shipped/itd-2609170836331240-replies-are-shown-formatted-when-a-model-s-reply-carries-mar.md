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

<!-- abcd-review: OWED receipt=rcp-e0b9d86d3751 -->
Fidelity review OWED (receipt rcp-e0b9d86d3751).

## Grounds

- pursued: we expect the system's markdown parser plus a block split on presentation intents to render what chat models write without a dependency, because Text already draws inline styles and the intents name the blocks; wrong if the constructs Text cannot draw (tables, nested lists) turn out to be common in replies
