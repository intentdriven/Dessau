---
schema_version: 1
id: "iss-2609181124294842"
slug: "the-text-size-picker-is-delivered-inside-a-combined-appearan"
severity: "nitpick"
category: "observation"
source: "agent-finding"
found_during: "fidelity audits 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/GropiusChat.swift"
resolution: "The text-size picker now sits in its own Section(\"Text\") in Settings, as the spec named, leaving the Appearance section to the light/dark choice alone; an archtest reads the Text section's body for the picker."
impact: internal
---

The text-size picker is delivered inside a combined Appearance section in Settings rather than the spec's own Text section.

## Grounds

- pursued: we expect the spec's section layout to be the one shipped because the two settings scale different things and the spec named a Text section; wrong if the maintainer prefers one combined section on the Settings window's width.
