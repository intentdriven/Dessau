---
schema_version: 1
id: "iss-2609181054478851"
slug: "in-the-chat-client-the-thoughts-row-a-thinking-model-s-reaso"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "the maintainer's manual test of the 0.7.0 client, 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/GropiusChat.swift"
resolution: "The Thoughts row's label and its expanded text each carry a full-width content shape and a tap gesture that toggles the disclosure; held by TestChatClientThoughtsToggleOnAClickAnywhere."
impact: fix
---

In the chat client the Thoughts row (a thinking model's reasoning) expands and collapses only from the small disclosure triangle; clicking anywhere in the row or in the expanded thinking text should toggle it, so the thinking can be hidden while reading it (the maintainer's manual test, 2026-09-18).

## Grounds

- pursued: we expect a click anywhere in the row to be what a reader reaches for; wrong if the tap gesture interferes with selecting the thinking text by drag
