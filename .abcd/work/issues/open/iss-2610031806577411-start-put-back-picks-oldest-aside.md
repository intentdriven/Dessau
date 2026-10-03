---
schema_version: 1
id: "iss-2610031806577411"
slug: "start-put-back-picks-oldest-aside"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "adversarial review of fbf4fa6 on the fix for iss-2610031324593822"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/app/staged.go"
remedy: "Share asideLeft's newest-by-attempt selection with putBackAside, and test it at start."
---

A start puts back whichever aside copy the unsorted directory listing returns first, not the newest: with the model's folder missing and two asides left, an older version can be restored and, now that the clear removes an aside beside a model folder that checks out, the newer one removed. asideLeft (the swap path) already picks the newest.
