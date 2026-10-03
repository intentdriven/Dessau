---
schema_version: 1
id: "iss-2610032221548858"
slug: "debug-arm-races-transcript-mark"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "adversarial review of the fix for iss-2610032212269524"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gateway/control.go"
remedy: "Take the settings lock around the arming handler's check of the mark and its ArmDebugLog, so a save and an arm are ordered."
resolution: "The arming handler takes the settings lock around its check of the mark and its ArmDebugLog."
impact: fix
resolved_by:
  commit: "16947c3b7b4337a0dc11e3a717a6a2cf45e4d5ea"
---

The debug-log arming handler reads the transcript mark and then arms without the settings lock: a save that marks the model can run between the two, find nothing armed, and leave the arm to land after it, so a marked model is armed for debug and writes every prompt and answer to its log.
