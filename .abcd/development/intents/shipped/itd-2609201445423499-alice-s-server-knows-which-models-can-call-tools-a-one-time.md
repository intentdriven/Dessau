---
id: itd-2609201445423499
slug: alice-s-server-knows-which-models-can-call-tools-a-one-time
spec_id: spc-2609201451069488
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609091301112705]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# Alice's server knows which models can call tools: a one-time probe per model, recorded server-side and shown on the models list

## Press Release

Alice downloads a model and, the first time it is served, Dessau Server
asks it one question with one small tool declared, the way it already
measures a new model's servable context. If the model answers with a tool
call, the server records that it can call tools; if it answers with text or
nothing, it records that it cannot. The answer lives beside the model's
other measurements, shows on the models list as a plain capability field
that any client can read, and appears on the model's card in the control
panel. Bob's Dessau Chat reads it and knows, before he asks, whether search
will come from the model's own tool calls or from the search-first switch;
Carol's coding tool reads the same field. Nothing is asked twice: the probe
runs once per model, and again only when the model or the runtime changes.

## Why This Matters

The pinned model server renders tools into the chat template and returns a
tool call when the model makes one, but ignores `tool_choice` and cannot
force it; whether a given model ever makes one depends on its template and
its training, and on this runtime some families answer empty. Every client
that wants tools would otherwise have to find that out for itself, once per
client per model. The server already measures a model once and tells every
client (the context probe); this is the same shape for one more fact, and it
is what the two search intents build on.

## Mechanism

Confirmed by the maintainer at the 2026-09-20 interview: we expect one probe request per
model to settle whether it calls tools because the answer is a property of
the model's chat template and weights on this runtime, not of the prompt,
so a single well-formed request is as informative as a hundred. What would
show this wrong: a model that calls tools for some prompts and not others
on the same runtime, or the answer changing between runs with nothing
updated.

## Scope Conditions

- Served models on this Mac only; the on-device model and the bridge are <!-- cond: cond-2609201451067566 -->
  not probed, and a remote server's field is that server's.
- One probe per model per runtime version, run when the model is first <!-- cond: cond-2609201451063083 -->
  served, never on a client's request; the result is stored with the
  model's measurements and re-run only when the model or the runtime
  changes.
- The probe's prompt is fixed, short and the server's own; it carries no <!-- cond: cond-2609201451062229 -->
  person's text and is not logged as a request.
- The field is informative, never enforced: a client may still send tools <!-- cond: cond-2609201451067098 -->
  to a model recorded as unable, and the server relays them as it does today.

## Acceptance Criteria

Confirmed by the maintainer at the 2026-09-20 interview, every bullet walked and accepted:

- Given a model is served for the first time, when its context has been
  measured, then one tool-bearing request is sent to it and its answer is
  recorded as can-call-tools or cannot, with the runtime version it was
  measured under.
- Given the models list, when a client reads a model entry, then it carries
  the recorded capability in a plain field, and a model not yet probed says
  so rather than saying no.
- Given the control panel, when Alice opens a model's card, then the field
  is shown beside the measured context.
- Given the runtime is updated, when a model is next served, then it is
  probed again and the field updated.
- Given the probe, when it runs, then it appears in no request statistic
  and in no request log line, like the self-test's own runs.

## Open Questions

_None recorded yet._

## Audit Notes

<!-- abcd-review: OWED receipt=rcp-89e6e831c5c0 -->
Fidelity review OWED (receipt rcp-89e6e831c5c0).

## Grounds

- pursued: every client that wants tools would otherwise discover this per client per model; wrong if the answer varies by prompt
