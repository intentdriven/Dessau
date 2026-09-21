---
schema_version: 1
id: "iss-2609211334576018"
slug: "the-503-a-client-gets-while-the-budget-is-held-by-idle-work"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "live server v0.9.1 on 2026-09-21, 503 for every chat request"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gateway"
resolution: "The gateway is handed the idle loop's status and a no-room refusal to an entitled client names the model an idle job holds, the job, and for how long (TestARefusalNamesTheModelAnIdleJobIsHolding); the card's pill names the job (TestTheResidencyPillNamesTheJobHoldingTheModel)"
impact: fix
resolved_by:
  commit: "c9f75c6b"
---

The 503 a client gets while the budget is held by idle work reads 'not enough memory to load another model, and no model in memory can be freed (limit 76.8 GB)': true, but it names neither what holds the memory (a loading model under the context probe) nor that the holder is idle work the server started itself, so the person reads it as their model being too big. The refusal should name the holder and the job, and the panel should show the same (the card says only 'loading').
