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
resolution: "MessageRow latches the effect on the change of model.effectsToPlay containing this message's id — the moment the stream's defer queues it, which is the moment the reply finishes — and a row that finds the id already there consumes it without playing, so no row recreated by scroll can fire on it. TestChatClientEffectPlaysOnceWhenTheReplyFinishes holds it."
impact: fix
---

The text effect latches whether to animate at the reply block's first appearance - the first streamed token - while the message id is queued in the stream's defer, so a finishing reply does not animate and the stale id can instead fire on a row recreated by scroll; the promised played flag keyed by message id is half on the row's own state.

## Grounds

- pursued: the effect plays exactly once, when the reply finishes, on the row that was showing it; a recreated row draws plain text. Wrong if a reply animates twice, or a scrolled-away reply animates on its return.
