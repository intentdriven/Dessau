---
schema_version: 1
id: "iss-2609180608298680"
slug: "the-release-workflow-builds-the-client-on-github-s-macos-26"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "adversarial review of feat/client-macos-27, 2026-09-17"
origin: researcher-authored
production_mode: hand-written
found_at: ".github/workflows/release.yml"
resolution: "The release build job runs on the xcode-27 preview runner and a loud step selects Xcode 27 or fails."
impact: internal
---

The release workflow builds the client on GitHub's macos-26 image, which carries Xcode 26.0 to 26.6 and no Xcode 27, so the macOS 27 client cannot be released from it; GitHub's public-preview runner label xcode-27 (macOS 27.0 beta, Xcode 27 beta 6) is the one image that can. The maintainer chose to move the build job there (2026-09-17).
