---
schema_version: 1
id: "iss-2610071200183905"
slug: "a-403-from-the-content-cdn-during-the-update-check-s-config"
severity: "nitpick"
category: "ux"
source: "review-followup"
found_during: "security review of the LFS update-check fix (iss-2610042101439623), 2026-10-07"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/hub/hub.go"
remedy: "Word SmallFileAt's non-200 error by which host answered: name the content CDN and drop the token advice when the refusal came from off the Hub's origin."
---

A 403 from the content CDN during the update check's config fetch is worded as Hugging Face refusing access with the access token in Settings, though the CDN was never sent the token. The update check counts it as unreachable and never shows or logs the text, so no one sees it today; it would mislead anyone who surfaced it.
