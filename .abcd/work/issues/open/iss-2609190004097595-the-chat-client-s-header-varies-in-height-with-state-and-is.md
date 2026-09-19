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
---

The chat client's header varies in height with state and is far too tall in some: on a long scrolled reply the toolbar's scroll-edge region blurs and washes out roughly the top half of the window (screenshot 2026-09-18 14:31), and on a fresh conversation an empty band sits between the toolbar and the first bubble (screenshot 13:43). Apple Messages keeps one thin toolbar whatever the scroll state; the header must be one fixed height and the scroll-edge effect confined to it.
