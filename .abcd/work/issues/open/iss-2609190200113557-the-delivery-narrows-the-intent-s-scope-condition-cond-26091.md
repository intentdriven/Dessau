---
schema_version: 1
id: "iss-2609190200113557"
slug: "the-delivery-narrows-the-intent-s-scope-condition-cond-26091"
severity: "nitpick"
category: "observation"
source: "impl-review"
found_during: "fidelity audit of itd-2609182357325215"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-2609182357325215-bob-s-chat-client-pairs-with-alice-s-gropius-server-once-and.md"
---

The delivery narrows the intent's scope condition cond-2609190031322606, which says pairing is first-come and that anything on the network reaching the pairing endpoint pairs itself. internal/gateway/pairing.go now refuses any request carrying an Origin header and any content type but application/json, so a web page anybody on the network visits cannot enrol a key — the deliberate fix for iss-2609190110118690. The narrowing is right and it is recorded in the intent's Audit Notes; what is stale is the scope condition's own wording, which a later reader would take as the rule. Noted rather than actioned: nothing here needs a code change.
