---
schema_version: 1
id: "iss-2609180608283729"
slug: "install-sh-s-download-failure-still-says-from-the-latest-rel"
severity: "nitpick"
category: "bug"
source: "user-observation"
found_during: "adversarial review of feat/client-macos-27, 2026-09-17"
origin: researcher-authored
production_mode: hand-written
found_at: "install.sh"
resolution: "The failure names releases/$RELEASE_PATH, and the kept-release message is printed only when the assets come from the forge."
impact: fix
---

install.sh's download failure still says 'from the latest release' on the kept-release path, and the kept-release message prints in the release workflow's gate, which installs from a local directory; both mislead.
