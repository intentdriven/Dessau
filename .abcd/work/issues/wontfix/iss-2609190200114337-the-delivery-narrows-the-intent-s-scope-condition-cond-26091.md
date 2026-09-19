---
schema_version: 1
id: "iss-2609190200114337"
slug: "the-delivery-narrows-the-intent-s-scope-condition-cond-26091"
severity: "nitpick"
category: "observation"
source: "impl-review"
found_during: "fidelity audit of itd-2609182357325215"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-2609182357325215-bob-s-chat-client-pairs-with-alice-s-gropius-server-once-and.md"
wontfix_reason: "The narrowing is deliberate and recorded (it traces to resolved iss-2609190110118690, the CSRF finding of the security review); the scope condition's wording is qualified in the Audit Notes rather than the code widened back."
---

The delivery narrows the intent's scope condition cond-2609190031321181, which says the plain port and the shared API key are untouched. The bearer check, the loopback exemption and the refusal text are indeed unchanged and the replay proves the answers do not move, but the plain listener now carries one route it did not have: POST /pair, which asks for no credential and writes the settings file. The condition as written would tell a later reader that the plain listener's route set is fixed, and it is not. Noted rather than actioned: mounting pairing on the plain port is the design.

## Grounds

- declined: The narrowing is deliberate and recorded (it traces to resolved iss-2609190110118690, the CSRF finding of the security review); the scope condition's wording is qualified in the Audit Notes rather than the code widened back.
