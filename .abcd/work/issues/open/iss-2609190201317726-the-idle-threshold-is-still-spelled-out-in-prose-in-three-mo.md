---
schema_version: 1
id: "iss-2609190201317726"
slug: "the-idle-threshold-is-still-spelled-out-in-prose-in-three-mo"
severity: "nitpick"
category: "drift"
source: "agent-finding"
found_during: "binding the idle-threshold placeholder and its prose to the server's default"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/ui/static/app.js"
---

The idle threshold is still spelled out in prose in three more places, all of them copies of config.DefaultIdleThresholdSec: internal/ui/static/index.html's Self-test hint says a run starts 'whenever nothing has asked this Mac for a model for five minutes', and internal/ui/static/app.js says the same thing twice in status lines (the self-test status around line 883 and the context-probe status around lines 1834-1835). The snapshot now carries idle_threshold_sec and the Context probe field's placeholder and prose are rendered from it, so these three are what is left of the drift: a Mac serving a different threshold tells the operator five minutes anyway. They are status prose rather than form hints, so they need the figure threaded into the renderers that write them.
