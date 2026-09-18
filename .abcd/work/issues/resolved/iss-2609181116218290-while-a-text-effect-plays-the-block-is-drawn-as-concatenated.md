---
schema_version: 1
id: "iss-2609181116218290"
slug: "while-a-text-effect-plays-the-block-is-drawn-as-concatenated"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "fidelity audits 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/Effects.swift"
resolution: "The animated block carries .textSelection(.enabled) alongside the renderer, so a reply is selectable while its words animate. TestChatClientEffectKeepsTheReplySelectable holds it."
impact: fix
---

While a text effect plays, the block is drawn as concatenated Texts carrying no textSelection, so the reply is unselectable for the effect's duration - the falsifier the intent's Grounds names.

## Grounds

- pursued: selection is the reply's whatever is drawn over it. Wrong if the renderer and the selection cannot both be active and the words stop animating.
