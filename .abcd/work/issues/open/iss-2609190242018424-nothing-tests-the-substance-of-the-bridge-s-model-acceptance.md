---
schema_version: 1
id: "iss-2609190242018424"
slug: "nothing-tests-the-substance-of-the-bridge-s-model-acceptance"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "fidelity audit of itd-2609180959397172"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/bridge/discord/fake_test.go"
---

Nothing tests the substance of the bridge's /model acceptance criterion: picking a model and seeing the next message in that channel answered by it. The fake Discord and gateway in internal/bridge/discord/fake_test.go:92 offer exactly one chat model, so the existing tests cover bare /model showing the current model and an unknown name being refused, but never a second valid model being chosen and taking effect. The set-then-answer path — commands.go storing the choice on the channel's Conversation and answer.go reading it back before asking the gateway — is established by reading the code alone. Giving the fake a second ready chat model and driving /model then a message would hold the criterion the way the others are held.
