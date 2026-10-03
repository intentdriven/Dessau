---
id: adr-2610030929484599
slug: dessau-serves-decision-models-from-its-own-rewrite-of-the
status: accepted
date: 2026-10-03
supersedes: null
superseded_by: null
related_intents: [itd-2610030656210408]
related_rfcs: []
related_adrs: []
---

# ADR-2610030929484599: Dessau serves decision models from its own rewrite of the reference implementation

## Context

itd-2610030656210408 (planned 2026-10-03) serves Clef-family decision models at
`POST /v1/systemone`. Their makers publish a reference implementation,
`clef_mlx.py` (Apache-2.0), inside each model repository: request encoding, the
prompt layout, a joint schema head over the base model, loading through
mlx-vlm, and a small HTTP server. The adversarial design review of the intent
found the reference server unsafe to run as it is: it fetches image URLs for
clients, passes client keyword arguments to the image processor, reads the body
with a negative Content-Length, truncates an over-long state silently, falls
back to downloading from the Hub, and binds its own address. Dessau already
refuses to run Python that comes with a download (the decision of 2026-10-03
on iss-2610030709283687), and the maintainer chose at the interview to serve
only the exact upstream version Dessau has reviewed. The question this record
settles is how Dessau's own code relates to the reference.

## Decision

Decided by the maintainer on 2026-10-03.

**We will serve decision models from Dessau's own rewrite of what the
reference implementation does, not from a copy of it.** Dessau re-implements
only what it needs (request encoding, the prompt layout, the joint head, and
loading the reviewed weights through the runtime's mlx-vlm), in Dessau's own
style, as Python embedded in the binary and launched through the same private
launcher as every model server. It is held to the reference by golden fixtures:
for each reviewed build, the reference server's recorded output for a fixed
request set, which Dessau must reproduce (same top answer, every probability
within 0.001).

- **Attribution.** The rewrite is written against the reference's behaviour, so
  Dessau credits it: `ACKNOWLEDGEMENTS` names the reference implementation, its
  authors and its Apache-2.0 licence, and the rewritten module's header says
  which upstream version it was checked against.
- **Upstream changes** enter only through a Dessau release: someone reads the
  upstream change, re-does it in the rewrite, regenerates the golden fixtures
  from the new reference version, and the change gets an adversarial security
  review like any runtime change.
- **No reference code runs.** Nothing from a model repository is imported or
  executed, including the reference itself.

## Alternatives Considered

1. **A close, trimmed copy.** Keep the reference's structure and remove only the
   unsafe parts, so a new upstream version can be compared line by line.
   Rejected by the maintainer: Dessau would carry code shaped like someone
   else's, and the unsafe parts are woven through the server, not confined to
   it.
2. **Run the reference from the snapshot.** Rejected on 2026-10-03 (the decision
   line that chose the hand-run reference server meanwhile): it runs Python
   fetched from a model repository.
3. **The rewrite (chosen).** Cleaner to own and to review; every upstream change
   has to be understood and re-done by hand, which the golden fixtures make
   checkable.

## Consequences

- spc-2610030846273729's decision-server section describes a rewrite held by
  golden fixtures, not a trimmed copy.
- Each reviewed build carries its golden fixture set, generated from the
  reference at that build's pinned upstream version; the fixtures, not the
  reference code, are what the repository keeps.
- An upstream change to the prompt layout or the head is invisible until
  someone reviews it; a version Dessau has not reviewed is never served
  (itd-2610030656210408, "reviewed version only").
- `ACKNOWLEDGEMENTS` gains its first entry for code behaviour Dessau
  re-implements.
