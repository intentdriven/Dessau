---
schema_version: 1
id: "iss-2609190045306656"
slug: "the-two-eviction-grace-fields-in-the-control-panel-s-markup"
severity: "nitpick"
category: "drift"
source: "agent-finding"
found_during: "binding the panel's as-you-type hints to the Go rules they restate (iss-2609190029273153)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/ui/static/index.html"
---

The two eviction-grace fields in the control panel's markup carry the server's defaults as placeholder text: internal/ui/static/index.html has placeholder="120" on setGraceSec and placeholder="300" on setGraceWait, which are config.DefaultEvictionGraceSec and config.DefaultEvictionMaxWaitSec written out a third time. Nothing binds them, so a change to either Go constant leaves the blank field showing a figure the server no longer resolves it to -- the same drift as the hint beside them, which is now bound to the snapshot's defaults object. It should be that the two placeholders are filled from state.defaults when the settings form is drawn, or tested against the Go constants the way the hint is.
