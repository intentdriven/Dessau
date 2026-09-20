---
id: spc-2609201007366798
slug: alice-is-told-why-a-value-cannot-be-saved-as-she-types-it-in
intent: itd-2609200823520756
origin: researcher-authored
production_mode: hand-written
---

# Told why a value cannot be saved, as it is typed, in the server's own words

## Summary

This spec delivers itd-2609200823520756: a field-tagged refusal carried from
`internal/config` through `internal/app` to `internal/gateway`; a split of
`app.SetConfig` into a pure `Check` and an `Apply`, so the budget and
pinned-fit refusals become askable without saving; one new loopback-only
control-plane route, `POST /api/settings/check`, which validates the posted
document against a **redacted** clone of the stored one, mutates nothing and
answers with the refusals as JSON, each naming the `config.json` keys it is
about; a second, static route serving the bound table so the panel's
as-you-type hints stop being copies of numbers Go already holds; and, on the
page, a per-field error slot beside each control, filled on a debounced input
event and **advisory only**. **Save** stays live, no field is marked invalid,
and the answer to the real `POST /api/settings` remains the only authority on
refusal. The refusal prose itself does not change: what is added is a key
beside it.

`impact: additive`. No stored format changes: no new `config.json` field,
nothing new in what a save writes.

## Scope

In: `internal/config` (the field-tagged error type; `Validate` and
`Sampling.Validate` tag what they refuse; `SamplingBounds` gets a reader at
last); `internal/app` (`Check` split out of `SetConfig`; the budget and
pinned-fit refusals tagged); `internal/gateway/control.go` (the two new
routes, the redacted-clone check, the narrowing of a tagged refusal through
`changedSettings`, the rate and size bounds); `internal/ui/static`
(`index.html` error slots, `app.js` the debounced check and the render,
`style.css` the slot's one rule); `internal/ui` and `internal/gateway` tests;
`internal/archtest` (the check path writes nothing); `docs/control-panel.md`,
new; the changelog.

Out: everything under **Out of scope** below — principally the re-homing of
controls, which is itd-2609100519003748's.

## Approach

### 1. A refusal carries the key it is about

`internal/config` gains one error type, beside the `InvalidError` it already
has:

```go
// FieldError is a refusal that names the config.json keys it is about.
type FieldError struct {
	Fields []string // spelled as config.json spells them
	Err    error
}
func (e *FieldError) Error() string { return e.Err.Error() } // the prose is unchanged
func (e *FieldError) Unwrap() error { return e.Err }

// Fields returns the keys a refusal names, or nothing when it names none.
func Fields(err error) []string
```

`Error()` returns exactly today's sentence, so every message an operator has
ever read is the message they still read, every test asserting refusal prose
stays green, and the flat OpenAI envelope `POST /api/settings` answers with is
byte-identical. What is added is a key nobody had to parse prose to get.

`Config.Validate` stays fail-fast and returns one refusal; each `return` in it
is wrapped in a `FieldError` naming its keys — `port`, `host`, `bind_mode`,
`decode_concurrency`, `idle_threshold_sec`, `api_key`, `preload`, `tls_port`,
`clients`, `models`, `chat_rule`, `stats_months`, `stats_max_bytes`,
`log_level`, `max_resident_bytes`, the grace pair, and `sampling` from
`Sampling.Validate`, which already names its parameter in prose and now names
it in a field too. A cross-field rule names the whole set it read: the
eviction-grace refusal names `eviction_grace` **and** `api_key`, because it is
a sentence about both.

Naming the set rather than one key is what lets the gateway do the thing
`refusalNamingWhatChanged` already does structurally: narrow to the fields
**this document changed**. The attribution the panel renders is
`Fields(err) ∩ changedSettings(before, after, posted)`; when that intersection
is empty the refusal is about nothing the operator touched, and it is shown in
the pane rather than beside a control — which is the honest answer, and the
same wedge-avoidance rule the save path already follows.

### 2. `app.SetConfig` splits into `Check` and `Apply`

`SetConfig` today validates, canonicalises the per-model map, checks the
budget against this Mac, checks the pinned set against the budget, persists,
and then applies eleven things live, under `saveMu`. The first half is the
save predicate; the second half is the save. They separate cleanly:

```go
// Check is the whole predicate of a save, and performs none of it.
func (a *App) Check(c config.Config) error
// SetConfig is Check followed by the persist and the apply, under saveMu.
func (a *App) SetConfig(c config.Config) error
```

`Check` runs `c.Validate()`, `canonicalModels`, `checkBudgetFitsTheMachine`
and `checkPinnedFit`, in that order, against the settings in force read
through `a.Config()`. `SetConfig` calls `Check` under `saveMu` and is
otherwise unchanged, so there is exactly one predicate and it cannot drift
from the one a save uses — the property ac-7 is about.

Two details are load-bearing. `checkPinnedFit` **logs a warning** on the
inherited case it declines to refuse; that line belongs to a save, not to a
keystroke, so the logging moves to `SetConfig` and `Check` returns a tri-state
the caller reports or swallows. And `Check` takes no lock but the config
read-lock: a keystroke must never queue behind a save, nor make a save queue
behind it.

### 3. The check route

```
POST /api/settings/check   → {"ok": bool, "refusals": [{"fields": [...], "message": "..."}]}
GET  /api/settings/rules   → the bound table
```

Both are registered in `Control.Routes`, so both sit behind `loopbackOnly`
and `admitToControlPlane`; `isPanelNavigation` admits no path under `/api/`,
so neither can be reached by a link a page put in front of anybody.

The check handler:

1. Reads at most `config.MaxConfigBytes` through `http.MaxBytesReader`, the
   bound the save path already uses; a larger body is refused in the save
   path's words.
2. Spends a token from a bucket sized for a person typing — burst 8, refilled
   4 a second — and answers `429` in the OpenAI envelope when it is empty. The
   bucket is one counter for the plane, because this plane has no caller
   identity to key on; that is the accepted cost of ADR-2609091123526871 §7
   restated, not a gap this route introduces.
3. Takes **`redactConfig(c.App.Config())`** as its base — the same clone the
   panel is already served by `GET /api/settings` — clones it, and decodes the
   posted body into the clone exactly as `applySettings` does, `models`
   emptied when the body names it, `Clients` restored from the clone, and the
   posted secrets handled by the rule below.
4. Calls `c.App.Check(incoming)`, narrows any `FieldError` through
   `changedSettings(base, incoming, raw)`, and writes the answer.
5. Writes nothing, applies nothing, clears no notice, touches no lock but the
   config read-lock.

**The secrets.** A posted secret equal to the placeholder is left as the
placeholder the redacted clone already carries; any other posted value is used
as posted. The real `api_key`, `hf_token` and `discord_token` are never read
on this path, so there is nothing on it to compare a guess against. What the
redacted clone discloses is precisely what `GET /api/settings` discloses
today — that a secret is set, never what it is — and the only refusal that
reads a secret at all, the eviction-grace rule, asks whether it is empty, which
the placeholder answers the same way for every stored value. The response is
therefore identical whether or not a posted secret happens to equal the stored
one, and ac-2 is the regression test that keeps it identical.

**The rules route** answers with `config.SamplingBounds()` and the named
numeric bounds beside it — the grace pair, the statistics pair, the ports, the
idle threshold, the decode floor — derived from the Go constants and nothing
else. It carries no fact about this install, so it is static for the build and
the panel reads it once at load. This is a third seam beside the two the
2026-09-19 decision bound the existing hints by, and it is deliberately not
the snapshot: that decision's falsifier is a `defaults` object that grows past
the two figures the panel must state before a save, and a bound table on a
snapshot re-encoded every couple of seconds is exactly that growth.

### 4. The panel

Each control in the Settings pane gains a sibling `<p class="fielderr"
id="err-<controlId>" hidden>`, and the script keeps a map from `config.json`
key to slot. On `input` or `change` the script debounces 300 ms, keeps one
check in flight and coalesces what arrives while it is out, posts the same
body the submit handler builds, and writes each refusal's message into the
slot of every field it names; a refusal naming none is written to a new
advisory line at the head of the pane, never into `settingsMsg`, which is the
save's own answer and is not overwritten by a keystroke. A clean answer clears
every slot.

Advisory means, concretely and testably: the script contains no
`setCustomValidity`, no `reportValidity`, no assignment of `disabled` on the
submit control, no `aria-invalid` written from a check answer, and no path in
the settings submit listener that returns before `api('/api/settings', …)` is
called. The single `e.preventDefault()` that hands the form to `fetch` stays —
it is how the form has always posted, and it refuses nothing.

The two existing hints, `graceWaitHint` and `budgetHint`, keep their seams
(sent defaults; `app.BudgetChargeNote`) and gain nothing; the numbers the
numeric controls state come from the rules route, so the panel stops carrying
copies of bounds Go holds — the class of drift recorded as
iss-2609190029273153.

### 5. The docs

`docs/control-panel.md`, a **reference** page: what the panel checks as you
type, that the check is advisory, that **Save** is always live and the server
is the only authority on refusal, that the check is reachable from this Mac
alone, and a row apiece for `POST /api/settings/check` and
`GET /api/settings/rules` with what each answers. It is the control-panel page
itd-2609100519003748 owes; that intent extends this page rather than opening a
second one. Present tense, British English.

## Acceptance-criteria map

One row per criterion of itd-2609200823520756, in the intent's order. Every
test is watched to fail before the change and to pass after; a hand check is
named as a hand check and recorded in the shipping decision line.

| # | Criterion | Held by |
|---|---|---|
| ac-1 | A check changes nothing | `TestAValidateCallChangesNothing` (`internal/gateway`): a check over a temporary configuration directory, asserting every file byte-identical by hash, no process started or stopped, and the resident set, notices and bind plan unmoved. Plus `TestTheCheckPathWritesNothing` (`internal/archtest`), which scans the check handler and `App.Check` for `config.Save(`, a write onto the configuration path, and the `Pool.Set*`/`apply*` calls — the structural half, in the shape of `TestNoLifecycleVerbWritesTheSettingsFile`. |
| ac-2 | The secret-equality oracle is shut | `TestTheCheckRouteAnswersTheSameWhateverTheSecretIs` (`internal/gateway`): a stored key, a document guaranteed to be refused, and the key posted beside it — once equal to the stored value, once not — asserting byte equality of the two responses, and that neither names a secret. The direct regression test for the finding in itd-2609081259493890's audit notes. |
| ac-3 | Per-field attribution names what changed | `TestACheckAttributesARefusalToTheFieldThisDocumentChanged` (`internal/gateway`): a table over the cross-field refusals — pinned fit, the memory budget, the eviction-grace-needs-a-key rule — asserting the answer names the changed field and not the unchanged ones the rule's wording is about. Built on `changedSettings`, beside `TestACrossFieldRefusalNamesAChangedField`. |
| ac-4 | The sentence appears beside its control | `TestARefusalIsDrawnBesideItsOwnControl` (`internal/ui`, node-lifted over the render function) and `TestEveryNumericSettingsControlHasAnErrorSlot` (`internal/ui`, over the markup, enumerated the way `TestEveryNumericSettingsControlIsProbed` enumerates). The second also asserts nothing from a check reaches `settingsMsg`. |
| ac-5 | The words are the server's | `TestThePanelComposesNoRefusalProseOfItsOwn` (`internal/ui`): the script carries no refusal sentence of its own for the fields this route covers, and no numeric bound for them either — those come from the rules route, held by `TestTheRulesRouteServesTheBoundsGoHolds` (`internal/gateway`), which compares the served table with `config.SamplingBounds()` and the named constants. |
| ac-6 | **Save** stays live | `TestNoSettingsControlIsNarrowerThanValidate` green **and unchanged** (no bound, exemption or probe edited), plus the new `TestNoScriptPathBlocksTheSettingsSubmit` (`internal/ui`): no `setCustomValidity`, no `reportValidity`, no `disabled` written to the submit control, no `aria-invalid` written from a check answer, and no return in the settings submit listener before the save is posted. This is the test the existing one cannot be: attributes say nothing about script. |
| ac-7 | The check half refuses without saving | `TestCheckReturnsTheBudgetAndPinnedFitRefusalsWithoutSaving` (`internal/app`): a budget under the pinned set and a budget over this Mac, each asserted refused by `Check`, with the configuration file byte-identical afterwards and the pool's budget and pinned set unmoved. |
| ac-8 | One docs page | `docs/control-panel.md` — the page itself, one Diátaxis type, plus `abcd docs lint` clean and a line in the shipping decision line naming it. A hand check, named as one. |

Scope conditions, and what holds each:

| cond | Held by |
|---|---|
| cond-2609201007360317 (advisory only) | ac-6's pair of tests. |
| cond-2609201007365487 (loopback plane, unchanged reach) | Registration in `Control.Routes` behind `loopbackOnly`; a test that both new routes are refused for a non-loopback remote address and for a cross-site fetch, beside the existing plane tests. |
| cond-2609201007369976 (purity) | ac-1's behavioural and structural tests. |
| cond-2609201007366101 (security review precedes landing) | The **Security** section below; the review's verdict recorded before the lane arms auto-merge. |
| cond-2609201007367527 (ships into the panel as it stands) | The slots are added beside the controls where they are today; no control moves in this spec. |
| cond-2609201007369542 (`impact: additive`) | A test that `config.Config`'s json-tagged field set is unchanged by this work, and `TestEverySettingHasAPanelControlOrAnExemption` green with a two-entry exemption table. |

## Security

This adds an unauthenticated input surface, called on every keystroke, to a
plane every account on this Mac reaches without a credential
(adr-2609091123526871 §7, `docs/bind-address.md`). The reach is accepted and
unchanged; what is new is the frequency and the fact that the caller may not be
Alice. The design answers four things:

- **The oracle class.** The route reads no stored secret, so it has nothing to
  compare a guess against, and the redacted clone gives it nothing back. This
  is the same failure adversarial review found in `refusalNamingWhatChanged`
  before itd-2609081259493890 shipped; the shape that closed it there —
  decide from what the caller sent, never from what is stored — is the shape
  here, and ac-2 is its regression test.
- **Disclosure.** The response carries refusal prose and `config.json` keys.
  The prose is the server's own and is already reachable by attempting a save;
  the keys are the file's own names. Nothing about the models on this Mac,
  the paired clients or the secrets appears in it.
- **Work amplification.** Bounded twice: the body at `config.MaxConfigBytes`,
  the rate at burst 8 refilled 4 a second, answered `429`. The check takes no
  save lock, so a flood cannot delay a save or be delayed by one.
- **Write surface.** There is none, and `TestTheCheckPathWritesNothing` is
  what keeps it none as the code moves.

**The `security-reviewer` agent must review this change before it lands** —
`internal/gateway` and `internal/config` are both declared trust boundaries in
AGENTS.md, cond-2609201007366101 makes the review a precondition rather than a
follow-up, and the precedent that makes it load-bearing is a real finding on
this exact code path. **This lane does not arm auto-merge until every must-fix
the review returns has been applied** and the verdict is recorded in the
shipping decision line. A queued PR is never pushed to again, so arming early
and fixing after is not available here.

## Verification gates

- `make test` green; `gofmt -l .` empty; `go vet ./...` clean.
- Every test named above watched red before the change and green after.
- `TestNoSettingsControlIsNarrowerThanValidate` and
  `internal/archtest/settings_surface_test.go` green **with an empty diff** —
  no path, marker, lookup form or exemption edited to accommodate the new
  markup.
- The node-lifted `internal/ui` tests actually run (node present), not skipped.
- `abcd docs lint` clean; `docs/control-panel.md` a single Diátaxis type.
- The security review run, its verdict and must-fixes recorded, before
  auto-merge is armed.
- Two hand checks, recorded in the shipping decision line: Alice types a
  budget below what her pinned models need and reads the sentence beside the
  field; she then presses **Save** with that sentence showing and the form
  posts, and the server refuses it in the same words.

## Out of scope

- **The re-homing of controls by task, the accessibility bars, the tab-strip
  work and the responsive layout.** All of that is itd-2609100519003748.
  What this spec owes that intent is nothing; what that intent owes this one is
  a per-field error slot beside every control once the controls move.
- Any gating: a disabled **Save**, an invalid field, a blocked submit.
- Any change to the control plane's reach or to bearer exemption. The ADR's
  accepted cost is restated, not revisited.
- Enumerating every refusal in one answer. `Validate` stays fail-fast and
  returns one refusal at a time; the response shape is a list so that a later
  change need not break it, and today the list holds at most one entry.
- Refusal prose changes, a new `config.json` field, migration of anything, and
  the Swift client, which never sees the server side.
