---
id: itd-2609200823520756
slug: alice-is-told-why-a-value-cannot-be-saved-as-she-types-it-in
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609081259493890]
severity: major
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# Alice is told why a value cannot be saved as she types it, in the server's own words, and Save stays live: a change to a control in the control panel is checked against the same rules the server saves by, the message appears beside the control, and nothing is disabled, marked invalid or blocked on the page's own judgement — the server remains the only authority on refusal. The check is pure, reads only what was posted, changes nothing and never discloses a stored value it was not given.

## Press Release

Alice is setting the memory budget on the Gropius control panel. She types a
figure, and before she reaches for **Save** a sentence appears beside the field:
the server's own words, saying that this figure is below what the models she has
pinned need to load. She corrects it and the sentence goes. Nothing was disabled
while she was typing; **Save** was live the whole time, and had she pressed it
anyway the server would have refused with the same sentence. The page did not make
a judgement about her figure — it asked the server and repeated the answer.

That distinction is the whole of this intent. The panel gains one route to ask
"would this save hold?" and one place beside each control to print the answer. The
route saves nothing, applies nothing and writes nothing; it reads the document it
was posted and returns what the server would have said. Bob and Carol never see
any of it: they use the API, and the panel — and this route with it — stays on
loopback, reachable from this Mac alone.

## Why This Matters

The promise "in the server's own words, as she types" has no surface to come from.
The control plane has seventeen routes and not one of them validates:
`POST /api/settings` is the only path that runs `config.Validate`, and on success
it saves. Today the only way to learn why a value is refused is to attempt the
save. The panel matches that shape — one message element, set once, after the POST
returns, plus `alert()` for model actions — and the `input` listeners that exist
set a touched flag and update two hand-written hints without validating anything.

Two things stand in the way of attributing an answer to a control. First,
refusals carry no field. `Config.Validate` is fail-fast and returns one prose
`error`, which the gateway flattens into the OpenAI error envelope, so every
settings refusal — port, host, grace, sampling, chat rule, pin fit, budget —
arrives as one `400` with one flat sentence, and the prose is not uniformly spelled:
some messages name the `config.json` key, others name the concept in English.
Attribution means either a field-tagged error carried through `internal/config`,
`internal/app` and `internal/gateway`, or string-matching prose that was never
written to be matched. Second, `config.Validate` is not the whole save predicate:
the budget and pinned-fit refusals live only inside `app.SetConfig`, which
validates, persists and applies in one body, so a check-without-saving path means
splitting that into a check and an apply. There is a cheaper seam alongside both:
`config.SamplingBounds` already returns a machine-readable bound table and is
served to nothing, so serving the rules from Go is the direction that makes
as-you-type feedback less duplicated rather than more.

Then there is the boundary, and it is why this is `severity: major` rather than a
tidy-up. adr-2609091123526871 §7 records that every bind holds loopback and the
gateway exempts loopback from the bearer check, so any other macOS account on this
Mac reaches the control plane without the key — an accepted cost, documented in
`docs/bind-address.md`. A validation route is therefore a new unauthenticated input
surface called on every keystroke, and this repository has already been bitten by
that hazard class on this exact code path: the audit notes on itd-2609081259493890
record that adversarial review found a secret-equality oracle in the refusal path.
An early `refusalNamingWhatChanged` compared stored and posted secret values, so a
loopback caller could test a guessed `api_key` or `hf_token` by posting it beside a
value guaranteed to be refused and reading whether the secret was named among the
changes. A per-keystroke validation route is that shape with the rate limit
removed. So purity is not a nicety here: validation is computed from the posted
document alone or from a redacted clone, it mutates nothing, and it never discloses
a stored value it was not given.

Finally, this must not undo what itd-2609081259493890 shipped. That intent
established that the client never refuses a form the server would have accepted,
and `TestNoSettingsControlIsNarrowerThanValidate` holds the markup's bounds to
`config.Validate` for nineteen numeric controls. "Told as she types" is compatible
with that rule in exactly one form: advisory. It explains; it does not gate. And
the gating failure is invisible to the existing test, which reads `min`, `max` and
`step` attributes and knows nothing about script that blocks a submit — so the
no-gating promise needs a test of its own.

## Mechanism

We expect a validation route to be safe on the loopback control plane **because**
validation can be made a pure function of the document it was posted: it computes,
returns and touches nothing. The falsifier is a refusal that cannot be computed
without reading a stored value — in which case that refusal is computed over a
redacted clone, or it is not offered as-you-type at all.

We expect the secret-equality oracle to be excluded by construction **because**
the oracle in the audited refusal path arose from comparing a posted secret with a
stored one; a validate route that never reads a stored secret has nothing to
compare, and a test that plants a guessed key beside a refused value and asserts
the response names no secret is the check that keeps it so. The falsifier is a
response whose content differs depending on whether a posted secret matched what
is stored.

We expect per-field attribution to be achievable **because** half of it is already
shipped and structural: `changedSettings` and `refusalNamingWhatChanged` in
`internal/gateway/control.go` compute which settings a save would move, so a
cross-field refusal can already name a field that actually changed. What is
missing is a field key on the error itself, which means a field-tagged error type
carried through `internal/config`, `internal/app` and `internal/gateway` rather
than string-matching prose that was never written to be matched. For the numeric
ranges the cheaper route is to serve the rules: `config.SamplingBounds` already
computes a machine-readable table. The falsifier is a refusal in `config.Validate`
that cannot be attributed to any single field even in principle, which is then
attributed to the pane rather than to a control.

We expect the split of `app.SetConfig` into a check and an apply to be the whole
of the server-side work **because** the refusals that `config.Validate` does not
carry — the memory budget and pinned-fit ones — live only in that method's body,
which validates, persists and applies together. The falsifier is a refusal
reachable only after persistence has begun.

We expect advisory-only feedback to hold the shipped rule **because** the rule is
about refusal, not about explanation: a sentence beside a control refuses nothing.
The falsifier is any script path that disables **Save**, marks a field invalid or
cancels the settings submit on the page's own judgement.

## Scope Conditions

- **Advisory only. It never gates.** **Save** stays live, no field is marked
  invalid, no submit is blocked on a client-side judgement, and the server remains
  the only authority on refusal (itd-2609081259493890). A message that appears
  beside a control is an explanation, and the operator may ignore it and save.
- **The loopback control plane, one Mac, one operator.** The route lives on the
  panel's control plane, which is loopback-only and bearer-exempt by
  adr-2609091123526871 §7 — so it is reachable by every account on this Mac and by
  nothing on the network. This intent does not change that reach; it accepts it,
  and designs for a caller who is not necessarily Alice.
- **Purity is a scope condition, not an implementation detail.** Validation is
  computed from the posted document alone, or from a redacted clone of the stored
  document; it writes no file, starts and stops no process, applies nothing, and
  never returns or confirms a stored value it was not given.
- **An adversarial security review is a precondition of landing.** Not a follow-up.
  The route is a new unauthenticated input surface on a declared trust boundary,
  and the oracle recorded in itd-2609081259493890's audit notes is the precedent
  that makes the review load-bearing rather than ceremonial.
- **It ships into the panel as it stands.** The route and the per-field slots do
  not wait on the control-panel redesign (itd-2609100519003748), which is what
  makes this intent's value independent of that redesign's scope. What the redesign
  owes this intent is a per-field error slot beside every control; until then the
  slots this intent needs are added where its own controls are.
- **`impact: additive`.** No stored format changes: no new `config.json` field, no
  change to what a save writes. A new route and new markup only.

## Acceptance Criteria

- **Given** the control plane, **when** a validate call is made with any document,
  **then** the route answers and nothing on this Mac changes. *Held by:* a test
  that a validate call over a temporary configuration directory leaves every file
  byte-identical, starts and stops no model process, and leaves the in-memory
  state — resident models, notices, bind plan — as it was.
- **Given** an `api_key` or `hf_token` value posted to the validate route beside a
  value guaranteed to be refused, **when** the posted secret happens to equal the
  stored one and when it does not, **then** the two responses are identical.
  *Held by:* a test that plants a guessed key beside a refused value in both
  arrangements and asserts byte equality of the responses — the direct regression
  test for the oracle recorded in itd-2609081259493890's audit notes.
- **Given** a document whose refusal depends on more than one field, **when** it is
  validated, **then** the response attributes the refusal to the field this
  document changed rather than to whichever field `Validate` happened to read
  first. *Held by:* a test over the cross-field refusals — pinned fit and the
  memory budget — asserting the named field is the changed one, built on the
  `changedSettings` computation that already exists in
  `internal/gateway/control.go`.
- **Given** a refusal attributed to a field, **when** the panel renders it,
  **then** the sentence appears in that control's own error slot and not in the
  pane-wide message element. *Held by:* a node-lifted `internal/ui` test on the
  render path, of the kind the existing `internal/ui` tests already use.
- **Given** any sentence the route returns, **when** it is read, **then** it is the
  server's own refusal text and not a sentence the panel composed. *Held by:* a
  test that the panel's script carries no refusal prose of its own for the fields
  this route covers — the drift this repository has already captured as
  iss-2609190029273153.
- **Given** a control showing a refusal message, **when** Alice presses **Save**,
  **then** the form posts. *Held by:*
  `TestNoSettingsControlIsNarrowerThanValidate` staying green, plus a new test that
  no script path disables the submit control or cancels the settings submit event
  on a client-side judgement.
- **Given** `app.SetConfig`, **when** the check half is called, **then** it returns
  the budget and pinned-fit refusals without persisting or applying anything.
  *Held by:* a test on the check half asserting the refusal and an unchanged
  configuration file.
- **Given** `docs/`, **when** this lands, **then** one page states what the panel
  checks as you type, that the check is advisory, and that the server remains the
  only authority on refusal — a single Diátaxis type. *Held by:* the page, plus a
  line in the shipping record naming it.

## Open Questions

_None recorded yet._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
