---
id: itd-2609181104498312
slug: bob-finds-a-conversation-from-a-search-bar-at-the-top-of-the
spec_id: spc-2609181104491592
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609151701196720]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# Bob finds a conversation from a search bar in the sidebar

## Press Release

Bob finds a conversation from a search bar at the top of the sidebar, like
Messages': typing filters the list live to the conversations whose title or
messages contain the words, and clearing the search brings every
conversation back.

## Why This Matters

A sidebar of many chats is only useful if the one from last week can be
found by what was said in it.

## Mechanism

We expect the search to be the system's because SwiftUI's searchable
modifier on the sidebar places the field where the platform puts it and
hands the client the text; filtering is one case-insensitive contains over
titles and messages. What would show this wrong: a search field the sidebar
column cannot host at its minimum width.

## Scope Conditions

- The search is over titles and message text, case-insensitive, live on <!-- cond: cond-2609181104490024 -->
  every keystroke; it is the system's searchable field, not a field of the
  client's own.
- An empty search shows every conversation; the selection is kept if the <!-- cond: cond-2609181104498637 -->
  selected conversation still matches.

## Acceptance Criteria

- Given several conversations, when Bob types a word that appears in one
  conversation's messages, then only that conversation is listed; when he
  clears the field, then all are listed again.
- Given the client's source, when the architecture tests run, then the
  sidebar carries the searchable modifier and the filter reads titles and
  messages.

## Open Questions

_None recorded yet._

## Audit Notes

<!-- abcd-review: OWED receipt=rcp-513e379573b4 -->
Fidelity review OWED (receipt rcp-513e379573b4).

## Grounds

- pursued: we expect the system's searchable field on the sidebar to be Messages' search bar for free; wrong if the sidebar column cannot host it at its minimum width
