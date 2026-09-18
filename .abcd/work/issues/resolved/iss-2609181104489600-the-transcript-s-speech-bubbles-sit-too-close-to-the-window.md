---
schema_version: 1
id: "iss-2609181104489600"
slug: "the-transcript-s-speech-bubbles-sit-too-close-to-the-window"
severity: "nitpick"
category: "observation"
source: "user-observation"
found_during: "the maintainer's manual test of the 0.7.0 client, 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/GropiusChat.swift"
resolution: "The transcript carries 28 points of horizontal padding and 12 vertical, so a bubble never touches an edge."
impact: fix
---

The transcript's speech bubbles sit too close to the window's left and right edges; add more padding on both sides so a bubble never touches an edge (the maintainer's manual test, 2026-09-18).

## Grounds

- pursued: we expect the wider margin to read as Messages' transcript; wrong if the maintainer finds the bubbles now too narrow
