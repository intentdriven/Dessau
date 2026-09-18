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
---

While a text effect plays, the block is drawn as concatenated Texts carrying no textSelection, so the reply is unselectable for the effect's duration - the falsifier the intent's Grounds names.
