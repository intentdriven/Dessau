---
schema_version: 1
id: "iss-2609180608290286"
slug: "the-effect-never-plays-messagerow-reads-effectstoplay-in-bod"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "adversarial review of feat/client-macos-27, 2026-09-17"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/GropiusChat.swift"
resolution: "MessageRow latches the decision in its own state on first appearance and consumes the model's set at that moment."
impact: fix
---

The effect never plays: MessageRow reads effectsToPlay in body and consumes it in onAppear, the model publishes, the row re-evaluates with animate false and the EffectText branch is destroyed after one frame. The decision must be latched in the row's own state.
