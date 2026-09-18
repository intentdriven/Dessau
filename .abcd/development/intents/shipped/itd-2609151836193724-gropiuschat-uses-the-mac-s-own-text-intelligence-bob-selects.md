---
id: itd-2609151836193724
slug: gropiuschat-uses-the-mac-s-own-text-intelligence-bob-selects
spec_id: spc-2609170842097413
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609151701196720]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# GropiusChat uses the Mac's own text intelligence

## Press Release

GropiusChat uses the Mac's own text intelligence. Bob selects a reply, or
his own draft in the composer, and the system's Writing Tools appear the way
they do in every Apple app — proofread, rewrite, summarise — working on the
text where it is. In the composer the rewrite lands in place of what he
selected; on a reply it opens in the system's panel, ready to copy. The
client sends nothing to Gropius or anywhere else for this: it is the Mac's
on-device feature, present because the client is a native macOS 27 app.
Alice, running the server, sees no request from it.

## Why This Matters

Writing Tools are what a Mac user reaches for when a draft is not right, and
they are absent from the client only because its composer was a hand-built
text view. A native app gets them by being native; keeping them out would be
the styling the trunk removes, in another form.

## Mechanism

We expect Writing Tools to appear without code of the client's own because
SwiftUI's `TextField` and a selectable `Text` carry them on macOS when Apple
Intelligence is on, and the trunk makes the composer a standard `TextField`.
What would show this wrong: a selectable `Text` whose context menu on macOS
27 shows no Writing Tools item, in which case the reply views need
`writingToolsBehavior(.complete)` or a `TextEditor` to carry them.

## Scope Conditions

- Apple Intelligence switched on, on a Mac and in a language the system <!-- cond: cond-2609170842094850 -->
  supports; the client neither checks nor shows an error when it is off — the
  system decides whether the menu item appears.
- The 27 client only, on the trunk's standard composer: a multi-line <!-- cond: cond-2609170842097488 -->
  `TextField` sends on Return and takes a line break on Option-Return, and a
  rewrite that carries line breaks lands in the field as text, not as a send.
- The reply view is shared with the formatted-replies and text-effects <!-- cond: cond-2609170842096052 -->
  intents: a reply is one or more selectable `Text` blocks, and an effect
  never leaves a block unselectable once it has played.
- The first three criteria are checked by hand on the maintainer's Mac and <!-- cond: cond-2609170842092363 -->
  recorded in the shipping decision line; the fourth is the architecture
  test.

## Acceptance Criteria

- Given Apple Intelligence is on, when Bob selects words in a reply and opens
  the context menu, then Writing Tools is there and a rewrite opens in the
  system's panel, read-only, with the result ready to copy; the reply itself
  never becomes editable.
- Given the composer holds a draft, when Bob selects it and chooses a rewrite
  from Writing Tools, then the rewritten text replaces the selection in the
  composer and nothing is sent to any server.
- Given Apple Intelligence is off, when Bob selects text, then no Writing
  Tools item appears and the client shows no error of its own.
- Given the client's source, when the architecture tests run, then the
  composer is a standard `TextField` and no `writingToolsBehavior(.disabled)`
  appears anywhere in the client.

## Open Questions

_None recorded yet._

## Audit Notes

<!-- abcd-review: OWED receipt=rcp-d6fd7dcb1049 -->
Fidelity review OWED (receipt rcp-d6fd7dcb1049).

## Grounds

- pursued: we expect Writing Tools to arrive with no code of the client's own once the composer is a standard TextField and replies are selectable Text, because the system attaches them to those controls; wrong if a selectable Text on macOS 27 shows no Writing Tools in its context menu
