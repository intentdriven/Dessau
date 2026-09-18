---
schema_version: 1
id: "iss-2609180608284326"
slug: "the-kept-client-release-subtest-runs-git-tag-list-which-retu"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "adversarial review of feat/client-macos-27, 2026-09-17"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/archtest/minimum_macos_test.go"
resolution: "The git tag assertion is dropped; the subtest holds the installer's tag to the README's."
impact: fix
---

The kept-client-release subtest runs git tag --list, which returns nothing in CI's shallow checkout, so it fails on every CI run; the README and install.sh parity is the check that can hold there.
