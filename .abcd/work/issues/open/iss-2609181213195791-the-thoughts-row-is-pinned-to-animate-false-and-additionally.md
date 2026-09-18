---
schema_version: 1
id: "iss-2609181213195791"
slug: "the-thoughts-row-is-pinned-to-animate-false-and-additionally"
severity: "nitpick"
category: "observation"
source: "agent-finding"
found_during: "fidelity audits 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/GropiusChat.swift"
---

The Thoughts row is pinned to animate false and additionally styled callout, italic and secondary, so the promise that thoughts are drawn the way the reply below them is holds for the parse and the blocks but not for the drawing; an emphasised word renders italic inside italic text.
