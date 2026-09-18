---
id: spc-2609170842097413
slug: gropiuschat-uses-the-mac-s-own-text-intelligence-bob-selects
intent: itd-2609151836193724
origin: researcher-authored
production_mode: hand-written
---
# GropiusChat uses the Mac's own text intelligence

## Summary

Nothing is built for Writing Tools; the trunk's standard controls carry them.
The composer is `TextField("Message…", text: $model.input, axis: .vertical)`
with `.lineLimit(1...8)` and `.onSubmit { model.send() }`; replies are
selectable `Text` blocks. This spec fixes what must NOT be done — no
`writingToolsBehavior(.disabled)`, no custom text view — and names the one
architecture test and the three manual checks.

## Scope

In scope: `internal/archtest`'s `TestChatClientKeepsWritingTools`, which
reads every file under `client/GropiusChat/` and fails on
`writingToolsBehavior(.disabled)`, on `NSViewRepresentable` and on
`NSTextView`; the composer's Return/Option-Return behaviour is the standard
control's; the client README says Writing Tools work in the composer and on
replies when Apple Intelligence is on.

Out of scope: any client-side check of Apple Intelligence's state (the system
decides whether the menu item appears); the reply becoming editable.

## Approach

The composer's field takes a rewrite that carries line breaks as text, since a
multi-line `TextField` holds line breaks and `onSubmit` fires only on the
Return key. The reply blocks stay `Text(...).textSelection(.enabled)`; the
system's context menu on a selection carries Writing Tools, and the results
open in the system panel read-only. The effects intent's renderer is applied
only while an effect plays, so a reply is selectable the rest of the time.

## How the acceptance criteria are met

1–3 are manual on the maintainer's Mac with Apple Intelligence on and off,
recorded in the shipping decision line. 4 is the architecture test.

## Verification

Suite green; the manual checks recorded.
