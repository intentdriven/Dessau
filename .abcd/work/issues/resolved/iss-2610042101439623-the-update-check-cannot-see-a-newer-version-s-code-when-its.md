---
schema_version: 1
id: "iss-2610042101439623"
slug: "the-update-check-cannot-see-a-newer-version-s-code-when-its"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "fidelity review rcp-c60839654fc3 of itd-2610030857275099, 2026-10-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/app/updatecheck.go"
remedy: "When config.json is an LFS pointer, fetch its real content through the resolve endpoint the download already uses (or mark the version 'cannot check'), so a version that ships its own code is never offered Update."
resolution: "The update check reads an LFS config.json from the content CDN, held to the listed sha256 with no token sent there, and marks a version whose config cannot be held to it cannot_check, never offered"
impact: fix
---

The update check cannot see a newer version's code when its config.json is stored in Git LFS. internal/app/updatecheck.go (around line 238) reads config.json to decide whether a newer version names code of its own. When that file is an LFS pointer the check cannot read it, so the version is marked 'available' with Update offered. Only the update's own launch checks refuse it later, after a download.
