---
schema_version: 1
id: "iss-2610031239271873"
slug: "dropped-hash-reads-as-changed-at-every-check"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "adversarial review of spc-2610030929021692 step 1"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/registry/registry.go"
remedy: "In spc-2610030929021692 step 2, compare only the paths that were recorded, read an unrecorded path as unknown rather than changed, and test a version with a dropped entry."
resolution: "The update check compares only the recorded paths for change, and reads a listed file the record has no hash for as added only when it is absent from the model directory, so a dropped hash never marks a model (TestAnUnrecordedFileIsAddedOnlyWhenItIsNotOnDisk)."
impact: internal
resolved_by:
  commit: "5edd2a1"
---

A file whose recorded hash was dropped would read as changed at every update check. sanitizeVersion in internal/registry drops a hash entry whose name is outside the plain path alphabet or past the length bound, and a listing entry with an LFS object of size 0 keeps the pointer's blob id; a check that compares the Hub's full listing with the recorded hashes would then report such a file as added or changed every time, a permanent false mark. Found by the adversarial review of spc-2610030929021692 step 1 on 2026-10-03.
