---
schema_version: 1
id: "iss-2609181124294352"
slug: "the-text-size-setting-s-standard-step-pins-large-uncondition"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "fidelity audits 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/GropiusChat.swift"
---

The text-size setting's standard step pins .large unconditionally rather than expressing the absence of a preference, unlike the Appearance enum's System case shipped in the same change, so a Mac whose system text size is not large is overridden by the client's default.
