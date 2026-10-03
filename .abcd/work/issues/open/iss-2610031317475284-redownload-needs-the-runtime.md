---
schema_version: 1
id: "iss-2610031317475284"
slug: "redownload-needs-the-runtime"
severity: "minor"
category: "ux"
source: "impl-review"
found_during: "adversarial review of spc-2610030929021692 step 3"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/app/staged.go"
remedy: "Decide whether a staged version is held to the whole launch Precheck (as the spec says) or only to its model half (registry.CheckModelCode) when the runtime is absent; the spec's wording is the maintainer's to change."
---

Downloading a model again fails until the Python runtime is installed. A staged version must pass the launcher's Precheck, which checks the managed interpreter before the model, so every re-download or Update of a ready model fails while the runtime is still being set up, where a re-download needed no runtime before. Found by the adversarial review of spc-2610030929021692 step 3 on 2026-10-03.
