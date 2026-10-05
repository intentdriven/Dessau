---
schema_version: 1
id: "iss-2610042030098334"
slug: "buildrequest-s-fallback-cuts-the-newest-user-turn-to-budget"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "independent review of fix/bridge-history-opens-on-user, 2026-10-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/bridge/discord/conversation.go"
remedy: "Trim the tail of the turn until the encoded body fits the budget, judged the way the gateway judges it (len(body)/4 + max_tokens)."
resolution: "buildRequest's fallback keeps the longest tail whose encoded body fits the budget, found by halving, instead of budget/4 runes"
impact: fix
resolved_by:
  commit: "3fe92633"
---

buildRequest's fallback cuts the newest user turn to budget/4 runes, not to the budget in encoded bytes, so a turn of four-byte emoji, or of characters json.Marshal escapes to six bytes (<, >, &, control characters), still sends a body the gateway judges over the window. At a served window of 4096 with a 16 KiB message the body is judged at 5653 tokens with &lt; repeated and 4117 with emoji, and the gateway refuses it.
