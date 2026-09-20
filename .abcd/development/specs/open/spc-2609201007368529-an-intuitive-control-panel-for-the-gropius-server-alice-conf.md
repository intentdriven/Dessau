---
id: spc-2609201007368529
slug: an-intuitive-control-panel-for-the-gropius-server-alice-conf
intent: itd-2609100519003748
origin: researcher-authored
production_mode: hand-written
---

# An intuitive control panel for the Dessau server

## Summary

This spec delivers itd-2609100519003748 by re-authoring the three files
already served from `internal/ui/static` — `index.html`, `style.css` and
`app.js` — with no build step, no framework and nothing fetched to render
(cond-2609201007368809). The panel keeps its seven tabs by name
(cond-2609201007367077); what changes is where a setting sits inside the
Settings pane, which task heading it sits under, what the markup says about
itself to a keyboard and a screen reader, and how a refusal or a destructive
action reaches Alice. The centre of the record is the inventory below: every
control, roll-up block and read-only line on the panel today, with its
destination, and the short list of what is deliberately dropped. Nothing here
adds a route, a setting, a stored format or an outbound host.
Impact: additive.

Two obligations sit beside each other and must not be confused. The per-field
error slot is this intent's: the markup gains a slot beside every Settings
control and a script path that can address it. Filling that slot as Alice types
is the sibling's — itd-2609200823520756, specced as spc-2609201007366798 —
and in both intents **Save** stays live and the server remains the only
authority on refusal (cond-2609201007367883, itd-2609081259493890).

## Scope

In: `internal/ui/static/index.html`, `internal/ui/static/style.css`,
`internal/ui/static/app.js`; new test files under `internal/ui/` and one new
file under `internal/archtest/`; the new page `docs/control-panel.md` and the
sweep of the existing pages listed under **Documentation**; the README
sections listed there; the changelog.

Out: `internal/gateway` (no route is added, changed or removed),
`internal/config`, `internal/app`, the menu-bar app's own menus, the Swift chat
client, the validation route and the as-you-type message (the sibling's), and
every existing file under `internal/ui/*_test.go` and
`internal/archtest/settings_surface_test.go`, whose diff must be empty when
this lands (acceptance criterion 11).

## The inventory

This is the section acceptance criterion 1 names, and the table a reviewer
diffs the delivered panel against. It is complete for the panel as it stands
today: every element id in `internal/ui/static/index.html` appears in it, and
every control the script builds at run time is named in the row of the block
that builds it.

**How to read the columns.** *Element* is the id in the markup, or the builder
in `app.js` for a control that has no id of its own. *What it is* is one of
**control** (Alice acts on it), **roll-up** (a block of figures derived from
the snapshot), **read-only line** (a sentence derived from the snapshot), or
**container** (a box the script fills). *Key* is the `config.json` key the
submit body posts for that control, or `—` for anything that posts no setting.
*Destination* is where it sits in the new arrangement.

Criterion 1's test walks this table: every id in the Destination column must
exist in the delivered markup, and for every row with a key, the script's
binding for that key must reach that id.

### Header and chrome, on every tab

| Element | What it is | Key | Destination |
| --- | --- | --- | --- |
| `serverStatus`, `statusDot`, `statusText` | read-only line | — | Header, unchanged. `statusText`'s sentence is also written to the live region when it changes. |
| `recordingBadge` | control (an `<a href="#">` today) | — | Header, unchanged in place; becomes `<button type="button">` with its listener bound in the script. |
| `warnings` | container of roll-up banners | — | Above the tab strip, unchanged: the exposed-server-with-no-key warning stays where the Posture pane's prose says it is. |
| `setupBanner`, `setupSpinner`, `setupFailMark`, `setupStage`, `setupBlurb`, `setupErr` | roll-up | — | Above the tab strip, unchanged. The stage heading is written to the live region when it changes. |
| — (new) `live` | read-only line | — | New: one polite live region, after the header, visually hidden. |
| — (new) `tab-models-tab` … `tab-settings-tab` | control ids | — | New: an id per tab button so each pane can name its tab in `aria-labelledby`. |

### My Models (`tab-models`)

| Element | What it is | Key | Destination |
| --- | --- | --- | --- |
| `resources` | roll-up | — | Head of My Models, unchanged — the 2026-09-09 placement, re-affirmed by cond-2609201007367668 and held by a test that no element precedes it inside the pane. |
| `graceQueue` | read-only line | — | Directly under the roll-up. |
| `modelList` | container of model cards | — | Under the two lines above, unchanged. |
| card: state and pinned pills, `modelInfoLine`, `measurementText`, progress bar | read-only lines | — | Inside each card, unchanged. |
| card: `Cancel`, `Retry`, `Remove`, `Load`, `Unload`, `Measure now`, `Use this window` (`btn`) | controls | — | Card action row, unchanged; each gains an `aria-label` naming its model, set with `setAttribute`. |
| card: `Delete` (`confirmBtn`) | control | — | Card action row; the timed arming becomes the untimed two-step described under **Approach**. |
| `modelsEmpty` | read-only line with a link | — | Foot of the pane; its `<a href="#">` becomes a button. |

### Find Models (`tab-search`)

| Element | What it is | Key | Destination |
| --- | --- | --- | --- |
| — (new) `searchMachine` | read-only line | — | New id at the head of the pane. The sentence `renderSearch` writes today — this Mac's memory and free disk, and how many results were hidden as too large — is written here instead of into the results box, so the state comes before the control that acts on it (criterion 2) and Alice reads it before her first search rather than after. |
| `searchForm`, `searchInput`, its submit button | controls | — | Under the machine line, unchanged. |
| the two-route hint (`mlx-community`, `author:`, repository id) | read-only line | — | Under the search bar, with its wording unchanged. |
| `searchResults` | container of result cards | — | Unchanged. |
| result card: `Download`, `Downloaded`, `Downloading…` (`btn`) | controls | — | Unchanged; each gains an `aria-label` naming its repository. |

### Statistics (`tab-stats`) — the usage dashboard

There is no Usage tab and none is added: the usage dashboard is this pane
(cond-2609201007367077).

| Element | What it is | Key | Destination |
| --- | --- | --- | --- |
| `statsStoreLine` | read-only line | — | **Moves here from Settings**, to the head of the pane, outside `statsBody` and outside `statsOff`. The line describes the records this pane shows; it must stay readable with recording switched off, because records kept from an earlier spell survive the switch. |
| `statsClear` (**Clear records**) | control | — | **Moves here from Settings**, beside the line above, for the same reason. It posts `/api/stats/clear`, not a setting, so it carries no key and leaves the Settings enumeration untouched. Its `confirm()` dialog becomes the untimed two-step. |
| `statsOff` | read-only line with a link | — | Unchanged; its `<a href="#">` becomes a button. |
| `statsBody`, `statsTotals`, `statsModels` | roll-ups | — | Unchanged. |
| `statsTable`, `statsRows` and their hint | roll-up | — | Unchanged. |
| `statsRange` (+ its `<label for>`) | control | — | Unchanged. It changes a view, not a setting. |
| `statsHistoryBounds`, `statsHistoryEmpty`, `statsHistoryBusy` | read-only lines | — | Unchanged. |
| `statsHistoryBody`, `statsDaysTable`/`Rows`, `statsLatencyTable`/`Rows`, `statsSpreadTable`/`Head`/`Rows`, `statsSizesTable`/`Rows`, `statsOverridesTable`/`Rows`, `statsFootprintTable`/`Rows`, `statsHoursTable`/`Rows` and the memory hint | roll-ups | — | Unchanged, each in its `.scroll` wrapper, which becomes a labelled, focusable region (see **Accessibility**). |
| `statsSummaryBlock`, `statsSummaryNote`, `statsSummaryTable`, `statsSummaryRows` | roll-up | — | Unchanged. |
| `selftestHint`, `selftestBody`, `selftestTable`, `selftestRows` and their hint | roll-up | — | **Moves to the foot of the pane** from its head. The self-test reports what Dessau measured of its own accord; the traffic above it is what the server did for other people, and that is what an operator opening Statistics came for. |

### Connect (`tab-connect`)

| Element | What it is | Key | Destination |
| --- | --- | --- | --- |
| `endpoints`, and the `Copy` button on each row | roll-up, controls | — | Unchanged, at the head of the pane. Each `Copy` gains an `aria-label` naming its URL. |
| `curlExample`, `pyExample` | read-only blocks | — | Unchanged. |

### Clients (`tab-clients`)

The pane is kept as a pane of its own: the paired set is never written by a
settings save, and drawing it beside the settings form would invite one
(adr-2609182357322050, and the reason written into the markup today).

| Element | What it is | Key | Destination |
| --- | --- | --- | --- |
| `serverFingerprint` | read-only line | — | Unchanged, at the head of the pane. |
| `clients` and each row's name, fingerprint and times | container, read-only lines | — | Unchanged; every field stays `textContent`, never interpolated markup. |
| row: `Revoke` (`confirmBtn`) | control | — | Unchanged in place; the timed arming becomes the untimed two-step. Its accessible name names the fingerprint's client. |

### Posture (`tab-posture`)

| Element | What it is | Key | Destination |
| --- | --- | --- | --- |
| `posture`, holding the ten lines `reach`, `private`, `transport`, `panel`, `key-network`, `key-local`, `announce`, `log`, `stats`, `selftest` | read-only lines | — | Unchanged, and the pane's opening tag is preserved byte for byte (see **What the existing tests pin**). |

### Settings (`tab-settings`)

Every row below stays inside `<section id="tab-settings">`. What changes is the
group each control sits in, the group's name — the job it does, not the key it
carries — and the order of the groups. Each group is a `<fieldset>`, never a
nested `<section>`.

**A. Who can reach this server** (`group-reach`)

| Element | What it is | Key | Destination |
| --- | --- | --- | --- |
| `bindNotice` | read-only line | — | Head of group A, above the controls. |
| `setHost` | control | `host`, `bind_mode` (via `bindSelectBody`) | Group A. |
| the restart hint beside it | read-only line | — | Group A, after the select, wording unchanged. |
| `setPort` | control | `port` | Group A. |
| `setTLSPort` and its hint | control | `tls_port` | Group A. |
| `setAdvertise` and its hint | control | `advertise` | Group A; the hint stays immediately after the box. |
| `setKey` | control | `api_key` | Group A. |
| `genKey` | control | — | Group A, beside `setKey`. |

**B. How much of this Mac models may use** (`group-memory`)

| Element | What it is | Key | Destination |
| --- | --- | --- | --- |
| `budgetHint` | read-only line | — | Head of group B. |
| `setBudget` | control | `max_resident_bytes` | Group B. |
| `concHint` | read-only line | — | Group B, above `setConc`. |
| `setConc` | control | `decode_concurrency` | Group B. |
| `contextList` and its prose | container of controls | `models.*.served_context` | Group B — the served window is how a long-context model is made to fit this Mac, which is this group's job. |
| `pinList` | container of controls | `models.*.pinned` | Group B. |
| `pinBudget` | read-only line | — | Group B, above `pinList`. |

**C. How long a model stays loaded** (`group-residency`)

| Element | What it is | Key | Destination |
| --- | --- | --- | --- |
| `graceHint` | read-only line | — | Head of group C. |
| `setIdle` | control | `idle_timeout_sec` | Group C, with the comment explaining why it carries no bounds preserved. |
| `setGrace` and its prose | control | `eviction_grace` | Group C. |
| `setGraceSec` | control | `eviction_grace_sec` | Group C. |
| `setGraceWait` | control | `eviction_max_wait_sec` | Group C. |

**D. How models answer** (`group-answers`)

| Element | What it is | Key | Destination |
| --- | --- | --- | --- |
| `setTemp`, `setTopP`, `setTopK`, `setMinP`, `setMaxTokens` | controls | `sampling.*` | Group D. |
| `overrideList` | container | `models.*.sampling` | Group D. |
| `ovModel`, `ovTemp`, `ovTopP`, `ovTopK`, `ovMinP`, `ovMaxTokens` | controls | `models.*.sampling` | Group D. |
| `ovApply` (**Set override**) | control | — | Group D. |
| `mergeList` and its prose | container of controls | `models.*.merge_system` | Group D. |
| `setChatPipelines`, `setChatTags` and their prose | controls | `chat_rule.pipeline_tags`, `chat_rule.required_tags` (via `chatRule`) | Group D. |

**E. What Dessau measures when nobody is asking** (`group-idlework`)

| Element | What it is | Key | Destination |
| --- | --- | --- | --- |
| `setContextProbe` and its prose | control | `context_probe` | Group E. |
| `setIdleThreshold` | control | `idle_threshold_sec` | Group E. |
| `idleThresholdDefault` | read-only line | — | Group E, in the sentence after the field, unchanged. |
| `setSelfTest` and its prose | control | `self_test` | Group E. |
| `selfTestIdleFigure` | read-only line | — | Group E, inside that prose, unchanged. |

**F. What Dessau writes down** (`group-record`)

| Element | What it is | Key | Destination |
| --- | --- | --- | --- |
| `setStats` and the paragraph beside it | control | `statistics` | Group F; the paragraph is reproduced verbatim, including "**Recording** appears beside the server status at the top of this page". |
| `setStatsMonths` | control | `stats_months` | Group F. |
| `setStatsMB` and the records-kept prose | control | `stats_max_bytes` | Group F. |
| `setLogLevel` and its prose | control | `log_level` | Group F. |

**G. Reaching your models from elsewhere** (`group-elsewhere`)

| Element | What it is | Key | Destination |
| --- | --- | --- | --- |
| `discordState` | read-only line | — | Head of group G. |
| `setDiscordBridge` and its prose | control | `discord_bridge` | Group G. |
| `setDiscordToken` | control | `discord_token` | Group G. |

**H. Downloading models** (`group-downloads`)

| Element | What it is | Key | Destination |
| --- | --- | --- | --- |
| `setHF` | control | `hf_token` | Group H — the token's only job is fetching a gated model, which is the Find Models tab's job, and it is the only setting whose task is that. |

**The form itself**

| Element | What it is | Key | Destination |
| --- | --- | --- | --- |
| `settingsForm` | container | — | Unchanged. |
| its submit button (**Save settings**) | control | — | Foot of the form, last in the focus order, never disabled. |
| `settingsMsg` | read-only line | — | Beside the submit button; its text is also passed to the live region. |
| — (new) `err-<controlId>` per Settings control | read-only lines | — | New: an empty, hidden per-field error slot beside every control in groups A–H, addressable by the script. This intent supplies the slot; spc-2609201007366798 fills it. |

**Settings with no control, unchanged at two.** `preload` and
`upstream_header_timeout_sec` keep their written exemptions in
`internal/archtest/settings_surface_test.go` (cond-2609201007362808). No third
is added.

### Dropped

Nothing that is a control, a roll-up block or a read-only line is dropped. What
is dropped is five pieces of behaviour and markup:

1. **`confirmBtn`'s three-second timer.** A destructive action that disarms
   itself after three seconds is an interruption for anyone working the page by
   keyboard or screen reader, and a trap for anyone who reads before pressing.
   Replaced by the untimed two-step below (criterion 4).
2. **`alert()`, through `alertErr`.** A modal dialog interrupts the task and
   says nothing to the page. Replaced by the pane's error slot plus the live
   region (criterion 4).
3. **`confirm()` on Clear records.** Browsers let a person permanently suppress
   it, which silently turns a destructive button into a no-op — the reason
   `confirmBtn` exists at all. Replaced by the same untimed two-step
   (criterion 4).
4. **The three `<a href="#">` elements and their inline `onclick` attributes**
   (`recordingBadge`, the Find Models link in `modelsEmpty`, the Settings link
   in `statsOff`). An anchor that does a button's job is unreachable as what it
   is; the behaviour survives as a `<button type="button">` with a listener
   bound in the script (criterion 5).
5. **`form input:focus, form select:focus { outline: none; border-color: … }`.**
   A colour change on a border is not a focus indicator. Replaced by a
   `:focus-visible` ring on every focusable element (criterion 9).

## Approach

### The information architecture, by task

**All seven tabs remain, by name: My Models, Find Models, Statistics, Connect,
Clients, Posture, Settings.** None is renamed, none is merged into another,
none is removed and no eighth is added — cond-2609201007367077 settles that,
and the history test already asserts that no second statistics tab appears. The
re-homing therefore happens in three places: the order of blocks inside a pane
(Statistics, Find Models), two read-only pieces that move between panes
(`statsStoreLine` and **Clear records**, from Settings to Statistics, where the
records they describe are shown), and the Settings pane's eight task groups
A–H above.

**Two placements are kept deliberately, not by omission.** The resources
roll-up stays at the head of My Models (cond-2609201007367668), and the Clients
pane stays a pane of its own. Both are re-affirmed here so that a reviewer
diffing this spec against the delivered panel can tell a decision from an
oversight.

**Why the Settings pane's groups are the whole of the settings re-homing.**
Every numeric Settings control is probed by `TestNoSettingsControlIsNarrowerThanValidate`
and `TestEveryNumericSettingsControlIsProbed`, which slice the markup from
`<section id="tab-settings"` to the first `</section>` and look for each control
inside that slice; `TestTheAdvertisingControlSaysAChangeWaitsForTheNextStart`
does the same for the advertising box. A control moved to another tab
disappears from that slice and those tests fail — which would mean editing them,
which criterion 11 forbids. So settings are found by task *within* the Settings
pane, under headings that name the job, and the panel's other panes gain no
setting. This is a constraint discovered in the tree, not a preference, and it
is stated here so the delivered panel is not read as a half-finished
reorganisation.

### What the existing tests pin, and the three shapes they forbid

`internal/archtest/settings_surface_test.go` and the `internal/ui` tests must
be green and unchanged (criterion 11). They are string scans, so they pin the
authoring format in ways that are invisible from the markup. The redesign obeys
all of these:

- **The served files stay `internal/ui/static/index.html` and
  `internal/ui/static/app.js`**, and the scanned source stays the served source:
  no build step, no bundler, no template layer (cond-2609201007368809).
- **The element lookup stays the literal `$('someId')` form**, and every control
  the form posts keeps its key and its lookup on one source line.
- **The submit body keeps its marker `const body = {`, its helpers
  `bindSelectBody` and `chatRule`, its key order and its column alignment.**
  `settings_test.go` asserts the literal `advertise:          $('setAdvertise').checked,`
  — spacing included — so the body is re-indented by nobody.
- **The Settings pane holds no nested `<section>`.** The pane's slice ends at
  the first `</section>`; task groups are `<fieldset class="group">` elements.
- **`<section id="tab-posture" class="panel">` is preserved byte for byte.**
  `posture_test.go` asserts that exact opening tag, so the Posture pane cannot
  carry `role="tabpanel"` as markup. Every pane therefore takes its
  `role="tabpanel"`, `aria-labelledby` and `tabindex="-1"` from `initTabs()` in
  the script, uniformly, rather than six panes in markup and one in script.
  This is a deviation from the letter of criterion 6, which says the panes'
  roles are held by an assertion over the markup; the criterion's substance —
  a failing case an assertion can find — is kept by a node-lifted test over
  `initTabs`, and the tab strip's own roles are in the markup as the criterion
  says. Criterion 11 wins the collision because the alternative is editing a
  test to accommodate new markup, which is exactly what that criterion forbids.
- **`confirmBtn` and `alertErr` keep their names and their arity.** The node
  harness in `clients_test.go` stubs both by name; renaming either would edit a
  test file. `alertErr` no longer alerts, and says so in a comment above itself.
- **`<p class="hint">` paragraphs keep their class, their wording and their
  position relative to their control.** `stats_test.go` reads the request
  statistics paragraph as the first `<p class="hint">` after its legend, and
  `settings_test.go` reads the advertising hint as the first one after its box.
  A group's state line is therefore marked with a `data-state` attribute and
  placed at the head of the group, never between a control and its hint.
- **`renderPosture` writes into `#posture` and nowhere else, and `app.js`
  contains no `showTab('posture')`.** The new listeners bound for the three
  ex-anchors navigate to `search`, `settings` and `stats` only.
- **The node-lifted functions stay top-level `function name(...)` declarations**
  — `extractFunction` finds them by that shape — and the pure ones keep their
  names: `modelInfoLine`, `renderModels`, `renderSearch`, `renderClients`,
  `watchStats`, `renderHistory`, `graceWaitHint`, `bindSelectBody`, `chatRule`,
  `measurementText`, and the rest the tests lift.

### Accessibility as markup

**The tab strip.** `<nav class="tabs" role="tablist" aria-label="Control panel
sections">`, and each button becomes
`<button class="tab" id="tab-models-tab" data-tab="models" role="tab"
aria-controls="tab-models" aria-selected="true" tabindex="0">` — one tab stop
for the strip, roving `tabindex` over the seven, `data-tab` kept because three
tests read it. `showTab(name)` additionally sets `aria-selected` on every tab,
moves the roving `tabindex`, and moves focus to the newly selected tab. A new
`onTabKey(event, tabs, current)` — a pure function returning the index to
select — handles Left and Right with wrap-around, and Home and End. Escape does
nothing: there is nothing to escape from.

**The panes.** `initTabs()` runs once at load and gives every `.panel` element
`role="tabpanel"`, `aria-labelledby="<its tab's id>"` and `tabindex="-1"`, and
`showTab` focuses the newly selected pane so a keyboard lands inside the
content it just chose rather than back at the top of the document.

**One polite live region.** `<p id="live" class="sr-only" role="status"
aria-live="polite" aria-atomic="true"></p>`, sitting after the header, written
with `textContent` only. A new pure function `stateChangeSentences(prev, next)`
diffs two snapshots and returns the sentences for the changes Alice did not
make on the page: a download finishing or failing, a model loading, unloading
or being evicted, a measurement completing, the setup stage changing, the
server connection dropping or returning, a client pairing or being revoked.
`render()` calls `announce(stateChangeSentences(previous, state))`; `announce`
is also what the settings save outcome, the refusals that used to be `alert()`
and the two-step confirmations pass through, so the page has one region and not
five.

**A programmatic label per control.** Every `<input>`, `<select>` and
`<textarea>` in the markup takes an explicit `<label for="…">`; the implicit
wrapping labels the form uses today become explicit ones with the same words.
Controls the script generates — the pin boxes, the merge boxes, the served
context fields, the override rows and the card action buttons — take an
`aria-label` naming the model or the repository, set with `setAttribute` and
never interpolated into a template literal.

**A visible focus ring.** `style.css` gains a `--focus` token in both colour
schemes and one rule:
`:focus-visible { outline: 3px solid var(--focus); outline-offset: 2px; }`.
The existing `form input:focus, form select:focus { outline: none; … }` rule is
removed. No rule in the stylesheet removes an outline without drawing a
replacement in the same rule.

**Anchors doing buttons' work become buttons.** The three named in the Dropped
list, with their inline `onclick` attributes replaced by listeners bound in
`app.js`. No element carries a positive `tabindex`.

**The focus order, defined** (criterion 5), is document order with three
additions: the **Recording** button when it is shown, then the tab strip as a
single tab stop, then the selected pane (which receives focus on a switch but
is not in the tab order), then that pane's controls in document order. Inside
Settings that is groups A, B, C, D, E, F, G, H and then **Save settings**, which
is last. Each `.scroll` wrapper around a wide table becomes
`<div class="scroll" role="region" tabindex="0" aria-label="…">`, so a keyboard
can scroll it and a screen reader announces what it is.

### No modal interruption

`alertErr` stops calling `alert()`. It writes the message into the error slot
of the pane the action belongs to — one `data-error` element per pane, plus the
per-field slots in Settings — and passes the same sentence to `announce()`. The
slot is cleared when the next action on that pane succeeds.

`confirmBtn(label, confirmLabel, cls, onConfirm)` keeps its name and arity and
loses its timer. The first press replaces the button, in place, with an inline
group: the question as text, a **Confirm** button which takes focus, and a
**Cancel** button. Escape or **Cancel** restores the original button and returns
focus to it. There is no `setTimeout` in the function and no time at which an
armed confirmation disarms itself; it waits for Alice. **Clear records** uses
the same function instead of `confirm()`.

### The per-field error slot

Each control in groups A–H gains a sibling
`<p class="fielderr" id="err-<controlId>" data-error-for="<controlId>" hidden></p>`,
and `app.js` gains one function, `fieldError(controlId, message)`, which writes
a message into that slot or clears it. This intent ships the slot, the function
and the empty state; it ships no caller that fills a slot as Alice types, no
validation route and no client-side judgement. **Save** is never disabled, no
field is marked invalid, and no submit is cancelled on anything the page
decided (criterion 13). spc-2609201007366798 is what fills the slot.

### A layout that fits a phone

One breakpoint, `@media (max-width: 480px)`, covering the three things
criterion 10 names: the tab strip wraps to a second row rather than scrolling
(`flex-wrap: wrap`); every wide table stays inside its `.scroll` region, which
scrolls within its own box so the document never scrolls horizontally, and
which is focusable so a keyboard can reach it; the Settings form's labels and
fields stack to full width. The header's brand and status wrap. Checked by hand
at 390 px, recorded in the shipping decision line as a hand check.

### The offline property, as a test

The property, stated so it can fail: **the page renders and every control works
with this Mac offline, and the only outbound host the panel carries is a link
an operator clicks.** `TestThePanelFetchesNothingFromOffTheMac` scans the three
static files for a subresource with a scheme — `<link href>`, `<script src>`,
`<img src>`, `@import`, `url(…)` in the stylesheet, `@font-face`, and any
`fetch`, `EventSource` or `XMLHttpRequest` target that is not a same-origin
path — and asserts there is none; then it asserts that every `https://`
occurrence that remains is an `<a href>`, that there are exactly six of them,
and that each names a page that exists under `docs/`. The six are
`posture-reference.md`, `context-probe.md`, `self-test.md`,
`statistics-store-reference.md`, `logging.md` and `discord-bridge.md`. A hand
check with this Mac's networking off, recorded in the shipping decision line,
is the other half.

### Documentation

**One new page, one Diátaxis type: `docs/control-panel.md`, a reference.** It
describes the panel as it is — the seven tabs, what each holds, and for the
Settings pane the eight task groups A–H with every control named under the
group it sits in. It gives no recipes: the thirteen pages below already own
those, and a page that is both a reference and a how-to is the thing criterion
12 forbids. It is the page the others link to for "where does this live".

**The sweep**, by file. Recipe pages whose navigation instructions this
re-homing changes, and which are edited:

`docs/getting-started.md` (eight `Settings → …` paths, plus the tour of My
Models, Find Models and Connect), `docs/request-statistics.md` (the
`Settings → Request statistics` path, and **Clear records**, which is now on
the Statistics tab), `docs/sampling-defaults.md`,
`docs/pinning-models.md`, `docs/context-probe.md`, `docs/eviction-grace.md`,
`docs/memory-budget.md`, `docs/system-message-merging.md`,
`docs/chat-models.md`, `docs/discord-bridge.md`, `docs/self-test.md`,
`docs/mesh-vpn.md`, `docs/pairing.md`.

Reference and explanatory pages checked, and edited only where they name a
group or a control's home: `docs/models-list.md`, `docs/bind-address.md`,
`docs/statistics-store-reference.md` (**Clear records**'s new home),
`docs/memory-budget-explained.md`, `docs/sampling-reference.md`,
`docs/posture-reference.md`, `docs/statistics-explained.md`,
`docs/pairing-explained.md`, `docs/lifecycle.md`, `docs/lifecycle-reference.md`,
`docs/response-headers.md`.

`README.md`: the features list where it names the Statistics tab, the Posture
tab and the Discord token's home, and the security section where it says a key
is set in Settings. The one instruction that says how to open the panel — the
menu-bar item — is unaffected.

Checked and unaffected: `docs/logging.md`, `docs/sampling-explained.md`,
`docs/self-test-reference.md`, `docs/eviction-grace-explained.md`. The sweep
list, with what was found on each page, goes in the shipping decision line.

## The acceptance-criteria map

The intent's criteria carry no minted ids in this store — only its scope
conditions do — so they are numbered here in the order the intent lists them.
Every test named below is new unless it says otherwise, lives in a new file, and
is **watched red against the panel as it stands today** before the markup
changes; the two hand checks are named as hand checks and recorded in the
shipping decision line, in the way the 2026-09-20 ledger line records two other
checks as owed.

| # | Criterion | Held by |
| --- | --- | --- |
| 1 | The spec carries the inventory, and the delivered markup matches it | This section's table, plus `TestThePanelMatchesTheSpecInventory` in a new file under `internal/archtest/` — it globs `.abcd/development/specs/*/spc-2609201007368529-*.md` so it survives the spec moving bucket, parses the Destination and Key columns, asserts every id exists in `index.html`, and for each row with a key asserts the script's binding for that key reaches that id, reusing `controlIDsBoundTo` from the package. |
| 2 | State comes before the controls that change it, per pane | `TestEveryPaneReportsBeforeItAsks`: in each pane, and in each `fieldset.group` inside Settings, the first `data-state` element precedes the first `<input`, `<select`, `<button` or `<textarea`. Blocks with no state line are listed in the test, the way the exemption table is written down. |
| 3 | Refusals and notices state what is, rather than warning | `TestThePanelsNoticesStateWhatIsRatherThanWarn`, modelled on `TestTheBindModeLabelsNameNoVendorAndPromiseNothing`: the refusal and notice strings in the markup and in the script's error paths carry none of the forbidden words that test already names. |
| 4 | No `alert()`, no `confirm()`, and no timed arming | `TestThePanelInterruptsNobody`: `app.js` contains no `alert(`, no `confirm(` and no `window.` dialog call. Plus the node-lifted `TestTheConfirmationIsTwoStepsAndNotTimed`: `confirmBtn`'s body contains no `setTimeout`, a second press calls the action, and Cancel restores the original button. |
| 5 | Every control keyboard-reachable, in the order this spec defines | `TestEveryControlIsKeyboardReachable`: no `<a href="#">`, no `onclick=` attribute, no positive `tabindex` in the markup. Plus `TestTheFocusOrderIsTheOneTheSpecDefines`: the eight Settings groups appear in document order A–H and the submit button is last. |
| 6 | The tab strip is a tab list with arrow keys | `TestTheTabStripIsATabList` over the markup (`role="tablist"`; each tab `role="tab"`, `aria-selected`, `aria-controls` naming an existing section id). Plus the node-lifted `TestEveryPaneIsGivenItsTabPanelRole` over `initTabs` and `TestTheTabStripMovesOnArrowKeys` over `onTabKey` and `showTab` (selection moves, focus follows). The deviation from "each pane `role=tabpanel` in the markup" is argued under **What the existing tests pin**. |
| 7 | State changes Alice did not initiate are announced | `TestThePanelHasOnePoliteLiveRegion` over the markup, plus the node-lifted `TestEveryUninitiatedStateChangeIsAnnounced` over `stateChangeSentences` for each of the named events, plus an assertion that `render` passes its result to `announce`. |
| 8 | A programmatic label per control | `TestEveryControlCarriesAProgrammaticLabel`, enumerating the markup the way `TestEveryNumericSettingsControlIsProbed` enumerates: a `<label for>` whose target exists, an `aria-label` or an `aria-labelledby`. Plus a node-lifted test that the generated rows and card buttons set `aria-label`. |
| 9 | A visible focus ring, never removed unreplaced | `TestNoRuleRemovesTheFocusRingWithoutReplacingIt` over `style.css`: no `outline: none` or `outline: 0` without a replacement indicator in the same rule, and a `:focus-visible` rule that draws one. |
| 10 | The layout reflows at phone width | `TestTheStylesheetReflowsAtPhoneWidth`: a `max-width` media query exists and names the tab strip, the scroll regions and the form's selectors. Plus one recorded **hand check at 390 px**. |
| 11 | The guard and the node-lifted tests are green and unchanged | `make test`, plus `git diff --stat` over `internal/archtest/settings_surface_test.go` and `internal/ui/*_test.go` being empty — a **hand check**, named as one, in the shipping record. Every constraint this depends on is listed under **What the existing tests pin**. |
| 12 | One documentation page, one Diátaxis type, and the sweep | `docs/control-panel.md` as a reference; `abcd docs lint` for links and stray markdown; the sweep list above, with what was found on each page, in the shipping record. |
| 13 | Save is never disabled and the page never gates a save | `TestNoSettingsControlIsNarrowerThanValidate` staying green (existing), plus `TestThePanelNeverGatesItsOwnSubmit`: `app.js` contains no `setCustomValidity(`, `reportValidity(` or `checkValidity(`, the markup carries no `novalidate`-driven substitute and no `disabled` on the submit button, and the settings submit listener contains no `return` between its `preventDefault()` and its `POST /api/settings`. |

## Security

`internal/ui` is not named in AGENTS.md's list of trust boundaries, but the
panel it serves sits on one: the control plane is loopback-only and exempt from
the bearer check, so every account on this Mac reaches it
(adr-2609091123526871 §7, `docs/bind-address.md`), and the page renders strings
chosen by whoever paired a client — which, under first-come pairing, is
anything on the network. A name that reached markup would run same-origin
against the whole settings API. The redesign is therefore reviewed by the
security-reviewer agent before it lands, on these points:

- **Nothing here adds a route, a method or a parameter.** The seventeen control
  plane routes in `internal/gateway/control.go` are untouched. The validation
  route, its purity argument and its own adversarial review belong to
  itd-2609200823520756 / spc-2609201007366798.
- **The accessibility work is the injection risk of this change.** An
  `aria-label` naming a model or a client is attacker-influenced text; every
  one is set with `setAttribute`, never interpolated into a template literal,
  and the existing rule that every client field is set with `textContent`
  stands unchanged. The live region is written with `textContent` only.
- **No new storage and no new host.** No `localStorage`, no cookie, no
  `sessionStorage`, no fetched subresource; the offline test above is what
  arms the last of those.
- **The error slots disclose nothing the page was not given.** They render the
  server's own refusal text and the panel's own sentences, and this intent
  ships no caller that sends a value anywhere to have it judged.

## Verification gates

`make test` green; `gofmt -l .` empty; `go vet ./...` clean; `abcd docs lint`
clean. The node-gated tests skip where node is absent, so the run that counts is
on a Mac with node installed, and the shipping record says so. The two hand
checks — the panel at 390 px, and the panel with this Mac's networking off —
are recorded as hand checks in the shipping decision line. The security review
above happens before the change is presented. Every new test is watched to fail
against the panel as it stands before the markup changes, and the shipping
record names the failure each one showed.

## Out of scope

The as-you-type refusal, its validation route and its security review
(itd-2609200823520756 / spc-2609201007366798 — this intent ships the empty slot
and nothing that fills it); the Swift chat client, which never sees the server
side; the menu-bar app's own menus; any new setting, any change to
`config.json`'s shape, and any third entry in the exemption table; multi-user
or remote administration (cond-2609201007366571); translation of the panel; and
a browser-driven test harness, which cond-2609201007368809 rules out by ruling
out the toolchain that would carry it.

## Falsifiers, from the Mechanism claim

A defect from the 2026-09-19 review that cannot be fixed without a toolchain; a
change to the redesigned page that makes `settings_surface_test.go` or any
`internal/ui` test need rewriting to stay green; a control that goes missing
while both the inventory test and the guard stay green; an accessibility bar
from criterion 5–9 that cannot be asserted without driving a real browser; a
redesign requirement that needs a fetched subresource to render. Any of these
reopens the intent rather than being patched around.
