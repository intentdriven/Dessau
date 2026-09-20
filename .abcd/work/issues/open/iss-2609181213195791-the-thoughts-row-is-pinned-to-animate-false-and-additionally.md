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

## Deferral 2026-09-19

Waits on the maintainer: the Thoughts row's italic, secondary, callout styling was their manual-test ask on 2026-09-18; dropping the italic to fix italic-in-italic is a look they should choose. Nitpick.

## Decision 2026-09-20

Maintainer's decision at interview: drop the italic; keep secondary and callout. Recorded in `.abcd/work/DECISIONS.md`.
