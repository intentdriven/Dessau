---
schema_version: 1
id: "iss-2609190031063965"
slug: "startlocked-asks-preemptible-holders-to-let-go-whenever-the"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "adversarial read of the soft-hold diff for iss-2609100526194406"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/runtime/pool.go"
resolution: "startLocked asks a preemptible holder to let go only when the plan that names it is enough, so a hold is taken only when taking it seats the load. Covered by TestAHoldIsNotTakenWhenTakingItWouldStillNotFit, watched to fail first."
impact: internal
---

startLocked asks preemptible holders to let go whenever the eviction plan names any, without first checking that the plan is enough. A plan can name every preemptible hold and still not fit — one ordinary client holding a second model in flight is all it takes — and the load is then refused anyway, having cancelled a self-test run that nobody could be served by. The ask belongs behind the plan's own enough flag: a hold is taken only when taking it seats the load.

## Grounds

- pursued: we expect a run to survive a load it could not have seated, because the ask is behind the plan's enough flag; wrong if a client is refused where a preempt would have served it.
