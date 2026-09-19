---
schema_version: 1
id: "iss-2609181124295668"
slug: "the-user-bubble-s-text-is-a-fixed-color-white-and-a-picked-b"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "fidelity audits 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/Bubbles.swift"
resolution: "The bubble defaults are semantic system colours (the accent colour and the system's secondary fill) that follow Light and Dark by themselves, a picked colour is wrapped in a dynamic system colour with a face per appearance, and the text on a bubble is the system's label colour read under the appearance its fill resolves as, so no fixed Color.white remains."
impact: fix
---

The user bubble's text is a fixed Color.white and a picked bubble colour is a stored literal hex, so neither follows the chosen appearance - against the promise that every colour the client uses is the system's.

## Grounds

- pursued: we expect a bubble to read in both appearances because every colour it draws is either semantic or given a light and a dark face; wrong if a person picking a mid-grey wants it to flip the way the system's own grey does, which one stored value cannot express
