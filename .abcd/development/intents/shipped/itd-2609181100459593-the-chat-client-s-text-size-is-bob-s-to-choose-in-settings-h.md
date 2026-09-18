---
id: itd-2609181100459593
slug: the-chat-client-s-text-size-is-bob-s-to-choose-in-settings-h
spec_id: spc-2609181100463204
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609151701196720]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# The chat client's text size is Bob's to choose

## Press Release

The chat client's text size is Bob's to choose. In Settings he picks one of
five sizes — smaller, the system's default, larger, extra large, huge — and
the whole window follows: the conversation, the sidebar, the toolbar and
Settings alike. Every font stays the system's, at a different scale, so the
window still looks like a Mac app at every size. The choice persists across a
relaunch. Alice, running the server, sees nothing of it.

## Why This Matters

A chat window is read for long stretches; the system's default is right for
most eyes and wrong for some, and a person who needs larger text should not
have to change their whole Mac for one app.

## Mechanism

We expect one setting to scale the whole window without touching a single
font because SwiftUI's Dynamic Type sizes are an environment value every
standard control and every `Text` already reads, and a size set at the
window's root reaches all of them. What would show this wrong: a control that
ignores the environment's size on macOS 27, or a layout that breaks at the
largest step.

## Scope Conditions

- The five steps are the system's own Dynamic Type sizes (small, large as <!-- cond: cond-2609181100466055 -->
  the default, extra large, double extra large, triple extra large); the
  client stores the choice, never a point size.
- The size is set at the root of every window and of Settings, and applies <!-- cond: cond-2609181100463398 -->
  to the Mac and the iPad clients alike.
- The maintainer's decision at the interview, 2026-09-18: the whole window, <!-- cond: cond-2609181100469817 -->
  not the conversation alone.

## Acceptance Criteria

- Given Settings open, when Bob picks "Larger", then the conversation, the
  sidebar, the toolbar labels and Settings itself grow together, and the
  layout holds.
- Given a size chosen, when Bob quits and relaunches, then the window opens
  at that size.
- Given "Default" chosen, when the window is shown, then it is
  indistinguishable from the client before this change.
- Given the client's source, when the architecture tests run, then the size
  is applied through the Dynamic Type environment at the window and Settings
  roots and the Settings picker offers exactly the five steps.

## Open Questions

_None recorded yet._

## Audit Notes

<!-- abcd-review: OWED receipt=rcp-dbe9ae70feb0 -->
Fidelity review OWED (receipt rcp-dbe9ae70feb0).

## Grounds

- pursued: we expect one Dynamic Type size at the window's root to scale every control without touching a font; wrong if a control on macOS 27 ignores the environment or the layout breaks at the largest step
