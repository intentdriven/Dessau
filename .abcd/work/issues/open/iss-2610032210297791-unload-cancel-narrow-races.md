---
schema_version: 1
id: "iss-2610032210297791"
slug: "unload-cancel-narrow-races"
severity: "nitpick"
category: "observation"
source: "agent-finding"
found_during: "adversarial review of the fix for iss-2610031818057157"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gateway/control.go"
remedy: "Leave as is unless it is seen; a tight guard would check the job's name inside Interrupt."
---

Two narrow windows between the panel's ProbeHolds check and its interrupt: a run that completes in between gets a stray incomplete mark on a fresh measurement (invisible on the card, cleared by the next measurement), and a self-test run that starts on the same model in between is preempted, which is a yield there, not a failure. Neither loses data.
