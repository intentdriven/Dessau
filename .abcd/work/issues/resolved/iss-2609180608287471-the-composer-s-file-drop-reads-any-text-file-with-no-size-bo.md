---
schema_version: 1
id: "iss-2609180608287471"
slug: "the-composer-s-file-drop-reads-any-text-file-with-no-size-bo"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "adversarial review of feat/client-macos-27, 2026-09-17"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/GropiusChat.swift"
resolution: "The drop reads a regular text file of at most 1 MiB, checked through resourceValues before the read."
impact: fix
---

The composer's file drop reads any text file with no size bound and follows symlinks, so a dropped multi-gigabyte file hangs the app; a regular-file check and a cap belong before the read.
