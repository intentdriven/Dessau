---
schema_version: 1
id: "iss-2609190242131524"
slug: "the-intent-itd-2609180959397172-contradicts-itself-about-whe"
severity: "nitpick"
category: "inconsistency"
source: "impl-review"
found_during: "fidelity audit of itd-2609180959397172"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-2609180959397172-bob-messages-dessau-from-discord-alice-pastes-a-discord-bot.md"
---

The intent itd-2609180959397172 contradicts itself about where a bridged request's identifiers are recorded. Its eighth acceptance criterion says 'the log line and the statistics record carry the bridge's name, the channel and user identifiers, the model, the sizes and the timing'; its scope condition cond-2609181018498110 says the statistics record carries one new fixed class, the source, 'and nothing a platform supplied: the channel and user identifiers go to the log line only'. The two cannot both be satisfied. The delivered code followed the condition — internal/stats/stats.go carries only the fixed source class and the identifiers reach the log line alone — which is the right call under adr-2609181004167097's fourth decision, and the fidelity audit scored the criterion MET_WITH_CONCERNS for the divergence. The criterion's text is what is wrong and should be corrected to name the log line only, so a later reader is not told the store holds a platform's identifiers.

## Triage 2026-09-19

This one waits on the maintainer's hand: it amends the acceptance criteria of a
shipped intent that a ratified fidelity verdict (rcp-91c0608082bd) has already
cited positionally, so an edit moves the ground the verdict stands on. The
decision it needs is whether the criterion is corrected in place with the
verdict re-ingested, or left as the record of what was promised with the
divergence standing in the audit where it already is.
