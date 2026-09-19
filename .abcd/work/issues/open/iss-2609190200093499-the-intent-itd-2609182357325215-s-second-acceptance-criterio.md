---
schema_version: 1
id: "iss-2609190200093499"
slug: "the-intent-itd-2609182357325215-s-second-acceptance-criterio"
severity: "minor"
category: "process"
source: "impl-review"
found_during: "fidelity audit of itd-2609182357325215"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-2609182357325215-bob-s-chat-client-pairs-with-alice-s-gropius-server-once-and.md"
---

The intent itd-2609182357325215's second acceptance criterion promises that Bob pairs from the iPad client and that the hand check is done on the maintainer's own device and recorded as such, because the simulator cannot answer for it. No such check exists: the shipping decision line in .abcd/work/DECISIONS.md and the spec's Verification section both state a real iPad as owed and not verified, and only a simulator build is recorded. The fidelity audit scores the criterion NOT MET on the record's own words. It should be checked on a real iPad and the outcome recorded in the spec's Verification, or the criterion should be narrowed by a decision that says the iPad is not claimed.

## Triage 2026-09-19

This one waits on the maintainer's hand: it needs a real iPad, which this
project owns none of and which no simulator answers for. An agent can neither
close it nor evidence it. What it needs from the maintainer is one of two
things — the check done on their device and its outcome written into the spec's
Verification and a dated decision line, or a decision that the iPad is not
claimed for this release, which narrows the criterion rather than leaving it
unmet.
