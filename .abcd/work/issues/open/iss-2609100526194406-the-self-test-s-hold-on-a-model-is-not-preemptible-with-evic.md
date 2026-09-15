---
schema_version: 1
id: "iss-2609100526194406"
slug: "the-self-test-s-hold-on-a-model-is-not-preemptible-with-evic"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "itd-2609100457007827 security review"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/runtime/pool.go"
---

The self-test's hold on a model is not preemptible: with eviction grace off, a client whose load needs the memory the self-test holds is refused once (503) before the self-test sees the refusal count move and yields, so the client's retry finds room. The correct fix is pool-side: a soft, source-tagged hold that the eviction plan may take and that cancels the holder's context, so the client's Acquire evicts the self-test's model instead of being refused. Found by the security review of itd-2609100457007827; the counter (Residency.Refusals) is the interim.
