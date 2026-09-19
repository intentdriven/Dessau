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
resolution: "Discharged by the iPad client (itd-2609180943290800, shipped 2026-09-18 on PR 62): one source builds for both systems from client/build.sh and client/build-ipad.sh, so the text-size and appearance intents' promise of the Mac and the iPad alike is now true of the tree."
impact: internal
---

Both the text-size and the appearance intents promise the Mac and the iPad clients alike, but the tree carries no iPad client: build.sh compiles one macOS-only target and the sole UIKit branch is a hex-conversion fallback.

## Grounds

- pursued: we expect the iPad client's landing to make the two intents' promise true; wrong if either modifier is missing from the iPad build's scene roots
