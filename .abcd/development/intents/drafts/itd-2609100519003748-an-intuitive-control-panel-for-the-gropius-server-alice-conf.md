---
id: itd-2609100519003748
slug: an-intuitive-control-panel-for-the-gropius-server-alice-conf
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609081259493890, itd-2609081718534201, itd-2609091903463596, itd-2609200823520756]
severity: major
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# An intuitive control panel for Dessau Server: Alice configures the server from a page that shows what is on before it asks what to change, finds each setting by the task it serves rather than by its key in the file, and is told in the server's own words, as she types, why a value cannot be saved — a state-of-the-art site that stays a page served from this Mac alone, reachable by every account on it and by nothing on the network, so that Bob at the next desk and Carol on the mesh see the API and never the panel

## Press Release

Alice runs Dessau on the Mac under her desk. She opens the control panel and the
first thing the page tells her is what is true right now: which models are
resident and what they are charging against the memory budget, who can reach this
server and by which addresses, whether anything is being recorded. Only then does
it ask her what she would like to change. When she goes looking for a setting she
looks for the job it does — how long a model stays loaded, how much of this Mac
models may use, who may connect — and finds it under that job, not under the name
it happens to carry in `config.json`. When she types a figure the server will not
accept, the page tells her why in the server's own words, beside the control she
is typing into, and **Save** stays live: the page never refuses on its own
authority, because the server is the only thing that knows whether a save holds.

She reaches the panel over loopback, from this Mac. Bob at the next desk points
his editor at the API and gets answers; Carol, on the mesh, does the same from her
laptop in the other room. Neither of them can open the panel, and neither of them
needs to: the panel is the operator's surface and the API is everybody's. The page
itself is three files served from inside the binary — markup, a stylesheet and a
script. It needs no toolchain to build, no framework to run, and nothing from
anywhere else on the internet to render, which is why it still works on a Mac that
has been taken off the network entirely.

## Why This Matters

The panel is the only interactive surface this product has, and it is the surface
that decides whether an operator who is not its author can run this server. Today
it loses that argument on evidence rather than on taste. Seven hundred lines of
markup carry one accessibility attribute. The tab strip is a row of buttons with
no `role="tablist"`, no `aria-selected`, no arrow-key navigation and no focus move
on switch, so a person using a keyboard or a screen reader cannot work the primary
navigation of the page. The panel re-renders live from an event stream with no live
region anywhere, so a download that finishes or a model that loads is announced to
nobody. Errors arrive through `alert()`. The stylesheet's only media queries are
`prefers-color-scheme`; there is no responsive breakpoint, so the page an operator
reaches from a phone on her own network is the desktop layout, scrolled. None of
this is architecture. All of it is authoring.

The second half of the argument is about finding things. The Settings pane is
ordered the way `config.json` is ordered, which is the order the fields were
added, so an operator looking for "how long does a model stay loaded" must already
know it is called an idle timeout. The repository has a guard that proves every
setting is *named* by the pane — it cannot prove that the control is where a person
would look, still posts the right key, or has not been demoted to a read-only
line. That gap is the whole risk of a redesign, and closing it is what makes the
redesign safe to attempt rather than a coin toss.

The third half is trust. The page's job is to state facts about exposure, not to
warn about them: who can reach this Mac, what is announced, what is written down.
An operator who is told the truth plainly can decide; one who is nagged learns to
click past the nagging. That stance — state before settings, facts not warnings,
nothing interrupts the task — has lived in this draft as prose since 2026-09-10.
Prose that no test can fail is the one kind of promise this repository does not
otherwise accept, which is why it becomes acceptance criteria here.

## Mechanism

We expect the redesign to work in plain files, with no build step and no
framework, **because** every defect the 2026-09-19 review found is a defect of
authoring rather than of architecture: aria attributes, a tab list with arrow
keys, a live region, a defined focus order, per-field error slots, a visible focus
ring and a responsive breakpoint are each expressible in the markup, stylesheet and
script that already ship. The falsifier is a named defect from that review that
cannot be fixed without a toolchain.

We expect the existing three-surfaces guard to keep working **because** the guard
is a string scan coupled to the authoring format in six ways — the paths of
`index.html` and `app.js`, the literal `$('someId')` lookup form, `id="…"` present
as literal markup, three named source markers, `<section id="tab-settings"` as the
Settings pane's boundary, and lower snake case object keys — and choosing plain
files changes none of the six. The same argument covers the `internal/ui` tests
that lift functions out of `app.js` and run them under node: the scanned and lifted
source stays the served source. The falsifier is a change to the redesigned page
that makes `internal/archtest/settings_surface_test.go` or any `internal/ui` test
need rewriting to stay green.

We expect the inventory in the spec, rather than the guard, to be what makes a lost
control detectable **because** the guard proves only that a setting is named
somewhere in the Settings pane: a control moved to a pane nobody looks in, demoted
to a read-only line, or left posting the wrong key all pass it. An inventory that
names every control, roll-up block and read-only line with its destination — and
lists anything deliberately dropped with a reason — is a document a reviewer can
diff the delivered panel against, and a test can hold the delivered markup to it
id by id. The falsifier is a control that goes missing while both the inventory
test and the guard stay green.

We expect mechanical accessibility bars to deliver what "intuitive" and "state of
the art" cannot **because** those two phrases have no failing case: no test can be
written that they fail, so they constrain nothing. Keyboard reachability of every
control, a defined focus order, `role=tablist` with `aria-selected` and
`aria-controls` and arrow keys, a live region for state changes, a programmatic
label per control and a visible focus ring each have a failing case that an
assertion over the markup or a node-lifted test can find. The falsifier is a bar
from that list that cannot be asserted without driving a real browser.

We expect the offline constraint to be statable as a testable property **because**
the line that matters is between an automatic subresource fetch and a link the
operator clicks: the page currently carries six links out to documentation on the
forge, so "no outbound host" is already false, while "the page renders and every
control works with this Mac offline" is true today and checkable. The falsifier is
a redesign requirement that needs a fetched subresource to render.

## Scope Conditions

- **No build step, no framework, and nothing fetched to render.** The panel stays
  plain markup, stylesheet and script under `internal/ui/static`, served from the
  embedded file system. The property that holds this is testable and is stated as
  such: the page renders and every control works with this Mac offline. The
  documentation links the panel already carries out to the forge stay as links an
  operator clicks — they are navigation, not subresources — and a link is the only
  form an outbound host may take. Answers Open Question 1 with option A
  (DECISIONS.md, 2026-09-20).
- **The resources block stays at the head of the Models tab.** The placement
  decided at interview on 2026-09-09 (itd-2609091903463596) is re-affirmed rather
  than re-opened: the redesign does not move it to a tab of its own, however it
  reorganises everything else.
- **The tabs are the panel's seven, by name: My Models, Find Models, Statistics,
  Connect, Clients, Posture, Settings.** The redesign re-homes controls within and
  between these; it adds no eighth tab and removes none. The usage dashboard is
  the **Statistics** tab — there is no Usage tab — and the posture page
  (itd-2609081718534201) is inside the scope of what is reorganised.
- **The as-you-type refusal is not this intent's to build.** It is
  itd-2609200823520756, the sibling this intent `builds_on`. What this intent owes
  it is the place the message goes: a per-field error slot beside each control,
  present in the markup and addressable by the script, which the panel does not
  have today. The route, the purity argument and the security review are the
  sibling's.
- **The panel stays reachable over loopback only, and every account on this Mac
  reaches it.** That is not a property this intent changes or defends; it is the
  accepted cost recorded in adr-2609091123526871 §7 and documented in
  `docs/bind-address.md`. The redesign adds no route and no reach.
- **The Settings pane exemption table stays at two entries.** `preload` and
  `upstream_header_timeout_sec` keep their written exemptions
  (spc-2609111941481833); the redesign does not add a third, which would mean a
  setting losing its control under cover of a reorganisation.
- **One operator, one Mac, one page.** No multi-user panel, no remote
  administration, no accounts. `impact: additive`: no stored format changes — but
  the redesign does invalidate navigation instructions in `docs/` and `README.md`,
  which is why the documentation sweep is an acceptance criterion rather than a
  follow-up.

## Acceptance Criteria

- **Given** the spec, **when** a reviewer opens it, **then** it carries an
  inventory naming every control, roll-up block and read-only line on the panel
  today, each with its destination in the new arrangement, and anything
  deliberately dropped listed with its reason. *Held by:* the spec section itself,
  plus a test that every element id the inventory names exists in the delivered
  markup and, for a control, that the script posts the `config.json` key the
  inventory names for it.
- **Given** any pane, **when** its markup is read in document order, **then** the
  state it reports comes before the controls that change it. *Held by:* an
  ordering assertion per pane over `internal/ui/static/index.html`.
- **Given** a refusal from the server or a notice the panel raises, **when** its
  text is read, **then** it states what is true rather than warning about what
  might be. *Held by:* a rule over the refusal and notice strings in the markup
  and script, of the kind `TestTheBindModeLabelsNameNoVendorAndPromiseNothing`
  already applies to the bind labels.
- **Given** the panel's script, **when** it is scanned, **then** it contains no
  `alert()` and no `confirm()` call. *Held by:* an assertion over `app.js`. The
  two `confirm()`/`alert()` sites that exist today are the statistics-clear
  confirmation and `alertErr`. Separately, the existing `confirmBtn` two-click
  arming is replaced by an inline affordance that does not depend on a timed
  three-second window, since a timed interaction is an interruption for anyone
  working the page by keyboard or screen reader. *Held by:* a node-lifted test on
  the replacement.
- **Given** the panel, **when** it is worked by keyboard alone, **then** every
  control is reachable and the focus order is the one the spec defines. *Held by:*
  an assertion over the markup that no control is unreachable (no anchor doing a
  button's work, no positive `tabindex`) plus the spec's stated order asserted in
  document order.
- **Given** the tab strip, **when** its markup is read, **then** it carries
  `role="tablist"`, each tab `role="tab"` with `aria-selected` and `aria-controls`
  naming its pane, each pane `role="tabpanel"`, and the script moves the selection
  on Left/Right arrow keys and moves focus to the newly selected tab. *Held by:*
  an assertion over the markup plus a node-lifted test on the key handler.
- **Given** a state change an operator did not initiate on the page — a download
  finishing, a model loading or unloading — **when** it arrives on the event
  stream, **then** its sentence is written into a live region. *Held by:* an
  assertion that the live region exists in the markup and a node-lifted test that
  the render path writes those sentences into it.
- **Given** any control on the panel, **when** its markup is read, **then** it has
  a programmatic label — a `<label for>`, an `aria-label` or an
  `aria-labelledby` — and no control relies on adjacent text alone. *Held by:* an
  assertion over the markup, enumerated the way
  `TestEveryNumericSettingsControlIsProbed` enumerates today.
- **Given** the stylesheet, **when** a focusable element has focus, **then** a
  visible focus ring is drawn and no rule removes it without replacing it. *Held
  by:* an assertion over `internal/ui/static/style.css` that no `outline: none`
  or `outline: 0` stands without a replacement indicator in the same rule.
- **Given** a viewport at phone width, **when** the panel is rendered, **then** the
  layout reflows rather than scrolling horizontally: the tab strip, the tables and
  the Settings form each fit. *Held by:* a responsive breakpoint asserted present
  in the stylesheet for the named selectors, plus one recorded hand check at phone
  width in the shipping record — a hand check named as a hand check, in the way
  the 2026-09-20 ledger line records two other checks as owed.
- **Given** `internal/archtest/settings_surface_test.go` and the `internal/ui`
  node-lifted tests, **when** the redesign lands, **then** they are green and
  unchanged — no path, marker, lookup form or exemption edited to accommodate the
  new markup. *Held by:* `make test` plus the diff over those files being empty.
- **Given** `docs/`, **when** the redesign lands, **then** there is one page about
  the control panel — a single Diátaxis type, reference or how-to, not both — and
  every existing page whose navigation instructions the re-homing changes has been
  swept. *Held by:* the new page plus a listed sweep in the shipping record naming
  each page checked; the twenty existing pages that mention the panel or its
  address are the candidate set.
- **Given** any value Alice types into any control, **when** she types it,
  **then** **Save** is never disabled, no field is marked invalid and no submit is
  blocked on a judgement the page made. *Held by:*
  `TestNoSettingsControlIsNarrowerThanValidate` staying green, plus a new test
  that the script contains no path which disables the submit control or cancels
  the settings submit event on its own validation.

## Review 2026-09-19

Two adversarial passes over this draft, read as separate hostile jobs: one against
the design and its feasibility on the tree as it stands, one against the record.
Neither pass plans anything and nothing below decides the three questions under
the next heading, which are the maintainer's. The draft stays a draft.

### Design and feasibility

**The headline promise has no surface to come from, and the half nobody noticed is
already built.** The control plane has seventeen routes and none of them validates:
`POST /api/settings` is the only path that runs `config.Validate`, and on success it
saves. So "the server's own words" are reachable today only by attempting the save.
The panel matches that — one message element, set once, after the POST returns, plus
`alert()` for model actions. There is no per-field error slot in the markup and no
per-field error state in the script; the `input` listeners that exist set a touched
flag and update two hand-written hints, and none of them validates. Against that, the
attribution half of the promise is already shipped and the draft does not cite it:
`changedSettings` and `refusalNamingWhatChanged` in `internal/gateway/control.go`
compute, structurally, which settings a save moved, so a cross-field refusal names a
field that actually changed. An intent whose headline promise is half-built, and which
does not say so, invites a planning interview that rebuilds it.

**Refusal messages carry no field.** `Config.Validate` is fail-fast and returns one
prose `error`; the gateway flattens it into the OpenAI error envelope, so every
settings refusal — port, host, grace, sampling, chat rule, pin fit, budget — arrives
as one `400` with one flat sentence. Nothing anywhere carries a machine-readable field
key, and the prose is not even uniformly spelled: some messages name the `config.json`
key, others name the concept in English. Attributing a message to a control means
either a field-tagged error type carried through `internal/config`, `internal/app` and
`internal/gateway`, or string-matching prose that was never written to be matched.
A second obstacle sits beside it: `config.Validate` is not the whole save predicate.
Budget and pinned-fit refusals live only inside `app.SetConfig`, which validates,
persists and applies in one body, so a check-without-saving path means splitting that
into a check and an apply.

**"As she types" collides head-on with a shipped, test-backed criterion, and this is
the sharpest thing the draft must settle.** itd-2609081259493890 shipped the rule that
the client never refuses a form the server would have accepted, and
`TestNoSettingsControlIsNarrowerThanValidate` holds the markup's bounds to
`config.Validate` for nineteen numeric controls. "Told as she types why a value cannot
be saved" is, on its face, a client-side refusal surface. It is compatible only if the
message is advisory — it explains, it does not gate. The moment the redesign disables
Save, marks a field invalid or blocks submit on a client-side judgement, it contradicts
a shipped criterion, and worse, it can do so while that test stays green, because the
test reads `min`/`max`/`step` attributes and knows nothing about script that blocks a
submit. The acceptance criteria must say which it is. The recommendation of this review
is that it never gates: Save stays live and the server remains the only authority on
refusal.

**A validate-as-you-type endpoint is a trust-boundary change on a plane every account
on this Mac reaches.** adr-2609091123526871 §7 states it: every bind holds loopback,
the gateway exempts loopback from the bearer check, so any other macOS account on this
Mac reaches `/v1` without the key and reaches the loopback-only control plane. The
draft's title sells that reach as the feature; the ADR records it as an accepted cost,
documented in `docs/bind-address.md`. Both halves belong in the record. Concretely,
a validation route is a new unauthenticated input surface called on every keystroke —
and this repository has already been bitten by that hazard class on this exact code
path. The audit notes on itd-2609081259493890 record that adversarial review found a
secret-equality oracle in the refusal path: an early `refusalNamingWhatChanged`
compared stored and posted secret values, so a loopback caller could test a guessed
`api_key` or `hf_token` by posting it beside a value guaranteed to be refused and
reading whether the secret was named among the changes. A per-keystroke validation
route is that shape with the rate limit removed. Any spec must carry: validation is
pure, it is computed from the posted document alone or from a redacted clone, it
mutates nothing, and it never discloses a stored value it was not given.

**A framework or a build step costs the repository's own sync guard, and that is a
larger bill than the open question states.** `internal/archtest/settings_surface_test.go`
— the test that arms the three-surfaces rule — does not drive a browser. It string-scans
the panel's source, and it is coupled to the current authoring format in six ways: the
exact paths of `index.html` and `app.js`; the literal element-lookup form `$('someId')`,
whose own comment says an id the script computes is not one it can check; `id="…"`
present as a literal in the markup; three named source markers it opens blocks at
(`const body = {`, `bindSelectBody`, `chatRule`); `<section id="tab-settings"` as the
Settings pane's boundary; and object keys spelled in lower snake case. Every framework
and every build step breaks all six — compiled templates remove the literal ids, a
bundler means the scanned file is not the served file, component libraries compute ids,
declarative bindings stop putting a key and an id on one line, and Go templates move the
markup and the object literal out of the scanned files entirely. So the honest cost is:
adopting a framework or a build step means rewriting the sync guard, and the replacement
must be at least as strong. The same applies, less severely, to the `internal/ui` tests
that lift functions out of `app.js` and run them under node — a build step means the
lifted source is not the shipped source. Those node tests also skip when node is absent,
and the CI workflow installs no node, so the macos runner image is carrying that
guarantee implicitly.

**The offline constraint is right and should be stated as a testable property.** The
no-telemetry ADR refuses data leaving the machine to the project, a vendor or a third
party, and a fetched subresource does disclose that this Mac opened its panel — but that
ADR is written about usage, hardware, error, model and configuration data, so leaning on
it alone is the weaker argument. The stronger one is the draft's own second clause: the
page is served from a Mac that may have no internet, so a page that needs one is a page
that does not render. The line that actually has to be drawn is between an automatic
subresource fetch and a link the operator clicks, because the panel already carries five
links out to the documentation. Stated as a property a test can hold — the page renders
and every control works with the Mac offline — it is unambiguous; stated as "no outbound
host", it is already false.

**"A state-of-the-art site" is unsupported, and the accessibility of the page is the
evidence.** In 586 lines of markup there is exactly one aria attribute, `aria-hidden` on
the logo. The tab strip is six buttons with no `role="tablist"`, no `aria-selected`, no
`aria-controls`, no arrow-key navigation and no focus move on switch. Two controls are
anchors doing buttons' work. The panel re-renders live from a streamed state with no live
region anywhere, so a screen reader is told nothing when a download finishes or a model
loads. Errors are `alert()` and one untagged message div. The stylesheet is 241 lines
with two media queries, both `prefers-color-scheme`, and not one responsive breakpoint.
No test asserts anything about accessibility. "Intuitive" and "state of the art" cannot
fail as written; the only falsifiable content is the three-part stance, and one third of
it — nothing interrupts the task — is directly refuted by `alert()` and by the two-click
confirm arming. The acceptance criteria should carry a small number of mechanical bars:
keyboard reachability of every control, a defined focus order, a live region for state
changes, a programmatic label per control, a visible focus ring. Those are testable;
"intuitive" is not.

**Nothing the press release promises needs a framework, and the evidence for that is
every defect above.** Aria attributes, a tab list, a live region, focus management,
responsive CSS, per-field error slots and a validation seam are plain HTML, plain CSS,
plain script and one Go handler. The panel is served as an embedded file system by a
file server; its weaknesses are authoring weaknesses, not architectural ones. Worth
naming so it is not mistaken for the conservative option: server-rendered templates are
a worse fit than they sound, because the panel is a live view — an event stream, a
re-render per snapshot, a two-second statistics poll — and a rendered page is a snapshot,
so the live behaviour would have to be re-added as the script that already exists, on top
of a template layer that has broken the sync guard on the way. There is also a seam the
repository has half-built and never used: `config.SamplingBounds` already returns a
machine-readable bound table and is served to nothing. Serving the rules from Go is the
direction that makes as-you-type feedback cheaper rather than more duplicated.

**Two things no option supplies.** There is no documentation page about the control
panel — twenty-five pages under `docs/` and none is about it — so a redesign that
re-homes settings by task changes where every existing page's instructions point, and
the documentation sweep belongs in the acceptance criteria or it goes stale the day this
lands. And the redesign re-opens a placement decision already taken: itd-2609091903463596
records that the resources block sits at the head of the Models tab rather than in a
seventh tab, decided at interview on 2026-09-09. A redesign organised by task will want
to move it; that is the maintainer's to re-take, not the redesign's to assume.

### Record discipline

**The frontmatter contradicts the ledger.** `builds_on` is empty. The DECISIONS.md line
of 2026-09-10 that created this draft says it refines itd-2609081259493890,
itd-2609081718534201 and itd-2609091903463596, and reverses nothing. An auditor reading
the frontmatter concludes this intent stands alone; an auditor reading the ledger
concludes it refines three shipped intents. `builds_on` is a planning-gate field, so this
review does not write it — the planning interview sets it to those three ids, or the
ledger line is corrected.

**Open Question 2's precondition expired.** It waits on three records landing. All three
are in `shipped/`: the sync test exists and is
`internal/archtest/settings_surface_test.go` with its four tests and its two-entry
exemption table, the resources block shipped, and the usage measurement shipped. A
question that reads as blocking when it is not is a question that blocks, so the waiting
clause goes and the live half — what is re-homed rather than rebuilt — stays.

**Open Question 2 also names a tab that does not exist and drops one the ledger named.**
There is no Usage tab. The panel has six: My Models, Find Models, Statistics, Connect,
Posture, Settings. The usage dashboard is the Statistics tab; the record id is right and
the tab name is wrong, which is the kind of error that survives into a spec. In the other
direction, the ledger names itd-2609081718534201, the posture page, among the pieces this
redesign reorganises, and the question does not mention it — although the posture page is
exactly the surface "shows what is on before it asks what to change" is about. The
draft's list and the ledger's list diverge in both directions.

**Open Question 3 was decided on 2026-09-10.** The same ledger line records that the
design stance — state before settings, facts not warnings, nothing interrupts the task —
stays in the draft's press release rather than becoming a principle. The question re-asks
it. Either it is stale, or the draft is deliberately re-opening a recorded decision, in
which case it must say so and say what changed; a decision silently re-asked is one that
can be answered two ways in two places. There is also a practical obstacle: CLAUDE.md
says the OPINIONS domain points at conventions under `.abcd/development/principles/`, and
that directory does not exist, so there is nowhere a principle could be written. Captured
as iss-2609190029331004.

**Severity and impact are understated.** This is `severity: minor`. itd-2609081259493890
— one architecture test, one checkbox and one read-only terminal verb — is
`severity: major`. This draft proposes rebuilding the only interactive settings surface
the product has, a surface that eight `internal/ui` test files and the three-surfaces
guard are string-coupled to, and whose authoring format that guard cannot survive a
change to. Recommend `severity: major`. `impact: additive` is defensible on the letter,
since no stored format changes, but it holds only if the scope conditions say so
explicitly, because a redesign that re-homes every control invalidates the navigation
instructions in `docs/` and in `README.md`.

**One intent, or a trunk with children.** The store supports it — the binary carries
`trunk` and `child` in its intent-kind vocabulary, and the schema has `kind`,
`suggested_kind`, `reclassification_history` and `builds_on` — but this repository has
never used it: of forty-nine intents, forty-one are `standalone` and eight are drafts
with `kind: null`. No trunks, no children. Choosing a trunk here means being the first,
which is a cost worth naming. The case for splitting is that the title carries four
promises that ship and fail independently: state shown before settings are asked for;
settings found by task rather than by file key; refusal explained as she types; and the
constraint of no build step, no framework and nothing fetched off the Mac. The third has
nothing structurally in common with the second — it is a new control-plane route on a
declared trust boundary with its own security review. The case against is that the fault
this intent exists to fix is incoherence, and a child per tab delivers exactly the
piecemeal result the intent is a complaint about; the decomposition has also been done
once already, by the maintainer, and produced this draft plus the build-step question
carved out as a separate decision. This review's recommendation: keep it `standalone`,
and carve out the as-you-type refusal as a sibling `standalone` intent that this one
`builds_on`. It is the only half with a trust boundary, the only half whose feasibility
is genuinely uncertain, and it can ship into the panel as it stands, which means its
value is not hostage to the redesign's scope.

**itd-1: the draft is four sections short, and correctly cannot be planned.** Press
Release is the seed stub, Why This Matters is the title pasted back, and Mechanism,
Scope Conditions and Acceptance Criteria are unanswered prompts. The consequence that
matters most: a redesign with no scope conditions has no boundary, and "finds each
setting by the task it serves" is unbounded over six tabs, more than twenty-five settings
and a per-model map.

**Personas are clean.** Alice, Bob and Carol, each doing the job the conventions give
them, no fourth name, and the maintainer is not referred to in the third person anywhere.

### Captured while reviewing

- iss-2609190029273153 — the panel's two as-you-type hints are unbound copies of Go
  rules, and the test for one of them restates the rule rather than binding to it, so
  both copies can drift together and stay green. Directly load-bearing here: generalising
  as-you-type feedback by writing more script rules multiplies exactly the drift the
  three-surfaces guard exists to police.
- iss-2609190029331004 — CLAUDE.md names `.abcd/development/principles/`, which this
  repository does not have, so the OPINIONS domain points at nothing and a written
  principle has nowhere to go.

## Open Questions

The three below are the maintainer's to answer: the first is a dependency sign-off, the
second re-opens a placement decision taken at interview, and the third is theirs to
write or to decline. Each carries the options as this review understands them and the
recommendation it would defend. None of them is decided here.

### 1. May the panel take a build step, a UI framework, or any asset fetched from outside the Mac?

**Answered 2026-09-20 — option A.** No build step, no framework, nothing fetched to
render. Recorded in `.abcd/work/DECISIONS.md` (2026-09-20) and carried as the first
scope condition above, where the offline property is stated as the testable form.

Opened as a decision to take before this draft is planned (DECISIONS.md, 2026-09-10). It
is two decisions in one — a dependency sign-off, which AGENTS.md requires before any new
dependency, and a boundary decision, since an asset fetched from a content network is a
new outbound host and the page is served from a Mac that may have no internet at all.
The review above adds a third cost the original framing did not carry: the repository's
own three-surfaces guard is a string scan over `index.html` and `app.js`, coupled to
their paths, to the literal `$('id')` lookup form, to `id="…"` as literal markup, to
three named source markers, to the `tab-settings` section boundary and to snake-case
object keys. Every option but the first breaks all six and obliges a replacement guard at
least as strong.

- **A. No build step, no framework, nothing fetched.** Plain markup, stylesheet and
  script in `internal/ui/static`, as today. Cost: every improvement is hand-written.
  Benefit: no dependency, no sign-off, nothing to rewrite, the guard and the node tests
  keep working unchanged, and the page renders offline by construction.
- **B. A build step, no framework** — a bundler, a minifier or a template pre-processor,
  vendored. Cost: a new dependency needing sign-off, plus the scanned source ceasing to
  be the served source, which breaks the guard and the node tests. Benefit: modest, since
  the panel is one page and three files.
- **C. A framework with a build step.** Cost: B's costs plus a rewrite of the panel, and
  the guard has to be rebuilt against a compiled artefact rather than source. Benefit:
  component structure and state binding that the current script hand-rolls.
- **D. Server-rendered Go templates with progressive enhancement.** Cost: the panel is a
  live view — an event stream, a re-render per snapshot, a two-second poll — so the live
  behaviour returns as the script that exists now, on top of a template layer that has
  already broken the guard. Benefit: rules served from Go rather than copied into script,
  which is the one real advantage, and it is available under A as well by serving the
  bounds the way `config.SamplingBounds` already computes them.

**Recommendation: A.** Not as a compromise but on the evidence: every defect this review
found — no aria, no tab list, no live region, no focus management, no responsive layout,
no per-field error slots, no validation seam — is fixable in the files as they stand,
with no dependency and no toolchain, and none of the press release's promises requires
otherwise. Choosing A also keeps the guard that makes the redesign safe to attempt.

### 2. What does the redesign re-home rather than rebuild?

**Answered 2026-09-20 — option A.** An inventory in the spec, with the resources
block staying at the head of the Models tab. Recorded in `.abcd/work/DECISIONS.md`
(2026-09-20) and carried as the first and second acceptance criteria and the second
and third scope conditions above.

The waiting clause this question carried is satisfied: itd-2609081259493890 (the sync
test), itd-2609091903463596 (the resources block) and itd-2609091712141073 (the usage
measurement) are all shipped, and the sync test exists. The live question is how the
re-homing obligation is armed, and it is a real one, because the guard that exists proves
a setting is *named* by the pane and says nothing about where it went or whether the
control still works.

- **A. An inventory in the spec.** Every existing control, roll-up block and read-only
  line named, with its destination in the new arrangement, and anything deliberately
  dropped listed with a reason. Cost: a long spec section and an argument per row.
  Benefit: the redesign cannot lose a control silently, and the inventory is what a
  reviewer diffs the delivered panel against.
- **B. Prose instruction plus the existing guard.** Say "re-home, do not rebuild" and
  rely on the sync test. Cost: the guard is a string scan over the Settings pane, so it
  catches a setting that vanished entirely and misses one moved somewhere nobody looks,
  demoted to a read-only line, or left with a control that posts the wrong value — which
  is the failure the sync intent's own mechanism claim names as the one enumeration
  passes green over.
- **C. Freeze what was decided at interview and re-home only Settings.** The resources
  block stays at the head of the Models tab, the posture page stays as it is, and the
  redesign's scope is the Settings pane. Cost: the press release's "shows what is on
  before it asks what to change" is mostly about the surfaces this option freezes.
  Benefit: the smallest change that delivers the settings-by-task promise.

**Recommendation: A, with the placement decision re-taken explicitly.**
itd-2609091903463596 records that the resources block sits at the head of the Models tab
rather than in a seventh tab, decided at interview on 2026-09-09; a redesign organised by
task will want to move it, and that is a decision to take again rather than to assume.
Two corrections belong in this question whichever option wins: there is no Usage tab —
the usage dashboard is the **Statistics** tab — and itd-2609081718534201, the posture
page, belongs in the list of what is reorganised, because the ledger names it and because
it is the surface the "shows what is on" promise is about.

### 3. Does "intuitive" need a written principle?

**Answered 2026-09-20 — option C.** No principle is written; the stance becomes
falsifiable acceptance criteria plus mechanical accessibility bars, and the
as-you-type refusal is carved out as itd-2609200823520756, which never gates.
Recorded in `.abcd/work/DECISIONS.md` (2026-09-20) and carried as the ordering,
facts-not-warnings, no-modal-interruption, keyboard, tab-strip, live-region,
label, focus-ring and Save-stays-live criteria above.

DECISIONS.md, 2026-09-10, already answered this once: the stance — state before settings,
facts not warnings, nothing interrupts the task — stays in the draft's press release
rather than becoming a principle. Re-asking it is either stale or a deliberate
re-opening, and the draft does not say which. There is also nowhere to write one:
CLAUDE.md points the OPINIONS domain at `.abcd/development/principles/`, which does not
exist (iss-2609190029331004).

- **A. Leave it where 2026-09-10 put it.** The stance stays press-release prose in this
  intent. Cost: it binds nothing beyond this intent and can fail no test; "intuitive" and
  "state of the art" remain words that cannot be wrong, and one third of the stance is
  already contradicted by the panel's `alert()` calls. Benefit: no new surface, no
  re-opened decision.
- **B. Write it as a principle.** Cost: the principles store has to be created first, and
  a principle binds every future surface including the Swift client, which is a wider
  commitment than this redesign needs. Benefit: the stance outlives this intent.
- **C. Convert the stance into mechanical acceptance criteria in this intent.** Each third
  of it becomes something a test can fail: state before settings as an ordering assertion
  on the pane; facts not warnings as a rule about the refusal and notice text; nothing
  interrupts the task as the removal of modal interruption, which is a concrete change
  since `alert()` is what the panel uses today. Alongside them, the mechanical
  accessibility bars — every control keyboard-reachable, a defined focus order, a live
  region for state changes, a programmatic label per control, a visible focus ring — which
  are the only testable content "state of the art" has.

**Recommendation: C, which keeps A rather than replacing it.** The 2026-09-10 decision
stands: the stance stays in the press release and no principle is written. What this
review would add is that the stance be made falsifiable where it applies, in this intent's
acceptance criteria, because a design promise that no test can fail is the one kind of
promise this repository does not otherwise accept. If the maintainer prefers B, iss-2609190029331004
is the precondition.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

**Answers 2026-09-20:** Q1 A; Q2 A with the resources block staying at the head of Models; Q3 C, as-you-type carved out as a sibling intent that never gates; severity major. Recorded in `.abcd/work/DECISIONS.md`.
