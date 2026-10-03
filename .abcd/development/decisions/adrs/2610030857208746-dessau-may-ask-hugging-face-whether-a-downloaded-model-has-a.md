---
id: adr-2610030857208746
slug: dessau-may-ask-hugging-face-whether-a-downloaded-model-has-a
status: proposed
date: 2026-10-03
supersedes: null
superseded_by: null
related_intents: [itd-2610030857275099]
related_rfcs: []
related_adrs: [adr-2609201008476813]
---

# ADR-2610030857208746: Dessau may ask Hugging Face whether a downloaded model has a newer version

## Context

The maintainer asked on 2026-10-03 for Dessau to check downloaded models for
newer versions at startup and at an interval set in Settings
(itd-2610030857275099). The accepted
[adr-2609201008476813](2609201008476813-local-telemetry-may-record-prompt-text-and-completions-only.md)
fixes Dessau's outbound connections as "exactly those needed to fetch models
and to provision its runtime", and says nothing about models leaves the
machine to any third party. A check the operator did not ask for at that moment
is a new kind of connection, and each one tells Hugging Face which repositories
this Mac holds, from this address, on a schedule. The earlier research behind
adr-2609061503319212 found that users of local-LLM tools treat even an update
check as suspect.

Constraints already locked: no public telemetry, ever; nothing reported to the
project or a vendor; HF_HUB_OFFLINE on every model-server child (the check is
Dessau's own, never a child's); decision models stay on their reviewed revision
(itd-2610030656210408). Today the hub downloads `main` and records no revision,
so a check has nothing to compare against until the downloaded revision is
recorded.

If accepted, this record supersedes adr-2609201008476813 in whole, restating it
with the outbound-connections clause widened by exactly this check; the old
record then changes only its status fields.

## Decision

_Open. Decided by the maintainer at the planning interview of
itd-2610030857275099; until then this record is `proposed` and
adr-2609201008476813 stands unchanged._

## Alternatives Considered

Laid out for the maintainer, none chosen yet:

1. **Opt-in check.** Off until the operator sets an interval; when on, one
   metadata request per downloaded repository (its current revision), with no
   identifier, cookie or token beyond what a download already sends. Nothing
   leaves the Mac by default.
2. **On by default, with a switch.** Checks run at startup and on a default
   interval until the operator turns them off. Most useful; contradicts the
   recorded "users treat an update check as suspect" finding.
3. **Operator-triggered only.** No scheduled check; a "Check now" action in the
   panel. Keeps the existing clause intact (the operator asks), but does not
   deliver "at startup and at intervals".
4. **Declined.** No update checks; the operator re-downloads by hand.

## Consequences

_Written with the decision._
