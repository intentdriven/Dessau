---
schema_version: 1
id: "iss-2609200816105905"
slug: "a-long-model-response-renders-as-one-speech-bubble-the-heigh"
severity: "minor"
category: "ux"
source: "manual-test"
found_during: "maintainer manual test of the chat client, 2026-09-20"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/Bubbles.swift"
---

A long model response renders as one speech bubble the height of the whole window: a multi-section answer with headings and bullet lists fills the pane as a single grey block, so a reader cannot tell where one part ends and the next begins, and the composer is pushed to the very bottom. Ask: break a long response into several bubbles, one per section, splitting at the response's own headings (or another stated boundary such as a paragraph count when there are none), so the conversation reads as a sequence of parts rather than one wall. Seen with a text-generation model answering a history question in the macOS client; the response carried an H1, an H2, and several bulleted sections.
