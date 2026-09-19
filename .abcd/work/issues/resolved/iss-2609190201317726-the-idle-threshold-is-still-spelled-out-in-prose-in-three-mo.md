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
resolution: "The three status copies of the idle threshold are written from the snapshot: a span in the Self-test hint filled by renderIdleProse, the posture page's self-test line, and the Self-test view's hint (now its own selfTestHint function). All read the threshold in force — the setting where one is set, the served default otherwise — through a shared intervalWords, which blankIsSentence now uses too; a panel told neither figure says 'the idle threshold' rather than inventing one."
impact: internal
---

The idle threshold is still spelled out in prose in three more places, all of them copies of config.DefaultIdleThresholdSec: internal/ui/static/index.html's Self-test hint says a run starts 'whenever nothing has asked this Mac for a model for five minutes', and internal/ui/static/app.js says the same thing twice in status lines (the self-test status around line 883 and the context-probe status around lines 1834-1835). The snapshot now carries idle_threshold_sec and the Context probe field's placeholder and prose are rendered from it, so these three are what is left of the drift: a Mac serving a different threshold tells the operator five minutes anyway. They are status prose rather than form hints, so they need the figure threaded into the renderers that write them.

## Grounds

- pursued: we expect a Mac serving a non-default threshold to say so in every place the panel names it, because all three now read state.config.idle_threshold_sec falling back to state.defaults.idle_threshold_sec; wrong if the effective threshold the server applies ever diverges from config.EffectiveIdleThresholdSec, which the JS mirrors by hand.
