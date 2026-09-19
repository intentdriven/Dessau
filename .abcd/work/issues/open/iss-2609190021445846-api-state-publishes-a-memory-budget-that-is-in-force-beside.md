---
schema_version: 1
id: "iss-2609190021445846"
slug: "api-state-publishes-a-memory-budget-that-is-in-force-beside"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "reading /api/state for a context-probe preflight fit check"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gateway/control.go"
---

/api/state publishes a memory budget that is in force beside a decode concurrency that may not be, so any fit arithmetic done from the snapshot can disagree with the pool. gateway.Control.snapshot fills machine.budget from Pool.MemoryBudget() (the value the pool was built with) but the only concurrency on the snapshot is config.decode_concurrency, the saved value; App.effectiveSettings states that the budget, the decode concurrency and the idle timeout are read once when the pool is built and a change takes a restart. App.chargeOf — the authoritative charge — uses Pool.DecodeConcurrency(). Between a concurrency save and a restart the panel's modelCharge() in internal/ui/static/app.js therefore computes a charge the pool does not agree with, and its comment at app.js:1251 claims the panel 'reads the figure the pool is running with rather than assuming one', which is the opposite of what state.config.decode_concurrency is. The snapshot should publish the concurrency in force beside the budget in force, or the panel and its comment should say which figure they are using.
