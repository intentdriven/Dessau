---
schema_version: 1
id: "iss-2610042101430192"
slug: "a-staged-update-removes-the-old-copy-of-a-model-before-the"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "fidelity review rcp-c60839654fc3 of itd-2610030857275099, 2026-10-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/app/app.go"
remedy: "Keep the old copy aside until the new version has been loaded and answered once (or the operator confirms), and restore it automatically when the first load fails, with a test where the new version passes the directory checks but fails to load."
resolution: "A staged update now holds the old copy aside until the new version has loaded and answered once, and puts it back automatically when that first load fails (restoreFallback, held across restarts)."
impact: fix
---

A staged update removes the old copy of a model before the new version has loaded successfully. The spec for itd-2610030857275099 says the old copy is removed only after the new one 'passes readiness'. DECISIONS.md reads readiness as the directory checks, so internal/app/app.go (around line 1759) deletes the old copy as soon as the new record is written. A new version that passes those checks but fails at its first load leaves nothing to fall back to: the model is unusable until the operator downloads an older version by hand.
