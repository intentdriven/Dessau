---
schema_version: 1
id: "iss-2609180608296549"
slug: "the-shimmer-leaves-the-run-s-late-glyphs-dimmed-when-progres"
severity: "nitpick"
category: "bug"
source: "user-observation"
found_during: "adversarial review of feat/client-macos-27, 2026-09-17"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/Effects.swift"
resolution: "The shimmer band clears the run before progress reaches 1 and opacity is pinned to full from 0.95."
impact: fix
---

The shimmer leaves the run's late glyphs dimmed when progress reaches 1, and they pop to full when the renderer is removed; the band must clear the run before the end.
