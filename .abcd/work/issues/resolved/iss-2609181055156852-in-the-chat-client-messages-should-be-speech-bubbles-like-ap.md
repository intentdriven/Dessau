---
schema_version: 1
id: "iss-2609181055156852"
slug: "in-the-chat-client-messages-should-be-speech-bubbles-like-ap"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "the maintainer's manual test of the 0.7.0 client, 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/GropiusChat.swift"
resolution: "Bubbles.swift draws a Messages-style bubble in the speaker's colour behind the text — the system's accent colour for the person, grey for the model, both stored as hex and changeable through two colour pickers in Settings with a Default button — and is the no-styling test's second named exception; held by TestChatClientDrawsMessagesAsBubbles."
impact: fix
---

In the chat client, messages should be speech bubbles like Apple Messages': the person's in the system's accent colour, the model's in grey by default, both colours changeable in Settings (the maintainer's manual test, 2026-09-18). This reverses the trunk's 'no styling of its own' criterion for the transcript; the bubble is a deliberate, named exception to the no-styling test, drawn with the system's colours.

## Grounds

- pursued: we expect Messages-style bubbles in system colours to read as native even though they are drawn by the client; wrong if the maintainer finds the tinted bubbles clash with the 27 design language
