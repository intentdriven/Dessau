---
schema_version: 1
id: "iss-2609181119348004"
slug: "the-chunkdelay-option-says-it-paces-the-chunks-after-the-fir"
severity: "nitpick"
category: "documentation"
source: "impl-review"
found_during: "adversarial review of fix/open-captures-2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/mlxtest/fake.go"
resolution: "The ChunkDelay doc says it paces every chunk after the first, and names the preamble-to-first-word gap as what a measurement of the first chunk is told apart by."
impact: internal
---

the ChunkDelay option says it paces the chunks after the first, which is no longer the whole truth: it also paces the first word after a role-only preamble
