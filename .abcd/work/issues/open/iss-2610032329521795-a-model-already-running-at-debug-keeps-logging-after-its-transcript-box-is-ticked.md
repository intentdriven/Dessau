---
schema_version: 1
id: "iss-2610032329521795"
slug: "a-model-already-running-at-debug-keeps-logging-after-its-transcript-box-is-ticked"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "re-review of the no-transcript disarm (PR 178)"
origin: researcher-authored
production_mode: hand-written
found_at: "docs/transcript.md"
remedy: "Say it on the transcript page and in the panel hint (unload the model to stop a run already at debug), or have the save name or unload resident models still at debug that it marks."
---

A model already running at the debug level keeps writing every prompt and answer to its log after its transcript box is ticked: the launch spent the debug mark, so the save's disarm finds nothing to take off the list, and the run logs until it next stops. docs/transcript.md and the panel's Transcript hint say ticking the box takes the model off the debug-logging list and say nothing of the run already at debug; the card shows the debug state, which limits the harm.
