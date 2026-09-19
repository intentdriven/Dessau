---
id: itd-2609091712142715
slug: settings-that-fit-the-model-and-the-machine-by-themselves-gr
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
impact: additive
origin: researcher-authored
production_mode: dictated-and-formatted
---

# Settings that fit the model and the machine by themselves: Gropius picks a served window, concurrency and sampling defaults per model from what the machine can hold and what the model has done, and a change applies without a restart. Held until the usage data says what must, should and could be configured.

## Press Release

> _Seeded from a quoted-text intent capture. Expand into the full press-release narrative before planning._

## Why This Matters

Settings that fit the model and the machine by themselves: Gropius picks a served window, concurrency and sampling defaults per model from what the machine can hold and what the model has done, and a change applies without a restart. Held until the usage data says what must, should and could be configured.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

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
| Decode concurrency | A launch flag, and a pool option fixed at start-up | The whole of Gropius must restart |

The control plane computes exactly this distinction when it answers a save: the
decode concurrency sits in the `restart` set beside the port and the bind
address, and a sampling change is answered with the list of loaded models it
will not reach until they reload. `docs/sampling-defaults.md` states it in
prose. So the headline's promise holds for one setting of three, and the phrase
has two readings — "without restarting Gropius" and "without relaunching the
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

Three of the four models in the 2026-09-06 campaign were bounded by Gropius's
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
would come to mean "serve whatever Gropius worked out today" — a silent change
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
  window, or a reading that is Gropius's bound rather than the model's — has to
  come back clean before a figure is enforced rather than displayed.

### What the maintainer must decide before this can be planned

- **Which settings are in scope.** The served window, the concurrency and the
  sampling defaults are three features with three different readiness dates and
  three different risk profiles. They may well be three intents.
- **Derived, or suggested?** `AdoptMeasurement` already implements the
  propose-and-confirm shape: Gropius measures, Alice adopts. Does this intent
  remove her from that loop, or does it only make the proposal better? These
  are different products.
- **Where a derived value lives.** Computed at read time from a zero — always
  current, invisible in the file Alice hand-edits — or written back into
  `config.json` as a concrete number, visible and stable but then
  indistinguishable from a value she chose. The two answers give opposite
  answers to "a derived value versus a set one".
- **What an operator's own value does to a derived one.** Whether an explicit
  setting pins the value permanently, is re-derived when its inputs change, or
  is honoured until Gropius can show it is wrong — and what Alice is told in
  each case. Whatever the rule, a save that touches nothing related must still
  succeed.
- **What "without a restart" means.** Without restarting Gropius, or without
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
