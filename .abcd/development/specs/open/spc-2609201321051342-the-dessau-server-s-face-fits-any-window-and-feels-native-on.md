---
id: spc-2609201321051342
slug: the-dessau-server-s-face-fits-any-window-and-feels-native-on
intent: itd-2609201315575657
origin: researcher-authored
production_mode: dictated-and-formatted
---
# The Dessau Server's face fits any window and feels native on the Mac

## Summary

This spec delivers itd-2609201315575657 by re-authoring the appearance, the
layout, the keyboard contract and the script layout of the three files served
from `internal/ui/static`, and by holding the menu-bar menu and the terminal
verbs to the panel's vocabulary. The stack is unchanged in kind and changed in
shape: plain markup, one stylesheet and native ES modules, no build step, no
dependency, nothing fetched (cond-2609201321055222). The stylesheet gains
cascade layers, one `light-dark()` token block, container queries and a
reduced-motion contract; the markup gains the keyboard half of the APG tabs
pattern and one announcement region; `app.js` becomes a small set of modules,
one per tab plus three shared ones. The source-scanning guard in
`internal/archtest` is taught to read a directory of scripts, and a sibling
guard is added for the tabs contract, so the accessibility promise is armed the
way the surface-sync promise is. Impact: additive — nothing stored on disk
changes.

**What this spec does not respecify.** The information architecture is
spc-2609201007368529's (for itd-2609100519003748): the re-homing of controls,
the inventory of every control and read-only line, the removal of `alert()`,
`confirm()` and the timed arming, the three ex-anchors becoming buttons, the
per-field error slots, the `role="tablist"`/`role="tab"`/`role="tabpanel"`
role attributes and the one `role="status"` element in the markup. That intent
lands first (cond-2609201321050310) and this one takes its arrangement as
given. Where this spec touches the same surface it says so and adds only the
part criterion 4 asks for: the tabs pattern's **keyboard half**, and the rule
about **what may and may not write** into the announcement region that
spc-2609201007368529 places.

**One cross-intent interaction, stated so nobody trips on it.**
spc-2609201007368529's criterion 11 requires that the eight node-lifted
harnesses in `internal/ui` and `internal/archtest/settings_surface_test.go`
have an empty diff when *that* intent lands. This intent's criterion 5 rewrites
exactly those files. The freeze belongs to the panel intent's landing, not to
this one; the sequence in cond-2609201321050310 is what keeps the two from
colliding, and this spec's changes to those test files are the first edits they
take after that freeze lifts.

**Three corrections to the record, found in the tree while writing this.**
(1) The 2026-09-20 research note says the panel repaints through `innerHTML` in
82 places; the tree holds **33 occurrences, 31 of them assignments**, and no
`insertAdjacentHTML` or `outerHTML` at all. The smaller figure is the one the
renderer trigger in section (E) is reasoned from. (2) The stylesheet's palette
is not the mark's: it declares `--red: #e63946`, `--yellow: #f4d35e` and
`--blue: #1d63c4`, none of which is a triad value, while the header mark and
the favicon carry the triad directly. Section (A) is therefore a palette
replacement and not a palette rename. (3) The menu-bar items are built inside
the tray callback as local variables, so no Go test can reach the list at all
today; criterion 8 needs the lift described in section (F) before it has
anything to assert against.

## Scope

In: `internal/ui/static/index.html`, `internal/ui/static/style.css`, the
replacement of `internal/ui/static/app.js` by the modules named in section (C);
`internal/ui/*_test.go` (the eight harnesses, converted);
`internal/archtest/settings_surface_test.go` (the directory scan) and four new
files under `internal/archtest/`; the command package's `menubar.go` (the item
list lifted to a table) and one new test beside it; `internal/lifecycle`'s
`term.go` (one shared table helper), `status.go`, `doctor.go`, `config.go` and
new plain-text goldens under `internal/lifecycle/testdata/`;
`docs/control-panel.md` and the pages listed in section (H); `README.md`; the
changelog.

Out: `internal/gateway` (no route, method or parameter is added, changed or
removed), `internal/config`, `internal/app`, `internal/hub`,
`internal/runtime`, `internal/capability`, `client/` (Dessau Chat), and the
mark's own drawings, which spc-2609201011302811 owns. No setting is added,
renamed or removed, so the exemption table in
`internal/archtest/settings_surface_test.go` keeps its two entries and gains no
third.

## Approach

### (A) The stylesheet

`internal/ui/static/style.css` is re-authored as four cascade layers, declared
in one statement at the top of the file so the order is a fact about the sheet
and not about where a rule happens to sit:

```css
@layer tokens, base, components, utilities;
```

`tokens` holds nothing but custom properties. `base` holds the element rules —
`body`, headings, tables, form controls. `components` holds the panel's own
classes — `.card`, `.statcard`, `.override`, `.tab`, `.pill`, `.banner`,
`.fact`, `.endpoint`, `.scroll`, `.group`. `utilities` holds the handful of
overrides that must win, `[hidden] { display: none !important; }` and `.sr-only`
among them. A rule outside a layer beats every layered rule, so the file
carries none: the layer statement is what makes the sheet's cascade legible
rather than a matter of source order, and a rule that escapes it silently
un-does the whole arrangement.

**Colour scheme and the collapse of the dark block.** `:root` declares
`color-scheme: light dark`, which is what makes the form controls, the
scrollbars and the default `Canvas`/`CanvasText` follow the Mac's setting
without a rule per control. Every colour token is then a single `light-dark()`
value, and the sheet's two hand-maintained `prefers-color-scheme: dark` blocks —
the `:root` palette and the one that darkens `.code` — are deleted. There is
one palette, in one place, and a colour role that needs a different value in
dark mode says so on its own line rather than in a second block a hundred lines
away.

**The token vocabulary, named in full.** The names borrow the DTCG naming
discipline and none of its tooling: nothing crosses a tool boundary here.

*Colour roles.* `--surface` (the page's ground), `--surface-raised` (a card, a
fieldset, an endpoint row), `--surface-sunken` (a code block), `--ink` (body
text), `--ink-muted` (a secondary line, a table heading), `--ink-inverted`
(text on a filled control), `--line` (a hairline between rows), `--line-strong`
(a control's border), `--accent` (the selected tab, a link, a filled primary
button), `--accent-ink` (text on `--accent`), `--focus` (the focus ring),
`--selection` and `--selection-ink` (a text selection). Each is a
`light-dark()` pair built from the CSS system colours first — `Canvas`,
`CanvasText`, `Field`, `FieldText`, `GrayText`, `ButtonFace`, `ButtonText`,
`ButtonBorder`, `LinkText`, `Highlight`, `HighlightText` — so the panel's
greys are the Mac's greys and not a second set beside them.

*The accent, and the one honest caveat in this section.* `--accent` is declared
twice on purpose:

```css
--accent: light-dark(#0a5fd3, #6aa9ff);  /* a sane default */
--accent: AccentColor;                   /* the accent Alice chose, where it is known */
```

An unknown value makes a declaration invalid at computed-value time and the
earlier one stands, so this is a fallback chain with no `@supports` and no
script. Form controls additionally take `accent-color: auto`, which is the
widely available route to the accent on a checkbox and a range. What this
cannot promise is that every recent browser knows the `AccentColor` keyword: in
one that does not, the panel keeps the default pair above and Alice's chosen
accent does not reach the page. That is a degradation, not a failure, and the
hand check named in criterion 1 is what discovers which side of it the
operator's browser is on. Stated here because the intent's press release
promises the accent outright.

*Spacing scale*, on a 4 px base: `--space-1: 4px`, `--space-2: 8px`,
`--space-3: 12px`, `--space-4: 16px`, `--space-5: 24px`, `--space-6: 32px`,
`--space-7: 48px`, `--space-8: 64px`. Nothing in the sheet carries a bare
pixel margin or padding.

*Type scale.* Families `--font-ui` (`ui-sans-serif, -apple-system,
"Helvetica Neue", sans-serif`, the stack already in place) and `--font-mono`
(`ui-monospace, "SF Mono", Menlo, monospace`). Sizes `--text-xs` (11px),
`--text-sm` (13px), `--text-base` (15px), `--text-md` (17px), `--text-lg`
(19px), `--text-xl` (26px), as `rem` values so the Mac's own text size setting
scales them. Line heights `--leading-tight` (1.2) and `--leading-base` (1.55).
Weights `--weight-regular` (400), `--weight-medium` (600), `--weight-bold`
(700). And `--measure: 68ch`, the reading width the press release asks for:
prose is held to it, and tables and card grids are not, which is the whole
difference between a page that opens out on a large display and one that is a
narrow column with empty glass either side.

*Radii.* `--radius-0: 0`, `--radius-1: 4px`, `--radius-2: 8px`,
`--radius-pill: 999px`. The product's look is hard geometry — the sheet's first
line has said so since it was written — so `--radius-0` is the default for a
card, a fieldset and a button, `--radius-1` is for a field, and
`--radius-pill` is for the status dot and nothing else. The scale exists so
that the one place a corner is rounded is a decision with a name.

*Motion.* `--motion-fast: 120ms`, `--motion-base: 240ms`, `--motion-slow:
400ms`, `--ease-out: cubic-bezier(0.2, 0, 0, 1)`, `--ease-in-out:
cubic-bezier(0.4, 0, 0.2, 1)`. Declared in `tokens`; **used only inside
`@media (prefers-reduced-motion: no-preference)`**, which is the rule stated
under criterion 3.

*Targets.* `--target-min: 44px`, the HIG figure, against WCAG 2.2 SC 2.5.8's
24 px AA floor. Every interactive element takes `min-block-size:
var(--target-min)` and `min-inline-size: var(--target-min)`.

**The triad, and the three status meanings.** Three tokens carry the mark's
colours and nothing else does:

| Token | Value | Meaning | The panel's own words for it |
| --- | --- | --- | --- |
| `--status-ready` | `#2B5DAA` | settled and available | the `ready` and `loaded` pills, the `up` status dot, the `ready` setup stage |
| `--status-working` | `#F5C518` | in flight | the `loading` and `downloading` states, a measurement running, every setup stage that is not `ready` or `failed` |
| `--status-failed` | `#E63329` | refused or broken | the `failed` pill, the `failed` setup stage, the `down` status dot, a save the server refused |

The meanings are bound to the state words the panel already renders, so this is
a mapping and not a new vocabulary. Three further tokens —
`--status-ready-ink`, `--status-working-ink`, `--status-failed-ink` — carry the
text colour that meets contrast on each ground, which matters for
`--status-working`: `#F5C518` takes dark text, the other two take light. The
green the sheet uses for `--ok` today disappears, because "ready" is one of the
three meanings and the triad is what states them. **Colour never carries a
meaning alone**: every status is also a word, which is what the pills and the
status line already do, so the mapping adds no information a person has to see
colour to get.

**Increased contrast.** `@media (prefers-contrast: more)`, which is what macOS
"Increase contrast" reaches, redefines `--line`, `--line-strong` and
`--ink-muted` to full-contrast values, raises the focus ring from 3 px to 4 px,
and gives every pill and every `.fact` a solid 2 px border so a block that was
distinguished by a tint is distinguished by an edge. `forced-colors` is not
addressed: it is the Windows high-contrast mode and cond-2609201321050218 says
this panel serves one platform.

**Container queries and `auto-fit` grids.** Every `.panel` declares
`container-type: inline-size` and `container-name: pane`, so each pane lays
itself out to the width **its own box** has been given rather than to the
viewport's — which is the difference between a panel that works in a split
view and one that works at 980 px. Four places take an `auto-fit` grid:

- **the model card** (`.card`) — its meta column and its action row sit in
  `grid-template-columns: repeat(auto-fit, minmax(18rem, 1fr))`, so a narrow
  pane stacks the actions under the name and a wide one puts them beside it;
- **the stat card list** (`.statcard`) — `repeat(auto-fit, minmax(16rem, 1fr))`,
  which is the statistics tab laid out as a grid on a large display and as a
  ladder in a narrow window;
- **the override rows** (`.override`) — `repeat(auto-fit, minmax(14rem, 1fr))`,
  so the model name, its values and its controls wrap instead of squeezing;
- **the wide tables** — each stays inside its `.scroll` region, and at
  `@container pane (max-width: 480px)` the repeated columns are hidden and each
  row is laid out as a two-column name/value grid, which is the press
  release's "the tables drop the columns that only repeat what the card above
  them already says".

The `@container` conditions are literals, because a container condition cannot
read a custom property. Three widths are named in a comment at the top of the
`components` layer and used consistently: narrow below 480 px, middle 480 px to
720 px, wide from 720 px. 360 px is the floor that must work
(cond-2609201321056390); `body`'s `max-width: 980px` becomes
`max-inline-size: min(100%, 1400px)` with the prose held to `--measure`
instead, so the wide end opens out.

**Focus.** One rule, `:focus-visible { outline: 3px solid var(--focus);
outline-offset: 2px; }`, and no rule in the sheet removes an outline without
drawing a replacement indicator in the same rule. The existing `form
input:focus, form select:focus { outline: none; border-color: … }` is deleted
by spc-2609201007368529; this spec's contribution is the `--focus` token and
the increased-contrast widening.

**Viewport, `dvh` and the safe area.** `index.html`'s viewport meta becomes
`width=device-width, initial-scale=1, viewport-fit=cover`, and `body` takes
`min-block-size: 100dvh` with
`padding-inline: max(var(--space-5), env(safe-area-inset-left))` and the
matching right and bottom insets. `dvh` rather than `vh` because a window
whose toolbar comes and goes should not leave the page a strip taller than the
glass. The insets do nothing on a desktop Mac and cost nothing there; they are
declared because `viewport-fit=cover` without them is the defect, and because
a Mac with a notch is a Mac.

**Motion.** The spinner's `@keyframes spin`, the progress bar's
`transition: width`, and any transition on a tab's selected state all move
inside `@media (prefers-reduced-motion: no-preference)`. Outside that block the
spinner is a static ring and the bar jumps; nothing is hidden and nothing is
lost.

### (B) The markup, beyond the panel intent

spc-2609201007368529 delivers the tab strip's `role="tablist"`, each tab's
`role="tab"`, `aria-selected` and `aria-controls`, each pane's
`role="tabpanel"` and `aria-labelledby` (set in `initTabs()`, for the reason
argued in that spec), the roving `tabindex`, and the single
`<p id="live" class="sr-only" role="status" aria-live="polite"
aria-atomic="true">`. None of that is respecified here. Two things are added.

**The tabs contract's keyboard half.** A pure function
`onTabKey(key, index, count)` returns the index to select, or `null` for a key
it does not handle:

| Key | Result |
| --- | --- |
| `ArrowLeft` | the previous tab, wrapping from the first to the last |
| `ArrowRight` | the next tab, wrapping from the last to the first |
| `Home` | the first tab |
| `End` | the last tab |
| anything else | `null` — the event is not consumed |

Activation is **automatic**: the tab the arrow key lands on is selected and its
pane shown in the same turn, with no `Enter` or `Space` required. APG permits
automatic activation when the panels render without noticeable latency, which
is true of seven sections that are already in the document and merely hidden.
The strip is one tab stop; `Tab` from the strip goes to the selected pane's
first control, never to the second tab. `Escape` does nothing: there is nothing
to escape from. The listener calls `preventDefault()` only for the four keys
above, so a browser shortcut on any other key is untouched.

**What may write to the announcement region, and what may not.** The one
`role="status"` element is written with `textContent`, never markup, and only
by `announce(sentence)`. The events that post to it:

1. a download finishing;
2. a download failing;
3. a model finishing loading, or being unloaded or evicted;
4. a context measurement completing;
5. the setup stage changing, and setup failing;
6. the connection to the server dropping, and returning;
7. a client pairing, and a client being revoked;
8. the outcome of a settings save — the success sentence and every refusal the
   server returns;
9. the sentence a destructive two-step confirmation asks, and the outcome when
   it is taken.

**The rule, stated so it can fail: the two-second statistics refresh never
writes to it.** `refreshStats` and everything it calls are forbidden from
reaching `announce`, because a region that announced a poll would re-read the
statistics tab to a screen reader every two seconds — which is the defect and
not the fix. Live regions are kept to two or fewer by the W3C technique; this
panel has one. The mechanical form of the rule is in section (C): `announce`
lives in `events.js`, and no statistics module imports it.

### (C) The script

**The module layout.** `internal/ui/static/app.js` becomes eleven files in the
same directory, all loaded as native ES modules with relative specifiers. No
import map, no bare specifier, no package manifest, no bundler.

| File | What it holds | Touches the DOM |
| --- | --- | --- |
| `app.js` | the entry module: `<script type="module" src="app.js">`. Imports the rest, runs `initTabs()`, opens the `EventSource`, installs the delegated listeners, and nothing else. | yes |
| `dom.js` | `$`, `announce`'s target lookup, `sr`-region access, `setLabel` (the `setAttribute` wrapper the accessible names go through). **`$` is exported from here alone** so the guard's `$('someId')` regex keeps meaning one thing. | yes |
| `format.js` | **the DOM-free module.** Every pure function the eight harnesses lift today, and the two constants they lift: `SAMPLING_FIELDS`, `PRIVATE_BIND`, `modelInfoLine`, `measurementText`, `contextLabel`, `pinLabel`, `graceWaitHint`, `numberOrNull`, `bindSelectValue`, `bindSelectBody`, `bindNoticeText`, `privateBindLabel`, `privateBindDisabled`, `extraBindOption`, `chatRule`, `clientPaired`, `clientSeen`, `figure`, `millis`, `msFigure`, `rateFigure`, `tokensLabel`, `bucketLabels`, `sharePercent`, `stopReason`, `busyLine`, `sparkline`, `onTabKey`, `stateChangeSentences`. Imports nothing. | **no** |
| `state.js` | the snapshot, `render()`'s dispatch, the `EventSource` subscription and its reconnection, the `settingsTouched` lock and the marked exceptions to it. | yes |
| `events.js` | the delegated listener table, the tab key listener, `announce()`, `fieldError()`, `alertErr()`, `confirmBtn()`. | yes |
| `tab-models.js`, `tab-search.js`, `tab-stats.js`, `tab-connect.js`, `tab-clients.js`, `tab-posture.js`, `tab-settings.js` | one module per tab, each exporting its renderers. `tab-settings.js` holds `const body = {` and the settings submit path; `tab-stats.js` holds `refreshStats` and the two-second interval. | yes |

Go's builtin MIME table maps `.js` to `text/javascript; charset=utf-8`, so the
embedded file system serves modules with no handler change.

**The three inline handlers become delegated listeners.** Module scope is not
global, so `onclick="showTab('stats');return false"` would stop resolving even
if it were kept. spc-2609201007368529 already turns the three anchors —
`recordingBadge`, the Find Models link in `modelsEmpty`, the Settings link in
`statsOff` — into `<button type="button">`. This spec adds the mechanism: one
listener on `document`, matching `[data-goto]` on the event's `closest()`, so
the three buttons carry `data-goto="stats"`, `data-goto="search"` and
`data-goto="settings"` and the script carries no per-button binding. No element
in `index.html` carries an `on*` attribute afterwards, which is what criterion 5
asserts and what makes the CSP follow-up in **Security** possible at all.

**The surface guard, taught to scan the directory — the new coupling rules,
precisely.** `internal/archtest/settings_surface_test.go` is a pair of string
searches over source, and the searches pin the authoring format in six ways
that are invisible from the markup. Each is restated below as it stands and as
it becomes.

| # | The coupling today | The rule after the split |
| --- | --- | --- |
| 1 | The script side is the one file `internal/ui/static/app.js`, named as a path literal in `settingsPaneMarkupAndScript` and again in `postedSettingsKeys`. The markup side is `index.html`. | The script side is **every `*.js` file directly under `internal/ui/static/`, read in sorted-path order and joined with a newline**, comments stripped as now. One helper, `panelScripts(t)`, is the only reader, and both halves call it. The markup side stays `index.html` alone. The two halves stay checked separately, for the reason that file already records: neither surface may satisfy a rule about the other. |
| 2 | `jsBlock` finds each of `const body = {`, `function bindSelectBody(` and `function chatRule(` with `strings.Index` — the **first** occurrence, silently. | Each of the three markers must occur **exactly once** across the concatenation. The guard counts them and fails on zero (it is reading the wrong files) and on two or more (a second `const body = {` in another module would otherwise shadow the real one without a word). |
| 3 | The element lookup is the literal `$('someId')` form (`panelControlIDRE`); an id the script computes is invisible to it. | Unchanged, and now load-bearing across files: `$` is exported from `dom.js` alone and imported by name, never re-declared, aliased or wrapped. A module that wrote `const el = document.getElementById(id)` would take its controls out of the guard's sight. |
| 4 | `controlIDsBoundTo` is **line-scoped**: a setting's key and the `$('id')` that carries it must share one source line. | Unchanged, and it survives the move only if the move does not reformat: a binding may not be split across lines, and `tab-settings.js` keeps every key-and-lookup pair on one line. |
| 5 | `objectKeyRE` matches a lower-snake identifier followed by a colon, and `settings_test.go` asserts the literal `advertise:          $('setAdvertise').checked,` — its column alignment included. | Unchanged. The submit body keeps its key order and its alignment; it is re-indented by nobody. |
| 6 | `settingsPaneMarkup` slices the markup from `<section id="tab-settings"` to the **first** `</section>`, so the pane may hold no nested `<section>`; and `posture_test.go` asserts `<section id="tab-posture" class="panel">` byte for byte. | Unchanged. Task groups stay `<fieldset class="group">`; the Posture pane's opening tag is preserved, which is why every pane takes its `role="tabpanel"` from `initTabs()` rather than from markup. |

Two rules are added that have no equivalent today, because a directory can fail
in ways a named file cannot:

7. **The scan fails if the directory holds fewer than two `.js` files, or if it
   holds `app.js` alone.** A bundler, a rename or a deletion that emptied the
   directory would otherwise leave every search passing over nothing. This is
   the guard's own liveness rule, of the kind
   `TestEveryExemptedSettingStillExists` already is.
8. **Every `.js` file in the directory is served.** The scan reads the source
   the binary embeds, and the `go:embed` pattern must cover the whole
   directory, so no file can be scanned that a browser never loads, and none
   loaded that the scan never reads. A new assertion checks the embed pattern
   against the directory listing.

**The eight node-lifted harnesses, converted.** Each of the eight helpers that
runs node today lifts function text out of `app.js` with `extractFunction` and
evaluates it. All eight instead `import()` `format.js`:

| Harness | File | What it lifts today | After |
| --- | --- | --- | --- |
| `evalBindSelect` | `bindhost_test.go` | `extraBindOption`, `renderBindOptions`, behind a `FakeSelect`/`FakeOption` stub | imports `extraBindOption` from `format.js`; `renderBindOptions` stays lifted, because it touches a select |
| `evalBindMode` | `bindmode_test.go` | the `PRIVATE_BIND` const plus `bindSelectValue`, `bindSelectBody`, `privateBindLabel`, `privateBindDisabled`, `bindNoticeText` | all six imported; `constDecl` is deleted |
| `renderClientsPane` | `clients_test.go` | `clientPaired`, `clientSeen`, `renderClients`, behind a node-tree DOM stub with `confirmBtn`, `alertErr` and `api` stubbed | imports the two pure ones; `renderClients` stays lifted behind the same stub |
| `evalPanelDOM` | `history_test.go` | the statistics and history row builders behind a `$`-stub document | imports the pure formatters; the row builders that write into `$` stay lifted |
| `evalPanel` | `panel_test.go` | one expression against named functions, `escapeHtml` stubbed | imports from `format.js`; the `escapeHtml` stub stays, for the reason its comment gives |
| `evalPanelExpr` (with `evalPanelArray`, `evalPanelNumber`) | `panel_test.go` | the JSON of one expression | imports |
| `evalPanelValue` | `settings_test.go` | an object-returning function | imports |
| `evalPanelControls` | `settings_test.go` | `SAMPLING_FIELDS`, `numberOrNull`, `writeSampling`, `readSampling` behind a value-coercing cell stub | imports the const and `numberOrNull`; `writeSampling`/`readSampling` stay lifted behind the cell stub, which is the thing that test is about |

The mechanism: `evalJS` gains a sibling, `evalModule(t, snippet)`, which runs
`node --input-type=module -e <snippet>`; the snippet's first line is
`const m = await import(<file URL of static/format.js>);`. Node allows
top-level `await` in module input, and a `file://` specifier needs no import
map and no directory walk. `evalJS` and `extractFunction` both survive for the
DOM-touching renderers that stay lifted, and both keep the skip when node is
absent. The prize is the one the research note named: a test can no longer pass
against a function the browser never loads, because the test loads the same
module the browser does.

**A new guard for the tabs contract.** `internal/archtest/tabs_contract_test.go`,
a string scan of the same kind as the surface guard:
`TestTheTabsContractIsTheAPGOne` asserts that every `.tab` button in
`index.html` carries `role="tab"`, `aria-selected` and an `aria-controls`
naming a `<section>` id that exists; that the strip carries `role="tablist"`
and an `aria-label`; that exactly one element carries `role="status"`; and that
the script's tab key listener names all four of `ArrowLeft`, `ArrowRight`,
`Home` and `End`. `TestOnlyEventsReachTheAnnouncementRegion` asserts that
`announce` is exported from `events.js` alone and that no statistics module
imports it — the mechanical form of the rule in section (B). The node-lifted
half, over `onTabKey` imported from `format.js`, covers the wrap-around and the
two ends.

### (D) The offline property, as a test

The property, stated so it can fail: **the markup and the stylesheet name no
host, and the only outbound addresses the panel carries are documentation links
an operator clicks.** spc-2609201007368529 writes
`TestThePanelFetchesNothingFromOffTheMac` over three files; this spec widens it
to the directory and adds the stylesheet's own vectors:

- no `<link href>`, `<script src>`, `<img src>`, `<iframe>` or `<source>` with
  a scheme, in `index.html`;
- no `@import`, no `@font-face`, and no `url(…)` with a scheme in `style.css`;
- no `fetch`, `EventSource`, `XMLHttpRequest`, `navigator.sendBeacon` or
  dynamic `import()` target in any module that is not a same-origin path
  beginning `/api/` or a relative module specifier;
- no bare module specifier and no import map anywhere;
- every `https://` occurrence that remains is an `<a href>` naming a page that
  exists under `docs/`.

The favicon stays the `data:` SVG URI it is today — a data URI reaches nothing
and is allow-listed by name, not by pattern. The hand check with networking off
is spc-2609201007368529's and is already owed by that intent; this spec does
not duplicate it, and says so under the acceptance map.

### (E) The renderer trigger

**The rule, in full.** Vendor Preact and htm when **either** of these becomes
true, whichever comes first:

1. **The hand-written exceptions to the settings-touched lock reach three.** An
   exception is a call made from `render()` *after* `renderSettings()` has
   returned early on `if (settingsTouched) return;`, because the thing it draws
   is live state rather than a value Alice is typing. There are exactly **two**
   today, both in `render()` and both carrying the comment that explains them:
   `refreshOverrideModels()` (the list of models to choose an override for) and
   `renderBridgeState()` (what the Discord bridge is doing). A third is the
   trigger.
2. **A live per-model row must keep an editable field** — a row inside the
   settings subtree that both updates from the snapshot and holds a value Alice
   can type into. The whole-form lock cannot express that: it is all-or-nothing
   per form, and a per-row diff written by hand is the thing a keyed renderer
   exists to do.

**The counting test.** Each exception carries a marker comment on the line
above its call, `// touched-exception: <why>`, and
`TestTheSettingsLockKeepsAtMostTwoExceptions` counts the markers across the
static script directory and **fails at three or more**. Its failure message
names the trigger, this spec and the vendoring plan, so the person who writes
the third exception is told what the record says to do instead of discovering it
later. The marker is a convention rather than a heuristic on purpose: counting
"calls after an early return" from source is a guess, and a guess that gates a
framework decision is worse than no gate.

**What vendoring would look like when the trigger fires.** Two committed files,
`internal/ui/static/vendor/preact.mjs` and `internal/ui/static/vendor/htm.mjs`,
roughly 4 kB together, imported with relative specifiers. **Still no build
step, still nothing fetched, still no package manifest and no lockfile.** Each
file carries a recorded source hash pinned by an architecture test, the idiom
spc-2609201011302811 uses for the icon rasters, so a vendored file cannot drift
from the release it was taken from without a test saying so. The offline
assertions in section (D) and the directory rules 7 and 8 in section (C) cover
`vendor/` unchanged — it is `.js`-adjacent source in the served directory, and
the embed pattern must include it. Conversion is renderer by renderer behind
`html` tagged templates, not in one movement, which means a second templating
primitive sits beside the `innerHTML` strings until the last renderer is
converted; that cost is accepted in advance as the price of not rewriting 31
assignments at once. **None of this happens in this intent**
(cond-2609201321052790): the trigger is written down and not exercised.

### (F) The menu-bar menu

**The lift that makes criterion 8 possible.** The items are built today as
local variables inside the tray callback, so no Go test can reach the list. The
titles and tooltips move to a package-level table in the command package's
`menubar.go`:

```go
type menuItem struct {
	Title, Tooltip string
	Enabled        bool
	Separator      bool
}
var serverMenuItems = []menuItem{ … }
var clientMenuItems = []menuItem{ … }
```

`runMenuBar` and `runClientMenuBar` build their menus by walking the table, so
the list a test reads is the list the Mac draws. The status line and the
endpoint line keep their titles set at run time from the table's entry as a
starting value.

**The server menu, held to the panel's vocabulary.**

| Item | Title | State |
| --- | --- | --- |
| 1 | the status line — `N models · N loaded` while serving, `Setting up MLX runtime…` while the runtime installs | disabled |
| 2 | the endpoint — the first endpoint's URL | disabled |
| — | separator | |
| 3 | `Open Control Panel` | enabled |
| 4 | `Copy Endpoint URL` | enabled |
| — | separator | |
| 5 | the quit item, naming the product | enabled |

Every noun in it is the panel's: **models** and **loaded** are the words the
model cards use, **control panel** is what the panel calls itself, and
**endpoint** is the Connect tab's word and the word beside its `Copy` buttons.
The quit item names the product as the rename lane leaves it named; this spec
fixes the item's shape and its position, not the product's name.

**The client menu** — the menu a second account's instance shows — keeps its
three items: the status line `Server running in another account` (disabled),
`Open Control Panel`, and the quit item. `Open Control Panel` is worded
identically in both menus, which is the point of holding it to a table.

**The icon.** `systray.SetTemplateIcon` at every call site, with the
solid-cube template glyph spc-2609201011302811 delivers: 44×44, the cube at
`translate(22,23) rotate(-8) scale(0.062)`, its three faces in `#000000` with
the top face restated for the lift. A template image carries one channel, which
is why the menu-bar rendering drops the coloured forms; the glyph is referenced
here, not redrawn. That spec's own template-image test holds the black-plus-alpha
property; this spec adds nothing to it.

**The test.** `TestTheMenuBarItemsAreTheOnesTheRecordNames` in the command
package asserts the table: the item count, the order, which two are disabled,
where the two separators sit, and each title. The vocabulary half is in
`internal/archtest` (section (G)), so one test holds all three surfaces to one
word list rather than three tests each holding one.

### (G) The terminal verbs

**The vocabulary rule.** A verb's human output uses the panel's nouns:
*model*, *models*, *ready*, *loaded*, *failed*, *endpoint*, *control panel*,
*settings*, *records*, *client*, *this build*. Where a verb has a word the
panel has no equivalent for, the word is added to the shared list with the
surface it comes from, the way the exemption table in the surface guard carries
its reasons — a word in one surface and not the others is what the rule exists
to catch.

**Aligned tables.** `internal/lifecycle/config.go` already pads with
`fmt.Fprintf(t.Out, "  %-*s  %s\n", width, key, …)`; `status.go` and
`doctor.go` each pad by hand in their own way, and `doctor.go` carries its own
`pad`. One helper moves into `term.go` — `Table(t Terminal, rows [][]string)`,
which measures every column and prints two spaces between them — and all three
verbs print through it. A verb that prints a table and does not use it is the
thing the goldens catch.

**One-sentence refusals that name the next step.** Every refusal a verb writes
is one sentence and names the next thing to do. The doctor findings already
carry `run: <command>`, which is the shape; the rule generalises it. A refusal
never explains the internals and never asks a question.

**Golden-output tests per verb**, beside the existing JSON goldens under
`internal/lifecycle/testdata/`. The human output is rendered through
`Terminal{Out: buf, Color: false, Redraw: false}` — the plain path a pipe and
`ACCESSIBLE` both select — **from the same fixtures the JSON goldens decode**,
so the pair cannot drift:

| Golden | Verb | From |
| --- | --- | --- |
| `status_serving.txt` | the status verb, serving | `status_serving.json` |
| `status_not_serving.txt` | the status verb, not serving | `status_not_serving.json` |
| `status_foreign.txt` | the status verb, a foreign listener | `status_foreign.json` |
| `doctor_report.txt` | the doctor verb | `doctor_report.json` |
| `config_show.txt` | the config-show verb | `config_show.json` |

The four writing verbs — install, place, uninstall, update — get one golden
each **for their refusal path only**, from the fakes the command package's
tests already carry, because their success path writes to disk and a golden
over it would be a test of the fake. The version verb gets none: it prints one
line, already asserted. A `-update` flag on the goldens is not added; a golden
this small is written by hand, and a flag that rewrites an expectation from the
code is how a golden stops being one.

**The shared vocabulary assertion.** `internal/archtest/surface_vocabulary_test.go`
holds one word list and reads all three surfaces against it: the panel's
`index.html`, the menu-bar table in the command package's `menubar.go`, and the
`.txt` goldens above. A noun that appears in one surface's user-facing strings
and in neither of the others fails, with the word and the surface named. This
is the cross-surface promise of AGENTS.md armed the way the settings-sync
promise is: a test, not a habit.

### (H) Documentation

`docs/control-panel.md` is **delivered by itd-2609100519003748** as a
reference describing the seven tabs and the Settings pane's task groups. This
intent **extends** it with four things and no recipes: the adaptive layout
(what a pane does at a narrow, a middle and a wide width), the system
appearance (light, dark, increased contrast, the accent, and the three status
colours with their three meanings), the keyboard contract (the tab strip's
arrow keys, Home and End, automatic activation, the focus ring, where `Tab`
goes), and the one announcement region with what does and does not reach it.

The pages that describe the menu bar, edited so their words are the panel's:
`docs/getting-started.md`, `docs/lifecycle.md`, `docs/bind-address.md`,
`docs/mesh-vpn.md`, `docs/request-statistics.md`, `README.md`.

The pages that describe the terminal verbs, edited the same way:
`docs/lifecycle.md`, `docs/lifecycle-reference.md`, `docs/getting-started.md`,
`docs/discord-bridge.md`, `README.md`.

Checked and expected unaffected, listed so the sweep is complete rather than
selective: `docs/posture-reference.md`, `docs/models-list.md`,
`docs/statistics-explained.md`, `docs/statistics-store-reference.md`,
`docs/pairing.md`, `docs/pairing-explained.md`, `docs/logging.md`. What was
found on each page goes in the shipping record.

## The acceptance-criteria map

The store mints no ids for an intent's acceptance criteria — only for its scope
conditions — so they are numbered here in the order the intent lists them.
Every test named below is new unless it says otherwise, and **is watched red
against the panel as it stands before the change**, with the failure it showed
recorded in the shipping record. Two checks are hand checks and are named as
such.

| # | Criterion | Held by |
| --- | --- | --- |
| 1 | The stylesheet is layered, declares `color-scheme: light dark`, carries one `light-dark()` token block drawn from the system palette and accent, no second dark block, and a rule set for increased contrast | `TestTheStylesheetIsLayeredAndHasOnePalette` over `style.css`: the `@layer tokens, base, components, utilities` statement is present and is the file's first at-rule; `color-scheme: light dark` is declared on `:root`; every colour token is declared once, in `@layer tokens`; no `prefers-color-scheme` block remains; a `prefers-contrast: more` block exists and redefines `--line`, `--line-strong`, `--ink-muted` and the focus width. Plus **hand check: light, dark and increased contrast on a real Mac with a non-default accent colour** — the check that also discovers whether the operator's browser knows `AccentColor`. |
| 2 | Every tab reflows at 360 px, a middle width and a wide width with no horizontal scrolling and nothing clipped | **A deviation from the criterion's letter, argued below:** `TestEveryPaneReflowsToTheWidthItIsGiven` in `internal/archtest`, a Go assertion that parses `style.css`, computes each `repeat(auto-fit, minmax(M, 1fr))` grid's column count arithmetically at 360 px, 600 px and 1440 px, and asserts one column at the narrow end for every grid whose `M` is 180 px or more, two or more at the wide end for the model card, stat card and override grids, `container-type: inline-size` on every `.panel`, no fixed `px` width above 360 px on a pane's descendant, and `overflow-x` kept on every `.scroll` region. Plus **hand check: the panel in a macOS split view**. |
| 3 | 44 px targets, a `:focus-visible` ring never removed unreplaced, `env(safe-area-inset-*)` with `viewport-fit=cover`, every transition and animation inside `prefers-reduced-motion: no-preference` | `TestEveryTargetAndRingAndMotionRuleIsInPlace` over `index.html` and `style.css`, enumerating the controls the way `TestEveryNumericSettingsControlIsProbed` enumerates: `min-block-size`/`min-inline-size` of at least 44 px on every interactive selector; a `:focus-visible` rule that draws an outline; no `outline: none`/`outline: 0` without a replacement indicator in the same rule; `viewport-fit=cover` in the viewport meta and at least one `env(safe-area-inset-` use; and **every** `transition:`, `animation:` and `@keyframes` reference inside a `prefers-reduced-motion: no-preference` block. |
| 4 | The tabs contract's roles and keys, automatic activation, focus following; and one `role="status"` region the statistics refresh never writes to | `TestTheTabsContractIsTheAPGOne` and `TestOnlyEventsReachTheAnnouncementRegion` in the new `internal/archtest/tabs_contract_test.go` (section (C)), plus the node-lifted tests over `onTabKey` imported from `format.js` (both ends, both wraps, `Home`, `End`, and `null` for an unhandled key) and over `stateChangeSentences` for each of the nine events in section (B). The roles themselves are spc-2609201007368529's and are re-asserted here rather than re-specified. |
| 5 | Native ES modules by tab, `type="module"` and relative specifiers only, no inline handler, no import map, no bare specifier, nothing from any origin | The surface guard taught to scan the directory, with the eight coupling rules in section (C) — the six restated and the two added; `TestNoMarkupCarriesAnInlineHandler` (no `on*` attribute, no `<a href="#">`, no positive `tabindex`); `TestEveryScriptIsAModuleWithARelativeSpecifier` (every `<script>` carries `type="module"`, every specifier is relative, no `<script type="importmap">`); and the eight harnesses in `internal/ui` importing `format.js` rather than lifting its text. |
| 6 | The panel renders completely and every control works with networking off | The widened `TestThePanelFetchesNothingFromOffTheMac` (section (D)). The offline **hand check is spc-2609201007368529's and already owed by that intent**; this spec does not re-owe it, and the shipping record cites the panel intent's record for it. |
| 7 | The triad occurs only in the mark and the three status meanings; every other colour resolves to a system colour, the accent, or a token derived from one | `TestTheTriadIsTheMarkAndTheThreeStatuses` over `index.html` and `style.css`: every occurrence of `#F5C518`, `#E63329` and `#2B5DAA` is either inside the header mark's `<svg>`, inside the allow-listed favicon data URI, or one of the three `--status-*` token declarations; and every other colour value in the stylesheet is a system colour keyword, `AccentColor`/`AccentColorText`, a `light-dark()` of those, or `var(--…)`. |
| 8 | The menu-bar items are the status line, the endpoint, open panel, copy URL and quit, in the panel's vocabulary, under the solid-cube template icon | `TestTheMenuBarItemsAreTheOnesTheRecordNames` in the command package over the lifted table (count, order, the two disabled entries, the two separators, each title), plus the shared vocabulary assertion below. The template-icon half is spc-2609201011302811's existing test and is cited, not duplicated. |
| 9 | Every verb's human output uses the panel's nouns, prints aligned tables, and states a refusal in one sentence naming the next step | The five plain-text goldens plus the four refusal goldens in section (G); `TestEveryTablePrintsThroughTheSharedHelper` (no verb formats a padded column of its own); `TestEveryRefusalIsOneSentenceAndNamesANextStep` over the refusal goldens; and `internal/archtest/surface_vocabulary_test.go`, which holds the panel, the menu-bar table and the goldens to one word list. |
| 10 | The spec records the renderer trigger in full | Section (E), plus `TestTheSettingsLockKeepsAtMostTwoExceptions`, which counts `// touched-exception:` markers across the static script directory and fails at three or more. |
| 11 | `docs/control-panel.md` describes the adaptive layout, the system appearance and the keyboard contract, and every page describing the menu bar or the verbs uses the panel's words | The edited pages in section (H); `abcd docs lint` for links, stray markdown and change-narration; and the sweep list with what was found on each page, in the shipping record. `docs/control-panel.md`'s existence is itd-2609100519003748's; its four new sections are this intent's. |

**Why criterion 2's holder is a Go assertion and not the node-lifted layout
test the criterion names.** The criterion asks for a test that "evaluates each
pane's container queries and track lists" and "asserts the resolved column
count". Resolving a container query is layout, and layout needs a layout
engine. Node has none, and a headless browser is a dependency and a toolchain
that cond-2609201321055222 rules out — the same ruling that put a
browser-driven harness out of scope in spc-2609201007368529. What *is*
computable without an engine is the arithmetic: given a declared
`minmax(M, 1fr)` track list, a gap and a container width `W`, the column count
is `max(1, floor((W + gap) / (M + gap)))`, and that is a property of the
declarations the stylesheet holds. Putting it in Go rather than node is
strictly better here for a second reason: CI installs no node, so a node test
is green-by-skip there, and this is a criterion that should fail in CI. The
criterion's substance — a failing case an assertion can find, at three named
widths — is kept; its letter is not buildable, and the hand check in a real
split view is the other half. The pattern is spc-2609201007368529's criterion 6
deviation, argued the same way.

## Security

`internal/ui` is not in AGENTS.md's list of trust boundaries, but the panel it
serves sits on one, and this change is reviewed by the security-reviewer agent
before it lands. The control plane is loopback-only and exempt from the bearer
check, so every account on this Mac reaches it (adr-2609091123526871 §7,
`docs/bind-address.md`), and the page renders strings chosen by whoever paired
a client — which, under first-come pairing, is anything on the network. A name
that reached markup would run same-origin against the whole settings API.

- **No new route, method or parameter.** The control plane is untouched. This
  is a change to markup, a stylesheet, the script's file layout and two Go
  surfaces that print text.
- **The announcement region is the injection surface this change adds.** Its
  sentences name models and clients, so they are attacker-influenced text. It
  is written with `textContent` only, never `innerHTML`, and
  `stateChangeSentences` lives in `format.js`, which touches no DOM at all and
  therefore cannot be the thing that writes markup. Accessible names on
  generated rows and card buttons go through `setLabel`, a `setAttribute`
  wrapper, and are never interpolated into a template literal — the rule
  spc-2609201007368529 states and this module layout makes mechanical.
- **No new storage and no new host.** No `localStorage`, `sessionStorage`,
  cookie or IndexedDB; no fetched subresource. Section (D) is what arms the
  last of those, and it now covers the stylesheet's `@import`, `@font-face`
  and `url()` vectors, which the narrower version did not.
- **The module split removes an execution surface rather than adding one.**
  Module scope is not global, so the panel's functions stop being reachable
  from a `javascript:` URL or a console one-liner typed by someone who was
  handed one, and the three inline handlers — the one thing in the page that
  executes an attribute's contents — are gone.
- **A stricter Content-Security-Policy becomes possible once the inline
  handlers are gone. It is a follow-up, not a criterion of this intent.** With
  no `on*` attribute and every script an external module, the panel could send
  `default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self'
  data:; connect-src 'self'; base-uri 'none'; form-action 'none'`. Two things
  must be established first, and neither is this intent's work: an inventory of
  the inline `style=` attributes the renderers set, each of which
  `style-src 'self'` would break, and a decision about the favicon's `data:`
  URI, which needs the `data:` source in `img-src`. And the honest limit —
  **CSP would not stop the injection risk above.** The 31 `innerHTML`
  assignments are same-origin script writing markup, which every one of those
  directives permits; `textContent` and `setAttribute` are what stop it. The
  header is defence in depth against a class of bug this panel has not got,
  which is why it is filed as a follow-up and not claimed as a mitigation.

## Verification gates

`make test` green; `gofmt -l .` empty; `go vet ./...` clean; `abcd docs lint`
clean; `abcd intent ready itd-2609201315575657` reporting ready. The
node-gated tests skip where node is absent and CI installs none, so the run
that counts for them is on a Mac with node installed, and the shipping record
says which machine ran them and what node reported. The two hand checks — the
panel in a macOS split view, and the panel under light, dark and increased
contrast with a non-default accent colour — are recorded as hand checks in the
shipping decision line; the offline hand check is cited to
spc-2609201007368529 rather than re-owed. The security review above happens
before the change is presented. Every new test is watched to fail against the
panel as it stands, and the shipping record names the failure each one showed.

## Out of scope

The information architecture — the re-homing of controls, the inventory, the
removal of `alert()`, `confirm()` and the timed arming, the per-field error
slots and the role attributes in the markup: itd-2609100519003748 /
spc-2609201007368529 (cond-2609201321050310). The as-you-type refusal and its
validation route: itd-2609200823520756 / spc-2609201007366798. Dessau Chat,
which never sees the server's surfaces, and any styling crossing between the
two beyond the shared mark (cond-2609201321058953). The mark's own drawings
and rasters: spc-2609201011302811. Reach beyond loopback — a phone, a tablet or
a laptop on the mesh opening the panel is out by adr-2609091123526871, not by
omission (cond-2609201321051646). Any framework, component library, bundler,
minifier, CSS toolchain, import map or package manifest
(cond-2609201321055222); Preact and htm are pre-committed for the trigger in
section (E) and are not vendored here (cond-2609201321052790). Any new setting
or any change to `config.json`'s shape, and any third entry in the surface
guard's exemption table. Translation of the panel. A browser-driven test
harness, and with it the letter of criterion 2's holder. `forced-colors`
support, which is another platform's mode (cond-2609201321050218). The
Content-Security-Policy header, filed as the follow-up **Security** names.
Pre-1.0, so no migration code and no compatibility shim: the markup, the
stylesheet, the module layout and the panel's element ids may all change
outright (cond-2609201321054714).

## Sequencing

This intent lands **after itd-2609100519003748**, as its own lane
(cond-2609201321050310): that intent delivers the arrangement, the role
attributes and `docs/control-panel.md`, and this one takes all three as given.
It also depends on **itd-2609200827202340** for the solid-cube template glyph
criterion 8 names, so section (F)'s icon half is only assertable once the mark
lane has landed; the item-list half does not wait for it. The ordering matters
in one concrete way beyond politeness: the panel intent freezes the eight
node-lifted harnesses and the surface guard, and this intent rewrites them, so
running the two lanes concurrently would put each intent's criteria in the
other's way.

## Falsifiers, from the Mechanism claim

A live-update case that cannot keep focus, selection or scroll position without
a keyed renderer — the written trigger in section (E), which fires rather than
being patched around. A layout requirement from criterion 2 that cannot be
expressed as declared tracks and container conditions and therefore cannot be
asserted without a layout engine. An appearance requirement that needs a
fetched subresource — a font, an icon set, a stylesheet — to render. A
`prefers-contrast` or accent requirement the operator's own recent browser does
not honour, found by the hand check, which would mean the system-first palette
buys less than the intent claims. A module boundary that cannot be drawn
without taking a setting's binding out of the surface guard's sight, which
would mean the guard and the modularity are in genuine conflict rather than
merely coupled. Any of these reopens the intent rather than being worked
around.
