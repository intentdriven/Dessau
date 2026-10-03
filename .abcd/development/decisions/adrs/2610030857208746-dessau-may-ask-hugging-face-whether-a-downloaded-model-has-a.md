---
id: adr-2610030857208746
slug: dessau-may-ask-hugging-face-whether-a-downloaded-model-has-a
status: proposed
date: 2026-10-03
supersedes: null
superseded_by: null
related_intents: [itd-2610030857275099]
related_rfcs: []
related_adrs: [adr-2609201008476813, adr-2609061503319212, adr-2609181004167097]
---

# ADR-2610030857208746: Dessau may ask Hugging Face whether a downloaded model has a newer version

## Context

The maintainer asked on 2026-10-03 for Dessau to check downloaded models for
newer versions at startup and at an interval set in Settings
(itd-2610030857275099). Three records bear on whether, and how, Dessau may.

1. The accepted
   [adr-2609201008476813](2609201008476813-local-telemetry-may-record-prompt-text-and-completions-only.md)
   says: "Nothing about usage, hardware, errors, models or configuration
   leaves the machine to the project, to a vendor or to any third party. This
   covers crash reports, update checks that carry metadata, …" and "The
   outbound connections Gropius makes are exactly those needed to fetch models
   and to provision its runtime". The research note behind its predecessor
   (`2026-09-06-anonymous-telemetry-sota`) lists model repository ids under
   "must never" because they reveal interests. A per-repository check sent to
   HuggingFace is about models and goes to a third party, so it touches both
   sentences, not one.
2. The 2026-09-08 decision line on the release check: "A release check is
   opt-in and off by default, rather than an amendment to the
   no-public-telemetry decision … panel-gating changes when, not whether. Off by
   default keeps the promise literally true for anyone who never turns it on."
   On that precedent an opt-in, off-by-default check needs no supersession; only
   a check that runs by default does.
3. A check of this kind already runs. Since 2026-09-21 Dessau asks the Hub, at
   every start and with no setting, for the category of each ready model that
   has none (`internal/app/category.go`; the 2026-09-21 decision line on
   adopted chat models). That request already tells HuggingFace which
   repositories this Mac holds. Whether it fell under "needed to fetch models"
   or was a reversal nobody recorded is for the maintainer to say; this record
   names it either way, and an update check extends that job rather than adding
   a second Hub caller.

Two facts shape every option. Each Hub request carries the operator's
HuggingFace token whenever one is set (`internal/hub/hub.go`), which ties the
model list to their account; a repository's current commit is readable without
a token even for gated repositories. Requests also carry a fixed `dessau/1.0`
User-Agent.

Constraints already locked: no public telemetry, ever; nothing reported to the
project or a vendor; HF_HUB_OFFLINE on every model-server child (the check is
Dessau's own, never a child's); decision models stay on their reviewed revision
(itd-2610030656210408). If this record ends by superseding
adr-2609201008476813, it does so in part, naming the sentences it widens, as
adr-2609061503319212 was narrowed in part; the ADR's other decisions (no public
telemetry, local telemetry opt-in, its two exceptions, content-free statistics)
are untouched.

## Decision

_Open. Decided by the maintainer at the planning interview of
itd-2610030857275099; until then this record is `proposed` and
adr-2609201008476813 stands unchanged._

## Alternatives Considered

Laid out for the maintainer, none chosen yet. Each sends no token unless the
repository refuses anonymous access, and each is shown wrong by the falsifier
named.

1. **Opt-in scheduled check.** Off until the operator turns it on and sets an
   interval; one small request per eligible repository. No supersession under
   the 2026-09-08 precedent: nothing leaves a Mac whose operator never turned it
   on. Shown wrong if operators who want it never find the switch.
2. **Scheduled check on by default, with a switch.** Most useful; reverses the
   2026-09-08 precedent and supersedes the two quoted sentences in part. Shown
   wrong if an operator who never chose it objects to the traffic.
3. **One listing request per organisation.** One request for, say, every
   `mlx-community` repository's current commit, compared on the Mac, so
   HuggingFace learns the organisations, not which repositories are held.
   Shown wrong if the listing is too large or too slow for a large
   organisation.
4. **Ride the operator's own actions.** Compare while Alice searches, or when
   she opens a model's card, so no new connection is made unasked; "when, not
   whether" by the 2026-09-08 line. Does not deliver "at startup and at
   intervals". Shown wrong if models nobody opens go stale anyway.
5. **Extend the existing startup job.** The category job already asks the Hub
   about some repositories at every start; asking for the commit in the same
   request adds no new connection, only a field. Ratifies that job by name.
   Shown wrong if the job's reach (every ready model) is wider than the
   operator expects.
6. **Declined.** No update checks; the operator re-downloads by hand.

## Consequences

_Written with the decision._
