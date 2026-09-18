---
schema_version: 1
id: "iss-2609180608297110"
slug: "connect-moves-the-chosen-model-to-the-first-chat-model-of-wh"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "adversarial review of feat/client-macos-27, 2026-09-17"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/GropiusChat.swift"
resolution: "connect() no longer moves selectedModel; a model the server does not serve is simply not offered."
impact: fix
---

connect() moves the chosen model to the first chat model of whatever server answered, so expanding a found server in the picker can switch the answerer without a pick, against the offer's 'nothing switches by itself'.
