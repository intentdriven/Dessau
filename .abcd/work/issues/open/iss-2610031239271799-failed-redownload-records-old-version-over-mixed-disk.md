---
schema_version: 1
id: "iss-2610031239271799"
slug: "failed-redownload-records-old-version-over-mixed-disk"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "adversarial review of spc-2610030929021692 step 1"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/app/app.go"
remedy: "Resolved by spc-2610030929021692 step 3: the re-download stages the new version beside the old and swaps it in only when complete, so a failed attempt never touches the served directory."
---

A failed re-download records the old version over a directory that may already be partly the new one. restoreReady in internal/app/app.go puts back the prior commit and file hashes, but files the attempt finished and renamed into place before it failed are already at the new commit; a later check sees a difference and marks the model, which errs the safe way, but the record does not describe the disk. Found by the adversarial review of spc-2610030929021692 step 1 on 2026-10-03.
