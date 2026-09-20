---
id: itd-2609201315575657
slug: the-dessau-server-s-face-fits-any-window-and-feels-native-on
spec_id: spc-2609201321051342
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609100519003748, itd-2609200827202340]
severity: minor
impact: additive
origin: researcher-authored
production_mode: dictated-and-formatted
---

# The Dessau Server's face fits any window and feels native on the Mac: Alice opens the control panel in a narrow window, a split view or on a large display and every tab lays itself out to the width it is given; the panel follows the Mac's light, dark and increased-contrast settings and its accent colour, with the three-form mark as the only brand colour; the tabs, controls and live state are reachable by keyboard and announced to a screen reader; the menu-bar menu and the terminal verbs speak the same words as the panel; and all of it is plain files served from this Mac alone, loading nothing from anywhere.

## Press Release

Alice runs the Dessau Server on the Mac under her desk. Most of the day the
control panel sits in a window pushed to one side of the screen — a third of it,
with her editor in the rest — and what she sees there is a page built for that
width, not a wide layout with its right-hand side cut off. The model cards stack
into a single column, the tables drop the columns that only repeat what the card
above them already says, the tab strip stays where her thumb expects it, and
nothing scrolls sideways. In the evening she drags the window onto the large
display in the corner and the same page opens out: cards across a row, the
statistics laid out as a grid rather than a ladder, the reading width of the
prose held so the sentences do not run the whole width of the glass. She never
chooses a layout. Each tab lays itself out to the width it has been given.

The panel looks like the Mac it is running on. It follows the light and dark
setting, it follows increased contrast, and the colour it highlights a selected
tab or a focused field with is the accent colour Alice chose in System Settings,
not one the product picked for her. The only brand colour on the page is the
mark in the corner — the yellow triangle, the red square, the blue circle — and
the same three colours where the panel states a status: ready, working, failed.
Everything else is the system's own palette, which is why the page does not look
like a web page that happens to be open on a Mac.

She works it from the keyboard when her hands are already there. Tab reaches
every control; the arrow keys walk the tab strip and the pane changes as the
focus lands; the focus ring is visible on every stop. When a download finishes
while she is typing in another pane, the panel says so once, in one place, and
her caret does not move. The two-second refresh behind the statistics tab says
nothing at all, because a panel that announced its own polling would be the
defect rather than the fix.

The words do not change when she leaves the page. The menu-bar dropdown says
status, endpoint, open panel, copy URL and quit, in the panel's nouns, under the
same small cube. When she asks the same question in the terminal, the verb
answers with the panel's nouns in aligned columns, and if something is wrong it
says so in one sentence that names the next thing to do. Three surfaces, one
vocabulary: what she learns in any of them is true in the other two.

Bob, at the next desk, never opens the panel. He points his editor at the
endpoint and gets answers. Carol, on the mesh in the other room, does the same
from her laptop. Neither of them can reach the panel at all, and neither of them
misses it: the panel is the operator's surface and the API is everybody's. The
page Alice works is plain files served out of the binary — markup, a stylesheet
and a handful of modules — so it renders, and every control on it works, on a
Mac that has been taken off the network entirely.

## Why This Matters

The panel is the product's face. A tester who has never read a line of this
repository forms their judgement of the server in the first ten seconds of the
page, and today the page argues against itself: one stylesheet with two
hand-written dark-mode blocks and no responsive rule at all, a fixed reading
column that is the same width in a split view as on a wide display, a tab strip
of plain buttons, and a brand palette that owes nothing to the Mac underneath
it. The functionality behind it is sound, which is exactly the problem — the
surface makes a finished server look unfinished. The test of whether this has
worked is behavioural and was named when the work was commissioned: if testers
still reach for the terminal or hand-edit the settings file after this ships,
it has not worked.

The stack question was put to the evidence rather than to taste, and the plain
files won it. Every defect on the list is a stylesheet or markup defect — a
container query, a token block, a focus ring, a tab list, a live region — so a
framework would fix none of them and would hand the panel a dependency tree in
a product that has none. The modern CSS this needs is Baseline in the browser
the operator is actually using, because that browser is their own recent Safari
or Chrome on loopback and not the open web; native modules give the modularity
the work needs without putting a compiler in front of the artefact the binary
embeds — which matters here, because the settings guard proves its promise by
scanning the served source, and a bundler would leave the scan passing over
files nobody ships. What is honestly left open is the live view: the panel's
whole-form lock during editing is a hand-rolled diff whose exception list grows
with every live field. So the reversal is pre-committed rather than argued
about later — at three exceptions, or the first live per-model row that needs
an editable field, a four-kilobyte keyed renderer is vendored in, still with no
build step and nothing fetched.

The third argument is the one that outlives this redesign. This product speaks
from three places, and a person who learns a word in one of them should meet the
same word in the others; a status that is called one thing on the page, another
in the menu and a third in a terminal table teaches the operator that no surface
is authoritative. That sync obligation is not new here, and neither is the way
it is armed: a cross-surface promise in this repository is held by a test, not
by a habit. This intent puts the shared vocabulary, the adaptive layout, the
system appearance and the keyboard contract on that same footing — each one
with a named holder, so that the next change to any of the three surfaces has
something to fail against.

## Mechanism

We expect a native-feeling, adaptive panel from plain files because every defect
found is a stylesheet or markup defect, modern CSS (container queries, light-dark
tokens, layers) is Baseline in the operator's own browser, and native modules
give modularity without a compiler; the falsifier is a live-update case that
cannot keep focus or selection without a keyed renderer, which is the written
trigger for vendoring Preact+htm.

## Scope Conditions

- **Mac only.** The panel is the face of a server that runs on Apple Silicon <!-- cond: cond-2609201321050218 -->
  Macs, and "feels native" means native to macOS: the system's colours, its
  accent, its light, dark and increased-contrast settings, its text stack and
  its 44 px target figure. No other platform is served, and no cross-platform
  appearance is attempted.
- **Loopback only.** The reach recorded in adr-2609091123526871 stands <!-- cond: cond-2609201321051646 -->
  unchanged: the panel answers on loopback, to every account on this Mac, and to
  nothing on the network. This intent adds no route and no reach, and a phone or
  tablet opening the panel over the network is out of scope by that decision
  rather than by omission.
- **The browser is the operator's own recent Safari or Chrome.** This is one <!-- cond: cond-2609201321054163 -->
  page, served over loopback, to the person running the server — not a public
  site — so Baseline newly available is the floor: container queries, cascade
  layers, nesting and `light-dark()` are used directly, with no polyfill, no
  transpilation and no support matrix.
- **Plain files, native modules, no build step, no dependency, nothing <!-- cond: cond-2609201321055222 -->
  fetched.** The panel stays markup, a stylesheet and native ES modules under
  the embedded static directory, loaded with relative specifiers. No bundler, no
  minifier, no CSS toolchain, no import map, no package manifest, and no
  subresource from any origin. The documentation links the panel carries out to
  the forge stay links an operator clicks.
- **Container widths from 360 px to wide.** The layout is specified over the <!-- cond: cond-2609201321056390 -->
  width a pane's container is given — a narrow window, a split view, a full
  screen, a large display — and not over a list of device breakpoints. 360 px is
  the narrow end that must work; the wide end is whatever the display offers.
- **System colours first; the triad as mark and status only.** The palette is <!-- cond: cond-2609201321051066 -->
  the Mac's, including the accent the operator chose. The three-form mark's
  yellow, red and blue (#F5C518, #E63329, #2B5DAA) appear in the mark itself and
  as the three status meanings, and nowhere else.
- **This builds on the planned panel intent and lands after it.** The <!-- cond: cond-2609201321050310 -->
  information architecture — the re-homing of controls, the inventory, the
  removal of `alert()` — is itd-2609100519003748 and its spec, and is not
  respecified here. That intent lands first; this one follows as its own lane
  and takes the arrangement it delivers as given.
- **The renderer trigger is written down, not exercised.** The hand-managed <!-- cond: cond-2609201321052790 -->
  re-render stays for this intent. Vendoring Preact and htm (roughly 4 kB, still
  no build step and nothing fetched) is pre-committed for the moment the
  hand-written exceptions to the settings-touched lock reach three, or a live
  per-model row needs an editable field — whichever comes first.
- **The chat client is out of scope.** Dessau Chat is the Swift client and is <!-- cond: cond-2609201321058953 -->
  governed by its own records; someone using it never sees the server's surfaces.
  Nothing here changes the client, and no styling crosses between them beyond
  the shared mark.
- **Pre-1.0.** No migration code and no compatibility shim. The markup, the <!-- cond: cond-2609201321054714 -->
  stylesheet, the module layout and the panel's element ids may all change
  outright; nothing stored on disk changes, which is why the impact is additive.

## Acceptance Criteria

- **Given** the panel's stylesheet, **when** it is read, **then** it is organised
  into cascade layers, declares `color-scheme: light dark`, and carries one token
  block whose colours are `light-dark()` values drawn from the system palette and
  the system accent, with no second hand-maintained dark block and a rule set for
  increased contrast. *Held by:* an assertion over `internal/ui/static/style.css`
  — the layer statement, the colour-scheme declaration, the single token block,
  no duplicated `prefers-color-scheme: dark` palette — plus one recorded hand
  check in the shipping record: the panel opened under light, dark and increased
  contrast with a non-default accent colour.
- **Given** any tab of the panel, **when** it is laid out at a container width of
  360 px, at a middle width and at a wide one, **then** it reflows to that width
  with no horizontal scrolling and no control clipped or pushed out of reach.
  *Held by:* a node-lifted layout test that evaluates each pane's container
  queries and track lists at the three widths and asserts the resolved column
  count and minimum content width, plus one recorded hand check of the panel in a
  macOS split view — named as a hand check, in the way this ledger records its
  other hand checks.
- **Given** every interactive control on the panel, **when** the markup and the
  stylesheet are read, **then** each target is at least 44 px in both directions,
  a `:focus-visible` ring is drawn on every focusable element and no rule removes
  it without replacing it, the page honours `env(safe-area-inset-*)` with
  `viewport-fit=cover`, and every transition and animation sits inside a
  `prefers-reduced-motion: no-preference` block. *Held by:* an assertion over
  `internal/ui/static/index.html` and `internal/ui/static/style.css`, enumerating
  the controls the way the existing settings assertions enumerate them.
- **Given** the tab strip, **when** it is worked by keyboard, **then** it carries
  `role="tablist"` with each tab `role="tab"`, `aria-selected` and
  `aria-controls`, each pane `role="tabpanel"` labelled by its tab; Left, Right,
  Home and End move the selection, activation is automatic on focus, and focus
  follows the selected tab. **And given** state that Alice did not initiate — a
  download finishing, a model loading, a save failing — **when** it arrives on the
  event stream, **then** its sentence is written into exactly one `role="status"`
  region, which the two-second statistics refresh never writes to. *Held by:* an
  archtest of the same string-scanning kind as the surface guard in
  `internal/archtest/settings_surface_test.go`, plus a node-lifted test on the
  key handler and on the announcement path.
- **Given** the panel's static directory, **when** it is read, **then** the script
  is a set of native ES modules split by tab, loaded with `type="module"` and
  relative specifiers only: no inline event handler in the markup, no import map,
  no bare specifier, no script or style from any origin. *Held by:* the settings
  surface guard taught to scan the directory of scripts rather than the single
  file it names today, plus the eight node-lifted tests in `internal/ui`
  importing the DOM-free module instead of lifting its text out of the source.
- **Given** this Mac with networking switched off, **when** the panel is opened
  and every control on every tab is worked, **then** the page renders completely
  and each control does its job. *Held by:* an assertion that no subresource in
  the markup or the stylesheet names a host — no `src`, `href`, `@import` or
  `url()` reaching off this Mac, the forge documentation links staying anchors an
  operator clicks — plus one recorded hand check with networking off.
- **Given** the markup and the stylesheet, **when** every colour in them is
  enumerated, **then** #F5C518, #E63329 and #2B5DAA occur only in the mark and in
  the three status meanings, and every other colour resolves to a system colour,
  the system accent, or a token derived from one of those. *Held by:* an
  assertion over `internal/ui/static/index.html` and
  `internal/ui/static/style.css` holding each occurrence of the three values to
  that allow-list.
- **Given** the menu-bar dropdown, **when** its items are read, **then** they are
  the status line, the endpoint, open panel, copy URL and quit, worded in the
  panel's vocabulary, under the solid-cube template icon of
  itd-2609200827202340. *Held by:* a Go test over the item list built in the
  command package's menu-bar source, asserting the item titles against the same
  vocabulary list the panel's markup is held to.
- **Given** any verb of the server binary, **when** a person reads its human
  output, **then** it uses the panel's nouns, prints its tables in aligned
  columns, and states any refusal in one sentence that names the next step.
  *Held by:* golden-output tests per verb, beside the existing lifecycle
  testdata, plus the shared vocabulary assertion that also holds the panel and
  the menu bar.
- **Given** this intent's spec, **when** a reviewer reads it, **then** it records
  the renderer trigger in full: vendor Preact and htm when the hand-written
  exceptions to the settings-touched lock reach three, or when a live per-model
  row needs an editable field. *Held by:* the spec section itself plus a test
  that counts the exceptions written beneath the settings-touched lock in the
  panel's settings module and fails above two.
- **Given** `docs/`, **when** this lands, **then** the control-panel page
  (`docs/control-panel.md`, delivered by itd-2609100519003748) describes the
  adaptive layout, the system appearance and the keyboard contract, and every
  page describing the menu bar or the terminal verbs uses the same words the
  panel does. *Held by:* the updated pages plus a listed sweep in the shipping
  record naming each page checked; `docs/lifecycle.md`,
  `docs/lifecycle-reference.md`, `docs/getting-started.md`,
  `docs/bind-address.md`, `docs/request-statistics.md`, `docs/mesh-vpn.md` and
  `README.md` are the candidate set.

## Open Questions

None: the third interview of 2026-09-20 answered the routing, the stack, the
reach, the surfaces, the palette and the sequence; see DECISIONS.md.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: the panel is the product's face for testers and it looks unfinished; a native-feeling, adaptive panel is what makes people trust the server; wrong if testers still reach for the terminal or the config file after it ships
