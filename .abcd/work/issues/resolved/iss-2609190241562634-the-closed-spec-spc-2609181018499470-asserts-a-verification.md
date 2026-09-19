---
schema_version: 1
id: "iss-2609190241562634"
slug: "the-closed-spec-spc-2609181018499470-asserts-a-verification"
severity: "minor"
category: "drift"
source: "impl-review"
found_during: "fidelity audit of itd-2609180959397172"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/specs/closed/spc-2609181018499470-bob-messages-gropius-from-discord-alice-pastes-a-discord-bot.md"
resolution: "The closed spec is left as it stands, as a ratified record is, and the correction is a dated line appended to .abcd/work/DECISIONS.md on 2026-09-19: it quotes the spec's two sentences, sets them against the shipping decision line that records all six live checks as owed, and states what is true — nothing of the bridge has been exercised against Discord itself and the live checks remain owed."
impact: internal
---

The closed spec spc-2609181018499470 asserts a verification that did not happen. Its acceptance-criteria map says the resume across sleep was 'checked by hand with a sleeping Mac and recorded in the shipping decision line', and its Verification section says 'the live checks (a real bot, a mention, a long answer, a sleep) by hand on the maintainer's Mac, recorded in the shipping decision line'. The decision line it points at, .abcd/work/DECISIONS.md:301, records all six live checks as OWED to the maintainer's own Mac and never claimed as done. The decision line is the honest one; the spec was closed carrying a claim its own citation refutes. The spec's two sentences should be corrected to say the live checks are owed, so that a later reader who trusts the spec is not told the bridge was checked against real Discord.

## Grounds

- pursued: we expect a dated correction in the decisions ledger to reach a later reader because the ledger is where this repository records a claim being narrowed; wrong if the maintainer decides a closed spec carrying a refuted sentence should instead be reopened and amended.
