---
schema_version: 1
id: "iss-2609181124296430"
slug: "both-the-text-size-and-the-appearance-intents-promise-the-ma"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "fidelity audits 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/build.sh"
---

Both the text-size and the appearance intents promise the Mac and the iPad clients alike, but the tree carries no iPad client: build.sh compiles one macOS-only target and the sole UIKit branch is a hex-conversion fallback.
