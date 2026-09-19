---
schema_version: 1
id: "iss-2609190004097595"
slug: "the-chat-client-s-header-varies-in-height-with-state-and-is"
severity: "major"
category: "ux"
source: "user-observation"
found_during: "manual-test"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/GropiusChat.swift"
resolution: "The header is pinned to one height on both systems: the toolbar's title display mode is inline rather than automatic, so it is never drawn large and never collapses; the transcript's top edge takes the hard scroll-edge style, so the effect ends at the toolbar instead of blurring the reply under it; the transcript anchors only its initial offset to the bottom, so a chat shorter than the window sits under the toolbar rather than at the window's foot; and the answerer button's icon keeps one size across the connecting state."
impact: fix
---

The chat client's header varies in height with state and is far too tall in some: on a long scrolled reply the toolbar's scroll-edge region blurs and washes out roughly the top half of the window (screenshot 2026-09-18 14:31), and on a fresh conversation an empty band sits between the toolbar and the first bubble (screenshot 13:43). Apple Messages keeps one thin toolbar whatever the scroll state; the header must be one fixed height and the scroll-edge effect confined to it.

## Grounds

- pursued: we expect one thin toolbar in every state because the three constructs that made it vary — an automatic title mode that resolves to a large collapsing title, an automatic scroll-edge effect that extends over content, and a scroll anchor that sets the alignment role — are each pinned; wrong if the header still changes height in a state neither screenshot covered, or if the hard scroll-edge style reads as heavier than Messages' on macOS.
