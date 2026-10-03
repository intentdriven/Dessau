---
schema_version: 1
id: "iss-2610032210290992"
slug: "cancel-before-run-keeps-bounds"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "adversarial review of the fix for iss-2610031818057157"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/contextprobe/probe.go"
remedy: "Have Cancel leave a mark that the interrupted run honours: a run ending in a yield for a cancelled model drops its bounds; Measure now clears the mark."
---

An operator's Unload can land after the loop has marked the probe's run as current but before Run has made the model's bounds: Cancel finds nothing to drop, the interrupted Run then makes fresh bounds and yields keeping them, and a later Measure now resumes from them although the measurement was cancelled.
