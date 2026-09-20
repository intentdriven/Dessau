---
schema_version: 1
id: "iss-2609181116215521"
slug: "the-gate-deciding-whether-a-reply-animates-matches-effect-wo"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "fidelity audits 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/DessauChat/Effects.swift"
---

The gate deciding whether a reply animates matches effect words over the whole reply's raw markdown text while the renderer matches per rendered block over the block's plain text, so the two matches can disagree.

## Triage 2026-09-18

Left open for the maintainer: making the gate agree with the renderer means matching over the parsed blocks, which puts a markdown parse in AppModel's stream path — a placement decision. Today the disagreement is one-sided and harmless: the gate is broader, so it can queue an id no block animates.

## Deferral 2026-09-19

Waits on the maintainer: matching the animation gate over parsed blocks puts a markdown parse in AppModel's stream path, a placement decision; the disagreement is one-sided and harmless today.
