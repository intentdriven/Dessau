---
schema_version: 1
id: "iss-2610031807030740"
slug: "start-validates-looser-than-rescan"
severity: "nitpick"
category: "bug"
source: "agent-finding"
found_during: "adversarial review of fbf4fa6 on the fix for iss-2610031324593822"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/app/fsutil.go"
remedy: "Decide staleness with the rescan's own check (inspectModelDir), or make validateModelDir refuse what inspectModelDir refuses."
---

The start decides an aside is stale when its model's folder passes validateModelDir, which is looser than the registry's inspectModelDir (that also refuses a .dessau-part file and weights through links): a model folder the rescan would refuse could make the start remove its aside copy. Practically unreachable — a staged copy is complete before the swap — but the two checks differ.
