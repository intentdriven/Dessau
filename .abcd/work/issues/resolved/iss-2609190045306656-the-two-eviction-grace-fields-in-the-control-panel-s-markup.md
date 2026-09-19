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
resolution: "The two literals are gone from internal/ui/static/index.html; renderGraceDefaults in app.js fills both placeholders from the snapshot's defaults object every time the settings pane is drawn, and offers no figure at all when the snapshot carries none. TestTheGracePlaceholdersAreFilledFromTheServersDefaults holds the markup to carrying no placeholder, the renderer to the server's two constants, and renderSettings to calling it."
impact: internal
---

The two eviction-grace fields in the control panel's markup carry the server's defaults as placeholder text: internal/ui/static/index.html has placeholder="120" on setGraceSec and placeholder="300" on setGraceWait, which are config.DefaultEvictionGraceSec and config.DefaultEvictionMaxWaitSec written out a third time. Nothing binds them, so a change to either Go constant leaves the blank field showing a figure the server no longer resolves it to -- the same drift as the hint beside them, which is now bound to the snapshot's defaults object. It should be that the two placeholders are filled from state.defaults when the settings form is drawn, or tested against the Go constants the way the hint is.

## Grounds

- pursued: we expect the panel to state only figures the server sent, because a default written into the markup drifts the moment a Go constant changes; wrong if the placeholders are needed before the first snapshot arrives, which would leave the fields silent on a panel that has not connected.
