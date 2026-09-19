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
resolution: "The pool gained a soft, caller-tagged hold (runtime.WithSoftHold): the eviction plan may name a model every one of whose holds is preemptible, as a last resort after every idle model, and takes it by asking the holders to let go rather than by stopping anything. The self-test's run context carries that yield, so a client whose load needs the run's memory evicts it instead of being refused, and the run records the yield it already had."
impact: fix
---

The self-test's hold on a model is not preemptible: with eviction grace off, a client whose load needs the memory the self-test holds is refused once (503) before the self-test sees the refusal count move and yields, so the client's retry finds room. The correct fix is pool-side: a soft, source-tagged hold that the eviction plan may take and that cancels the holder's context, so the client's Acquire evicts the self-test's model instead of being refused. Found by the security review of itd-2609100457007827; the counter (Residency.Refusals) is the interim.

## Grounds

- pursued: we expect a client's load to take the self-test's model and the run to end as yielded, because the hold now says it may be taken and the holder's cancellation is the run's own yield; wrong if a client's request is ever failed or made to wait longer by a preemption, or if a run comes back failed rather than yielded when the pool asks for its model.
