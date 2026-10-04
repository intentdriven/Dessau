---
schema_version: 1
id: "iss-2610042030092101"
slug: "a-bridged-request-that-fails-with-nothing-written-or-that"
severity: "major"
category: "bug"
source: "impl-review"
found_during: "independent review of fix/bridge-history-opens-on-user, 2026-10-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/bridge/discord/answer.go"
remedy: "In buildRequest, drop any user turn that is directly followed by another user turn, keeping the newest, so the history sent alternates whatever the store holds."
refines: [iss-2610032306154631]
---

A bridged request that fails with nothing written, or that the model answers with nothing, leaves the user turn it appended unanswered, so the channel's next message sends two user turns in a row. Gemma and Mistral v0.1/v0.2 chat templates require strict user/assistant alternation and raise on that, so every later message in the channel fails the same way and the channel is stuck until /reset.
