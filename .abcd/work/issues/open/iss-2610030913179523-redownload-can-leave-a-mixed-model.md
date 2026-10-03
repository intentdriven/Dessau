---
schema_version: 1
id: "iss-2610030913179523"
slug: "redownload-can-leave-a-mixed-model"
severity: "major"
category: "bug"
source: "plan-review"
found_during: "adversarial design review of itd-2610030857275099"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/hub/download.go"
remedy: "Treat a file as complete only when its sha256 matches the manifest (or its git blob id for small files), download at one pinned commit, and remove files under the model directory that the target revision does not list."
---

Re-downloading a model whose repository changed upstream can leave a silently mixed model. The downloader treats a file as already complete when its size matches the manifest (internal/hub/download.go, the 'Already complete from a previous run?' branch), so a re-quantised weight or an edited config.json of the same size keeps the old bytes without a hash check, and files the new revision dropped are never removed, so leftover shards sit beside the new ones for the runtime to load. Re-downloading a known repository is allowed today (internal/app/app.go keeps AddedAt across a re-download), and every download resolves main, so a re-download after an upstream change is exactly when it happens. Verified 2026-10-03 by reading the code; surfaced by the adversarial design review of itd-2610030857275099.
