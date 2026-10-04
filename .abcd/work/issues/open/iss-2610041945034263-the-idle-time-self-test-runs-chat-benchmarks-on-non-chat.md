---
schema_version: 1
id: "iss-2610041945034263"
slug: "the-idle-time-self-test-runs-chat-benchmarks-on-non-chat"
severity: "minor"
category: "architectural-insight"
source: "agent-observation"
found_during: "tracing an unexplained model load after a peer's run, 2026-10-04, v0.9.3"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/selftest/selftest.go"
remedy: "The maintainer decides whether non-chat models are self-tested at all: either skip every model that cannot chat, as Load now does (iss-2610031010371709), or give decision models their own short test through /v1/systemone once itd-2610030656210408 lands; record the choice in docs/self-test.md."
---

The idle-time self-test runs chat benchmarks on non-chat models. At 18:45 on 2026-10-04 it loaded mlx-community/clef-4bit, a decision model that cannot chat, to run pp512, tg128 and tg128xN through chat completions. Its previous self-test record for that model reads 'yielded'. A chat benchmark says nothing useful about a model that answers typed decisions, and it costs a cold load of a large model. Nothing records whether non-chat models (decision models, OCR models) should be self-tested at all.
