---
id: itd-2609091712142715
slug: the-served-window-fits-the-model-and-the-machine-by-itself-g
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609091712141073, itd-2609100457007827, itd-2609091301112705, itd-2609091903463596]
severity: minor
impact: breaking
origin: researcher-authored
production_mode: dictated-and-formatted
---

# The served window fits the model and the machine by itself: Dessau measures what a model can hold on this Mac and proposes it as the served window, Alice adopts it or keeps her own, and the change applies without restarting Dessau

## Press Release

Dessau proposes the served window for each model from what it measured on your
own Mac, and you decide whether to take it.

Alice runs Dessau on the Mac in her study, with four models on it. Until now
the window each model is served at was either the number the model's own file
declares — frequently unreachable on this machine — or a number she typed after
reading a benchmark somebody else ran. Dessau already measures what each model
can actually hold here. What is new is that the measurement arrives as a
proposal: the models list shows the measured window beside the declared one,
marked as Dessau's suggestion and dated, and one "Adopt" makes it the served
window for that model. Nothing is decided for her and nothing is written into
her `config.json` behind her back.

If Alice has typed her own figure, her figure stands: an explicit value pins
until she clears it, and clearing it hands that model back to the proposal. If
she has left the setting at zero, Dessau serves the figure it worked out,
computed when the setting is read rather than frozen into a file, and the panel
says the figure is derived and when it was measured. A derivation that has gone
stale — a new runtime, a new model, a machine that has changed — is shown with
its date and left at that; Dessau never enforces a figure it can no longer
stand behind. Adopting a window, or clearing one, takes effect without
restarting Dessau, though the model server may still be relaunched before it
serves the new figure.

Decode concurrency is out of scope, because it is one machine-wide setting
handed to every model server the pool launches, so there is no per-model figure
to propose and no evidence on record that varies it. Sampling defaults are out
of scope, because a temperature is a taste rather than a fit, and no usage
record can say what somebody's preferred temperature ought to be.

## Why This Matters

The served window is the one setting here where the measuring and the deciding
are both already built and still not joined up. The context probe writes a
measured window per model; `AdoptMeasurement` turns a measured figure into the
served window as an ordinary settings save. Between the two sits a person who
has to know that both exist, go and find the number, and satisfy herself that
it beats what she has. Alice will not do that for every model after every
runtime upgrade, so in practice the figure she serves drifts back to the
declared window — a claim about the architecture, not about her Mac — and her
clients size their prompts from a number nothing on the machine supports, then
lose an afternoon to a refusal halfway through a long file. A proposal she can
see costs her one decision per model and takes the guess out of the gateway.
Deriving the value silently instead would buy little and would cost the property
that makes the setting safe: a number Alice can see is a number she can
overrule.

## Mechanism

We expect the work here to be small and the risk to sit in the proposal rather
than in the plumbing, because both ends already ship and are exercised: the
context probe measures a servable window per model and publishes a
gateway-bounded reading as a floor that names the bound which stopped it, and
`AdoptMeasurement` already performs the propose-and-confirm save. What this
intent adds is therefore the quality of the proposal and the rule for when a
proposal has gone stale, not a new mechanism for applying one. We expect
zero-means-derived to be safe for this setting because two settings already
carry that shape on this codebase — the resident memory budget and the upstream
header timeout both read zero as "derive it" and a positive value as the
operator's override — so the read-time seam is precedent rather than invention,
and no new field is needed. We are wrong if the proposal is not trustworthy
enough to show: a repeat probe on the same Mac, the same models and the same
runtime returning a materially different window, or a reading that turns out to
be Dessau's own upstream bound rather than the model's, which makes the
proposed figure move whenever `upstream_header_timeout_sec` does. We are also
wrong if a value computed at read time cannot be shown with the date it was
measured, because a figure that cannot state its own age cannot be governed by a
staleness rule.

## Scope Conditions

- The served window only. Decode concurrency and sampling defaults are outside this intent: the concurrency is machine-wide and cannot express a per-model figure in the configuration's present shape, and sampling is a preference no usage record can decide.
- Suggested, not derived behind her back. Dessau measures and Alice adopts, as `AdoptMeasurement` already does; this intent improves the proposal, it does not remove her from the loop.
- A derived value lives at read time behind zero. Zero means "serve what Dessau worked out", resolved when the setting is read and shown on the panel as derived; the figure is never written into `config.json`. Zero's meaning changes for a value already in every operator's file, which is why `impact` is breaking; pre-1.0 permits the break and no migration is written for it.
- An explicit operator value pins. A figure Alice typed is honoured until she clears it, never re-derived under her, and a save that touches nothing related must still succeed.
- A stale derivation is displayed, never enforced. It is shown with the date it was measured; Dessau does not change a served figure under a running service on the strength of a reading it can no longer stand behind.
- "Without a restart" means without restarting Dessau. The model server may still be relaunched before the new figure is served.
- Single-machine evidence. Every figure on record comes from one 128 GB Apple Silicon Mac, so the derivation is fitted to that machine until a second Mac's figures exist; the same Apple Silicon, one-serving-process-per-Mac and pinned-runtime conditions the context probe records apply here unchanged.

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

Not written, and deliberately so. This draft is held (see `## Hold`), and the
bar for "shipped" depends on the evidence that lifts the hold: the prompt-size
distribution against the served window per model and the served-window refusal
rate are what say whether a proposal is any good, and the self-test cycle's
figures are what a criterion would be measured against. Criteria written before
those figures exist would be a guess wearing a gate's clothes. They are written
with the maintainer in the planning interview, once the hold lifts.

## Hold

Held since 2026-09-19 at the adversarial review below, and confirmed held by the
maintainer on 2026-09-20. The condition is not met: no period of real serving
with request statistics on has been read back, and no self-test cycle is on
record.

The evidence that lifts it, verbatim from the 2026-09-20 decision in
`.abcd/work/DECISIONS.md`:

> a stated period of real serving on a stated machine with request statistics
> on, read back as the prompt-size distribution against the served window per
> model, the served-window refusal rate, the observed in-flight counts and the
> sampling override rate per parameter; plus one completed self-test cycle with
> its figures on record.

Nothing else lifts it. Lifting for the served window alone now, and lifting
entirely, were both put to the maintainer on 2026-09-20 and both declined. Until
those figures are on record this draft is not planned and no spec is minted, and
`abcd intent ready itd-2609091712142715 --json` returns `ready: false` — the
correct state for a held draft rather than a defect in the record.

## Review 2026-09-19

Two adversarial passes over this draft — design and feasibility, then record
discipline — read against the shipped runtime, `../../decisions/DECISIONS.md`,
the shipped intents this draft waits on (itd-2609091712141073 usage
measurement, itd-2609091903463596 the resources view, itd-2609100457007827 the
self-test, itd-2609091301112705 the context probe) and the evidence under
`../../research/evidence/`. Verdict: **the hold stands.** Nothing was planned
and no spec was minted.

### The hold is not met

The title holds this draft "until the usage data says what must, should and
could be configured". That is three tests, and the record fails all three.

- **There is no usage data on record.** The dashboard that produces it shipped
  with itd-2609091712141073; request statistics are opt-in
  (cond-2609091737259918) and the self-test is opt-in and off by default
  (cond-2609100502039780). Nothing anywhere in `.abcd/` records either switch
  having been on, on any machine, for any period.
- **Nothing has been read back.** The only measurement records are the
  2026-09-06 campaign — four models, one Mac, one evening, run by hand. That
  campaign is what the dashboard and the self-test were built to replace:
  itd-2609091712141073 says the deciding facts "today are guessed", and
  itd-2609100457007827 says the campaign is "a record that goes stale with the
  next model, the next runtime or the next machine". Evidence that predates the
  instruments cannot satisfy a hold those instruments exist to lift.
- **No must/should/could split exists.** No record states which of the served
  window, the concurrency and the sampling defaults must, should or could be
  configured.

### What the runtime can and cannot do today

"A change applies without a restart" is already false for two of the three
settings the headline names, and the shipped code says so itself:

| Setting | Consumed where | What a change costs today |
| --- | --- | --- |
| Served window | The gateway, per request | Nothing — the next request is judged against it |
| Sampling defaults | mlx-lm launch flags | The model server must be relaunched |
| Decode concurrency | A launch flag, and a pool option fixed at start-up | The whole of Dessau must restart |

The control plane computes exactly this distinction when it answers a save: the
decode concurrency sits in the `restart` set beside the port and the bind
address, and a sampling change is answered with the list of loaded models it
will not reach until they reload. `docs/sampling-defaults.md` states it in
prose. So the headline's promise holds for one setting of three, and the phrase
has two readings — "without restarting Dessau" and "without relaunching the
model server" — that produce different pieces of work. The draft does not say
which it means.

### The three settings are three features, not one

- **The served window** is per-model already, and the automation gap is one
  step wide: the context probe writes a measured window per model, and
  `AdoptMeasurement` already turns that figure into the served window as an
  ordinary settings save. What is missing is only the decision to do it without
  asking.
- **The concurrency is not per-model at all.** It is a single machine-wide
  setting handed to every server the pool launches, so "per model" is not
  expressible in the configuration's present shape. Making it so is a new field
  on the one per-model settings struct, plus the live-reload fix above.
- **Sampling is the riskiest third and the least supported.** A temperature is
  a taste, not a fit, and nothing a usage record can hold says what one should
  be; the dashboard records how often clients override a default, which says a
  default is unwanted, not what it should be. DECISIONS items 8 and 9 make a
  wrong default actively dangerous: an out-of-range value starts a healthy
  looking server that then fails every request which omits the parameter, and
  an oversized `top_k` raises from inside compiled generation.

### The evidence that does exist gives a ceiling, not a default

Three of the four models in the 2026-09-06 campaign were bounded by Dessau's
own upstream timeout rather than by the model, which is why the probe intent
already rules that such a reading is published as a floor and names the bound
that stopped it. A window derived from such a figure inherits that ambiguity
and moves whenever the bound moves. The only concurrency evidence on record is
a single pair of concurrent 64K prompts on one model, whose finding is that the
right concurrency depends on the prompt-size distribution — precisely the usage
data that does not exist. The self-test measures generation under the
concurrency *in force*, so it cannot say what the concurrency *should* be.

### The derived-versus-set question has a precedent and an unexploded collision

Two settings already carry the shape this draft needs: the memory budget and
the upstream header timeout both treat zero as "derive it" and a positive value
as the operator's override, the latter with the reasoning written out — a
derivation encodes a measurement, and a measurement can be wrong for hardware
nobody tested. The served window reads the same way today, falling back to the
declared window when unset, so adopting the pattern costs no new field.

The collision is that zero currently means "serve the declared window" and
would come to mean "serve whatever Dessau worked out today" — a silent change
of meaning for a value already in every operator's `config.json`, resolving to
a number that moves between runs. Pre-1.0 permits the break; it does not excuse
leaving it unstated. And if a derived figure is instead written back into the
file as a concrete number, it becomes indistinguishable from one Alice chose,
and a derived window large enough to fail the pinned-fit check would refuse a
save about something else entirely — the fourth appearance of the wedge this
repository has already built three times.

### Record discipline

- `builds_on` is empty. Five records point at this draft — the usage
  measurement, the self-test ("the held draft itd-2609091712142715 ... is what
  these figures exist to feed"), the self-test's own spec, the decomposition
  note and the 2026-09-10 decision line — and it points at none of them, so a
  reader arriving here cannot find what would lift the hold.
- The hold lives in the H1, and therefore in the slug and the filename. When it
  lifts, the title becomes false and cannot be corrected without breaking every
  link into the record. It belongs in the body.
- The Press Release is still the seeding stub and Why This Matters is the
  headline pasted back, so the record holds one sentence of content, written
  twice. No persona appears anywhere: Alice is the person who would otherwise
  choose these settings by hand, and she is absent from the one record about
  choosing them for her.
- `impact: additive` understates it. Picking a served window without being
  asked changes what the gateway refuses and what the budget charges on an
  existing install; `impact: breaking` is the honest label under the pre-1.0
  rule.
- `abcd intent ready itd-2609091712142715 --json` returns `ready: false` on
  bucket, acceptance criteria, scope conditions, spec link and spec body. That
  is the correct state for a held draft, recorded here so a later reader does
  not mistake the gate's silence for readiness.

## Open Questions

Recorded 2026-09-19 at the adversarial review. The draft stays held. These are
in two groups, and they are two separate holds: the first is lifted by
evidence, the second only by the maintainer.

### What evidence would lift the hold

**This group is the live hold (2026-09-20).** The maintainer confirmed it stands
and is lifted by evidence only; `## Hold` above states the exact reading that
lifts it. The first two bullets are that reading. The third is now answered as a
scope condition rather than as evidence — one Mac is enough, and the record says
so — and the fourth falls out of scope with the concurrency; the fifth is the
condition under which a proposal may ever be enforced rather than displayed, and
under the 2026-09-20 decisions a derived figure is displayed in any case.

- **A period of real serving with request statistics on, read back.** The
  dashboard shipped; no reading of it exists. What is needed on record is a
  prompt-size distribution against the served window per model, the rate at
  which a served-window refusal bit, the observed in-flight counts, and the
  sampling override rate per parameter — taken from actual traffic over a
  stated period on a stated machine, not from a benchmark.
- **A completed self-test cycle with its figures on record.** The self-test is
  off by default and nothing says it has run. Its own mechanism claim is
  falsified "if a setting the figures were meant to decide turns out to need a
  measurement not in the set"; the concurrency choice looks like exactly such a
  setting, so a cycle would either supply the number or falsify the claim.
- **Evidence from more than one Mac, or a statement that one is enough.** Every
  figure on record comes from a single 128 GB machine. A derivation shipped to
  other people's Macs either needs a second machine's figures or an explicit
  scope condition saying it is fitted to this one.
- **A concurrency measurement that varies the concurrency.** The only such
  evidence is one pair of 64K prompts on one model, and the self-test measures
  the value already in force. Nothing on record can currently choose between 2,
  4 and 8.
- **Confirmation that a derived served window is stable enough to enforce.**
  The probe's own falsifier — a repeat probe returning a materially different
  window, or a reading that is Dessau's bound rather than the model's — has to
  come back clean before a figure is enforced rather than displayed.

### What the maintainer must decide before this can be planned

**Answered 2026-09-20** (maintainer's interview; the decisions are recorded in
the 2026-09-20 line of `.abcd/work/DECISIONS.md` naming this intent, and each is
now carried by this record's own body). This group is closed; the evidence group
above is the only hold that remains.

- **Which settings are in scope** — answered: the served window only, in `## Press Release` and the first `## Scope Conditions` bullet.
- **Derived, or suggested?** — answered: suggested, Dessau measures and Alice adopts as `AdoptMeasurement` already does, in `## Press Release`, `## Mechanism` and the second scope bullet.
- **Where a derived value lives** — answered: at read time behind zero, shown as derived, never written into `config.json`, in the third scope bullet and `## Mechanism`.
- **What an operator's own value does to a derived one** — answered: an explicit value pins until she clears it, in `## Press Release` and the fourth scope bullet.
- **What "without a restart" means** — answered: without restarting Dessau, a model relaunch may still be needed, in the H1, `## Press Release` and the sixth scope bullet.
- **What happens when a derivation goes stale** — answered: displayed with its date, never enforced, in `## Press Release` and the fifth scope bullet.
- **The record's own shape** — answered: the H1 is retitled so the hold clause leaves the slug, the hold moves to `## Hold`, and `impact` becomes `breaking` because zero changes meaning for a value in every operator's file.

- **Which settings are in scope.** The served window, the concurrency and the
  sampling defaults are three features with three different readiness dates and
  three different risk profiles. They may well be three intents.
- **Derived, or suggested?** `AdoptMeasurement` already implements the
  propose-and-confirm shape: Dessau measures, Alice adopts. Does this intent
  remove her from that loop, or does it only make the proposal better? These
  are different products.
- **Where a derived value lives.** Computed at read time from a zero — always
  current, invisible in the file Alice hand-edits — or written back into
  `config.json` as a concrete number, visible and stable but then
  indistinguishable from a value she chose. The two answers give opposite
  answers to "a derived value versus a set one".
- **What an operator's own value does to a derived one.** Whether an explicit
  setting pins the value permanently, is re-derived when its inputs change, or
  is honoured until Dessau can show it is wrong — and what Alice is told in
  each case. Whatever the rule, a save that touches nothing related must still
  succeed.
- **What "without a restart" means.** Without restarting Dessau, or without
  relaunching the model server? The second is not available for sampling or
  concurrency without a change to what mlx-lm is asked for, and the first makes
  the concurrency's start-up-only binding a prerequisite of this intent.
- **What happens when a derivation goes stale.** A displayed figure may quietly
  expire; an enforced one probably may not, because a setting that silently
  changes under a running service is worse than one that is merely wrong.
- **The record's own shape.** Whether to retitle so the hold leaves the H1 and
  the slug, and whether `impact` becomes `breaking`.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

**Answers 2026-09-20:** HELD; evidence lifts it (a read-back serving period with statistics on, plus one self-test cycle). Decisions 1–7: served window only; suggested; read-time behind zero; explicit pins; no Dessau restart; stale shown not enforced; retitle and impact breaking. Recorded in `.abcd/work/DECISIONS.md`.
