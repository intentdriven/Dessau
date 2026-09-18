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

<!-- abcd-review: OWED receipt=rcp-4f2011a3e0ab -->
Fidelity review OWED (receipt rcp-4f2011a3e0ab).

## Grounds

- pursued: we expect the reply's block parser to render the thoughts unchanged; wrong if streamed reasoning fragments render badly until complete
