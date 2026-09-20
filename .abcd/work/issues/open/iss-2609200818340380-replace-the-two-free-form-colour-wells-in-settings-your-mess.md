---
schema_version: 1
id: "iss-2609200818340380"
slug: "replace-the-two-free-form-colour-wells-in-settings-your-mess"
severity: "minor"
category: "ux"
source: "manual-test"
found_during: "maintainer manual test of the chat client, 2026-09-20"
origin: researcher-authored
production_mode: hand-written
found_at: "client/DessauChat/Bubbles.swift"
---

Replace the two free-form colour wells in Settings (Your messages, The model's replies, each opening the full system colour panel) with the style of the system's accent-colour row: a single row of named swatches (the multicolour system choice, then blue, purple, pink, red, orange, yellow, green, graphite), the chosen swatch ringed with its name shown beneath, in a Theme section. One choice colours both bubbles: the chosen colour is the user's bubble and a lighter tint of the same colour is the model's bubble, so the pair always reads as one palette. The default is the multicolour swatch, meaning follow this Mac's own accent colour, which is what the user bubble already falls back to today; the model bubble's fallback changes from the secondary grey to the tint. The stored value becomes a swatch name rather than two hex strings; pre-1.0, existing stored hex values are dropped, not migrated. Keep the rule that a colour is appearance-aware so the tint is legible in both light and dark.
