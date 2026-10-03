---
id: adr-2610031153127219
slug: api-clients-may-unload-an-idle-unpinned-model-through-a
status: accepted
date: 2026-10-03
supersedes: null
superseded_by: null
related_intents: [itd-2610031024247803]
related_rfcs: []
related_adrs: [adr-2610031016411351]
---

# ADR-2610031153127219: API clients may unload an idle, unpinned model through a Dessau route on the model API

## Context

The model API under `/v1` has been read-only plus inference: listing models
and answering requests. itd-2610031024247803 (planned 2026-10-03) lets a
program free a model's memory when it has finished with it, and the
maintainer decided at its interview who may, and which models are protected.
OpenAI's API has no unload verb, and its `DELETE /v1/models/{id}` deletes a
fine-tuned model, so the route must not borrow that shape. Model ids contain a
slash, which rules out a path parameter followed by a verb in Go's router. The
gateway already has one rule for who is entitled to see residency
(`entitled`): this Mac on a keyless install, key holders and this Mac on a
keyed one, paired clients.

## Decision

Decided by the maintainer on 2026-10-03.

**We will add one state-changing verb to the model API: `POST
/v1/dessau/unload` with `{"model": "<id>"}` and a JSON content type, admitted
only for the callers the gateway already entitles**, behind a setting that is
on unless `config.json` turns it off. It unloads a loaded model that is not
pinned, not serving or loading a request, and not inside its eviction-grace
window; every other case is refused at once (409), without waiting and without
interrupting idle work, and a refusal to a caller not entitled reveals nothing
about what is loaded. No `DELETE` is registered on `/v1`, and nothing reachable
from `/v1` can delete a model's files. An unload is recorded with its own stop
reason and the kind of caller, never the key.

The control panel's own `POST /api/models/unload` is unchanged and
documented; it carries no switch of its own (it is also the panel's Unload
button), and who can use it follows the panel-access setting
(adr-2610031016411351).

## Alternatives Considered

1. **Anyone the API admits.** Rejected: on a keyless install every device on
   the network, and a web page a person visits, could unload models.
2. **A key required everywhere.** Rejected: a keyless install, the common
   first-run setup, would offer no unload even to programs on this Mac.
3. **Unload pinned models too.** Rejected: it reverses the pin promise
   (itd-2609061441241254).
4. **A switch on the control panel's route.** Rejected: it cannot be enforced
   without also removing the panel's Unload button.
5. **Trusted callers, protected models refused (chosen).**

## Consequences

- `/v1` is no longer read-only plus inference; any further state-changing verb
  needs its own decision.
- A trusted caller may unload any unprotected model, whoever used it; Dessau
  does not track ownership.
- Statistics gain a stop reason distinct from the operator's unload.
