---
schema_version: 1
id: "iss-2609180608286062"
slug: "links-a-model-writes-open-with-any-url-scheme-the-markdown-r"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "adversarial review of feat/client-macos-27, 2026-09-17"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/Markdown.swift"
resolution: "The transcript's openURL action opens http and https only and discards every other scheme a model writes."
impact: fix
---

Links a model writes open with any URL scheme: the markdown renderer's .full syntax yields clickable links and no openURL handler restricts them, so a hostile server's reply can hand the Mac a file, shortcuts or settings URL on one click. The transcript should open http and https only.
