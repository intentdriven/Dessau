---
schema_version: 1
id: "iss-2609181116217704"
slug: "the-text-effect-latches-whether-to-animate-at-the-reply-bloc"
severity: "major"
category: "observation"
source: "agent-finding"
found_during: "fidelity audits 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/GropiusChat.swift"
---

The text effect latches whether to animate at the reply block's first appearance - the first streamed token - while the message id is queued in the stream's defer, so a finishing reply does not animate and the stale id can instead fire on a row recreated by scroll; the promised played flag keyed by message id is half on the row's own state.
