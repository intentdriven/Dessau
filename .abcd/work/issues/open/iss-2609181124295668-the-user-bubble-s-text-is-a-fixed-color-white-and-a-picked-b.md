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
---

The user bubble's text is a fixed Color.white and a picked bubble colour is a stored literal hex, so neither follows the chosen appearance - against the promise that every colour the client uses is the system's.
