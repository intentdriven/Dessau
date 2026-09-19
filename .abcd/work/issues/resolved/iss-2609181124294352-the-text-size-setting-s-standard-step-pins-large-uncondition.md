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
resolution: "TextSize.dynamicType is now optional and the standard step returns nil, so Default applies no Dynamic Type size at all; both scene roots apply it through a new .dynamicTypeSize(ifSet:) view extension that leaves the view untouched when no size is set, the shape Appearance's System case already had."
impact: fix
---

The text-size setting's standard step pins .large unconditionally rather than expressing the absence of a preference, unlike the Appearance enum's System case shipped in the same change, so a Mac whose system text size is not large is overridden by the client's default.

## Grounds

- pursued: we expect the default step to inherit the Mac's own text size because SwiftUI's environment value is only overridden when the modifier is applied; wrong if a scene root ever needs a size pinned to lay out correctly, or if some control reads the environment only when it has been set explicitly.
