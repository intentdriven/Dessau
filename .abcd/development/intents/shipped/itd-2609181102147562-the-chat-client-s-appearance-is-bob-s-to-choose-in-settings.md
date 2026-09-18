---
id: itd-2609181102147562
slug: the-chat-client-s-appearance-is-bob-s-to-choose-in-settings
spec_id: spc-2609181102146375
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609151701196720]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# The chat client's appearance is Bob's to choose

## Press Release

The chat client's appearance is Bob's to choose. In Settings he picks Light,
Dark or System, and the whole window follows at once — the conversation, the
sidebar, the toolbar and Settings alike. System means the client follows the
Mac's own appearance, as it does today. The choice persists across a
relaunch. Alice, running the server, sees nothing of it.

## Why This Matters

A chat window is often the one window someone wants dark at night or light
beside a document, without changing the whole Mac. Every colour the client
uses is the system's, so an appearance chosen here still looks like a Mac
app.

## Mechanism

We expect one setting to switch the whole window because SwiftUI's preferred
colour scheme, set at a scene's root, is what every standard control and the
system's colours resolve against, and no preference at all is exactly the
system's behaviour. What would show this wrong: a view that resolves a colour
outside the environment on macOS 27, or a window that ignores the preference
until it is reopened.

## Scope Conditions

- Three choices, and no more: Light, Dark, System; System is the absence of a <!-- cond: cond-2609181102149647 -->
  preference, not a third scheme.
- Applied at the root of every window and of Settings, on the Mac and the <!-- cond: cond-2609181102147799 -->
  iPad clients alike; the bubble colours and every other colour stay the
  system's and follow.

## Acceptance Criteria

- Given Settings open, when Bob picks Dark, then every open window and
  Settings turn dark at once without being reopened; when he picks Light,
  they turn light.
- Given System chosen, when the Mac's appearance changes, then the client
  follows it, as it did before this change.
- Given a choice made, when Bob quits and relaunches, then the window opens
  in that appearance.
- Given the client's source, when the architecture tests run, then the
  preference is applied at the window and Settings roots and the picker
  offers exactly Light, Dark and System.

## Open Questions

_None recorded yet._

## Audit Notes

<!-- abcd-review: OWED receipt=rcp-df22ad80d10d -->
Fidelity review OWED (receipt rcp-df22ad80d10d).

## Grounds

- pursued: we expect the preferred colour scheme at the scene roots to switch every window at once with no colour of the client's own; wrong if a window ignores the preference until reopened
