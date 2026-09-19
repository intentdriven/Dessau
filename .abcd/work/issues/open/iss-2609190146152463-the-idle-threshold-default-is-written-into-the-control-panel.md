---
schema_version: 1
id: "iss-2609190146152463"
slug: "the-idle-threshold-default-is-written-into-the-control-panel"
severity: "nitpick"
category: "drift"
source: "agent-finding"
found_during: "binding the eviction-grace placeholders to the server's defaults"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/ui/static/index.html"
---

The idle-threshold default is written into the control panel twice over: internal/ui/static/index.html carries placeholder="300" on #setIdleThreshold, and the hint below it says "Blank is five minutes" in prose. Both are copies of config.DefaultIdleThresholdSec, bound by nothing, so a change to the Go constant leaves the Settings page stating a figure the server has stopped using. This is the same drift the eviction-grace pair was bound out of (iss-2609190029273153, iss-2609190045306656), but it needs the snapshot's Defaults struct in internal/gateway/control.go to carry idle_threshold_sec before the panel can fill the placeholder from it. The hint's prose needs a decision of its own: either it names the figure from the snapshot too, or it stops naming one.
