---
schema_version: 1
id: "iss-2610071209008023"
slug: "the-self-test-skips-every-model-that-cannot-chat-so-a"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "maintainer interview on unsettled issues, 2026-10-07"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/app/selftest.go"
remedy: "After itd-2610030656210408 ships, add a short decision-model test to the self-test through the decision route, recorded beside the chat figures; until then the skip stands."
---

The self-test skips every model that cannot chat, so a decision model is never measured at all. Once itd-2610030656210408 lands, a decision model answers typed questions through its own route, and a short self-test there (latency for one typed question and its odds) would give Alice the same kind of figure chat models get, without a chat benchmark's cold load.
