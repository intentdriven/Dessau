# SOTA — the control panel's UI stack, buildless or not (2026-09-20)

Research for the design decision taken the same day: the panel stays plain
HTML, CSS and JavaScript, on the evidence that every defect found in it — no
aria, no tablist, no live region, no focus management, no responsive layout,
no per-field error slots — is fixable in plain files. This note tests that
decision against the alternatives, against what "a modern look and feel across
device screens" concretely means in 2026, and against what a build step would
cost the source-scanning guard in `internal/archtest`. Claims carry a tier:
[EVIDENCE] a measurement, a spec or this repository's own source;
[CONSENSUS] independent practitioners agreeing; [CONTESTED] credible
disagreement, both sides given; [ANECDOTE] a claim with no method behind it.

The pass was run by the sota-researcher agent and the note written from its
result by the interviewing session; the sources are the agent's.

## What the panel is today

[EVIDENCE — this repository] Three files under `internal/ui/static`, embedded
with `go:embed` and served on loopback: `index.html` (649 lines, seven tabs,
three inline `onclick` handlers), `app.js` (2,516 lines, one classic script,
every function global), `style.css` (241 lines). The stylesheet carries exactly
two `@media` rules, both `prefers-color-scheme: dark`, and no
`:focus-visible`, no `prefers-reduced-motion`, no `@container` and no
`@layer`. The live view is an EventSource on `/api/events` whose handler calls
`render()`, which re-runs every renderer; the Statistics tab adds a two-second
`refreshStats` interval. Eight tests in `internal/ui` lift pure functions out
of `app.js` as text and evaluate them under node, skipping when node is absent
— and the CI workflow installs no node, so those tests are green-by-skip
there.

## Findings

1. **Keep the plain files. The defects are stylesheet and markup defects, and
   a framework fixes none of them.** [EVIDENCE] Every item on the defect list
   lands in a 241-line stylesheet or in seven `<section>`s: `role="tablist"`
   and arrow keys, a `role="status"` region, `:focus-visible`, a container
   query, an error `<p>` per field. [EVIDENCE] The WebAIM Million 2025 report
   (1,000,000 home pages, automated detection, March 2025) found 94.8 % with
   detected WCAG 2 failures, improved from 95.9 % in 2024 — the framework era
   has not made pages accessible, so no framework will hand this panel its
   aria. Component libraries can ship good defaults; adopting one is not the
   same as adopting them.
   <https://webaim.org/projects/million/2025>
2. **Split `app.js` into native ES modules. No build, no import map, and the
   tests get better.** [EVIDENCE] `<script type="module">` with relative
   specifiers needs nothing from the network; an import map is only needed for
   BARE specifiers, which is the CDN case this panel must not have. [EVIDENCE
   — Go source] Go's builtin MIME table maps both `.js` and `.mjs` to
   `text/javascript; charset=utf-8`, so modules serve correctly from the
   embedded file system with no handler change. The real prize is the tests:
   a DOM-free module (the formatters, `modelInfoLine`, `resourcesSummary`) can
   be imported by a test instead of extracted from source as text, which
   removes the extraction machinery and the class of failure where a test
   passes against a function the browser never loads. Two costs, both small
   and both one-off: module scope is not global, so the three inline `onclick`
   handlers stop resolving and become delegated listeners (a gain — inline
   handlers are also what would block a strict Content-Security-Policy on the
   panel, which it does not yet send); and the archtest guard must scan a
   directory of scripts rather than one named file.
   <https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/script/type/importmap>
3. **Modern CSS now covers what a UI framework would have been hired for
   here.** [EVIDENCE] Container queries are Baseline widely available (all
   engines since 2023); cascade layers and nesting likewise; `light-dark()` is
   Baseline newly available since May 2024; `popover` shipped in Safari 17;
   `<dialog>` is widely available; anchor positioning, `contrast-color()`,
   `text-wrap: pretty` and scroll-driven animations shipped in Safari 26.0 on
   15 September 2025; same-document view transitions became Baseline newly
   available when Firefox 144 shipped on 14 October 2025. [EVIDENCE — context]
   This panel's audience is the operator's own recent Safari or Chrome over
   loopback, not the open web, so "Baseline newly available" is a safe floor
   here in a way it is not for a public site — a distinction worth writing into
   the intent, because it is the whole reason the plain-files answer holds.
   <https://webkit.org/blog/17333/webkit-features-in-safari-26-0/>,
   <https://web.dev/blog/same-document-view-transitions-are-now-baseline-newly-available>,
   <https://developer.mozilla.org/en-US/docs/Web/CSS/color_value/light-dark>
4. **The accessibility work is specified, not invented — and none of it is
   framework work.** [EVIDENCE] The ARIA APG tabs pattern gives the whole
   contract: `tablist`/`tab`/`tabpanel`, `aria-selected`, `aria-controls`,
   Left/Right and Home/End, and automatic activation on focus when panels
   render without noticeable latency — which is true of seven already-rendered
   sections. [EVIDENCE] For the live view, MDN and the W3C technique ARIA22
   are explicit: `role="status"` is implicitly polite, wants
   `aria-atomic="true"` if the whole container should be read, and live
   regions should be kept to two or fewer. The two-second poll must therefore
   NOT sit inside a live region: only the events worth interrupting a person
   for (a download finished, a save failed, setup failed) are announced. A
   panel that announced its own polling would be the accessibility defect.
   <https://www.w3.org/WAI/ARIA/apg/patterns/tabs/>,
   <https://developer.mozilla.org/en-US/docs/Web/Accessibility/ARIA/Guides/Live_regions>,
   <https://www.w3.org/WAI/WCAG22/Techniques/aria/ARIA22>
5. **A modern look and feel, in 2026, for a settings-shaped panel: six moves,
   all frameworkless.** [CONSENSUS across web.dev, MDN, WebKit and the HIG]
   (a) Type and colour from the system: `ui-sans-serif`/`-apple-system` is
   already in place; add `color-scheme: light dark` so form controls and
   scrollbars follow, and collapse the duplicated dark block into `light-dark()`
   tokens. (b) Tokens as custom properties inside `@layer tokens` — the DTCG
   Design Tokens Format Module reached its first stable version, 2025.10, on
   28 October 2025, backed by Adobe, Google, Figma and others, but it is an
   INTERCHANGE format: it earns its keep when tokens cross a tool boundary, and
   here it buys only the naming discipline, so borrow the vocabulary and skip
   the tooling. (c) Adaptive layout from container queries plus
   `grid-template-columns: repeat(auto-fit, minmax(…, 1fr))`, not from a
   viewport breakpoint list: the model card, stat card and override row should
   re-lay-out on the width they are given, which is the difference between a
   panel that works in a split view and one that works at 980 px.
   (d) Touch and safe areas: `dvh` rather than `vh`, `env(safe-area-inset-*)`
   with `viewport-fit=cover`, and a minimum target of 44 px — the HIG figure —
   against WCAG 2.2 SC 2.5.8's 24 px AA floor. (e) `prefers-reduced-motion:
   no-preference` around the spinner, the progress bar transition and any view
   transition. (f) Apple's 2025 design language, Liquid Glass (WWDC25, June
   2025, across iOS, iPadOS and macOS 26), is NOT to be imitated on the web:
   there is no web primitive for the material, and `backdrop-filter`
   pastiches read as costume. What IS borrowable, and free, is the part Apple
   described in prose — bolder, left-aligned typography for clarity, and
   system colours retuned across Light, Dark and Increased Contrast.
   <https://www.w3.org/community/design-tokens/2025/10/28/design-tokens-specification-reaches-first-stable-version/>,
   <https://www.apple.com/newsroom/2025/06/apple-introduces-a-delightful-and-elegant-new-software-design/>,
   <https://developer.apple.com/design/human-interface-guidelines>
6. **A build step does not break the settings guard; it breaks the inference
   the guard licenses.** [EVIDENCE — `internal/archtest/settings_surface_test.go`]
   Both halves are string searches over source: the script must NAME the key,
   and every control id the script reaches for must exist in the markup. Run a
   bundler and those searches still pass — over files that are no longer the
   artefact the binary embeds. Restoring the inference needs one of three
   things, and each costs more than the panel is worth. Scan the BUNDLE:
   defeated by minification renaming the lookups and by any transformation of
   the object literal the guard walks. Commit the bundle and add a freshness
   check: the standard "regenerate, then diff" idiom cannot detect a generator
   that has stopped emitting a file, and it compares the tree to the last
   commit rather than to a fresh build. Or make the build hermetic and re-run
   it in CI: this repository's workflow installs no node today, so this means
   adding a pinned node toolchain and a lockfile to a Go CI to serve 2,500
   lines of script.
7. **The no-dependency constraint is not fastidiousness; it is the 2025–2026
   threat model.** [EVIDENCE] The self-propagating "Shai-Hulud" npm worm was
   discovered on 16 September 2025, harvesting CI and cloud credentials from
   roughly 40 packages and republishing itself through stolen tokens; CISA
   issued an alert on 23 September 2025; the 2.0 wave in late November 2025
   touched tens of thousands of repositories and carried a fallback that
   attempted to destroy the user's home directory. A product whose panel has
   zero JavaScript dependencies today would, with one bundler, acquire a
   transitive tree that executes install scripts on the maintainer's own Mac.
   That is the single heaviest weight on this scale, and it is evidence rather
   than taste.
   <https://unit42.paloaltonetworks.com/npm-supply-chain-attack/>,
   <https://www.cisa.gov/news-events/alerts/2025/09/23/widespread-supply-chain-compromise-impacting-npm-ecosystem>,
   <https://securitylabs.datadoghq.com/articles/shai-hulud-2.0-npm-worm/>
8. **Of the light frameworks, exactly one keeps the no-build promise.**
   [EVIDENCE — vendor documentation] Preact + htm is documented for use with
   no build tools, as two ESM imports and a tagged template; vendored into the
   static directory it stays offline, and it is the only candidate that adds
   keyed rendering without adding a compiler. Solid offers a tagged-template
   path, and its own README says the template form is inferior — larger
   bundles, slower, manual wrapping of values — unless there is a real reason
   such as a no-build requirement. Svelte and Solid-with-JSX ARE compilers:
   the build step is not optional, so they are out by the decision already
   taken. Alpine evaluates attribute expressions through `new Function`,
   requiring `unsafe-eval` unless the CSP build is used, and that build
   restricts to `Alpine.data` registration and read-only property access.
   <https://preactjs.com/guide/v10/getting-started/>,
   <https://github.com/solidjs/solid/blob/main/packages/solid/html/README.md>,
   <https://alpinejs.dev/advanced/csp>
9. **htmx is the wrong shape for THIS panel, and the case against it is
   partly its author's.** [CONTESTED] Practitioner reports on the Go, templ
   and htmx stack are genuinely positive for server-rendered CRUD, reporting
   thousands of lines of client script deleted — and the panel IS served by a
   Go binary, so the fit is not absurd. But this panel's live view is a single
   JSON snapshot pushed over SSE and rendered client-side; going htmx means
   moving seven tabs' rendering into Go templates and the event stream into
   HTML fragments, which rewrites the server's UI contract and moves the
   settings guard from scanning a script to scanning templates. The
   often-circulated "htmx sucks" essay is htmx's own author writing on
   1 February 2024; its substantive points stand anyway — no component model,
   no component library ecosystem, behaviour embedded in markup, and a second
   API still needed for non-browser clients.
   <https://htmx.org/essays/htmx-sucks/>,
   <https://thefridaydeploy.substack.com/p/my-6-months-with-the-goth-stack-building>
10. **Web components and Lit: real, buildless, and a poor fit here.**
    [EVIDENCE] Lit runs from an import map with no build, and can be vendored
    for offline use. [CONSENSUS] Nolan Lawson's "Web components are okay"
    (28 September 2024) places their value in cross-framework reuse,
    decade-scale backwards compatibility and marketplace components — none of
    which a single embedded panel with one consumer has — and names
    accessibility and SSR as the standing costs. His May 2025 update records
    that the fix for cross-root ARIA is still an origin trial, so shadow DOM
    makes the panel's outstanding aria work HARDER, not easier. Web Awesome,
    Shoelace's successor (public release 2025), is the honest shortcut to a
    designed look with no build — a stylesheet and a module script — and is
    the one option that would buy a modern appearance in a day. It is rejected
    here on three counts, not on quality: it is a dependency tree in a product
    that has none, its theme is not this product's palette, and offline means
    vendoring a component library into the binary.
    <https://nolanlawson.com/2024/09/28/web-components-are-okay/>,
    <https://nolanlawson.com/2022/11/28/shadow-dom-and-accessibility-the-trouble-with-aria/>,
    <https://blog.fontawesome.com/web-awesome-component-library/>

## What each candidate actually adds to THIS panel

| Candidate | Build step | Works offline when vendored | What it adds here | What it costs |
|---|---|---|---|---|
| Plain files + ES modules + modern CSS | none | yes | modularity, testable modules, adaptive layout, dark mode, motion prefs | the re-render problem stays hand-managed |
| Preact + htm, vendored | none | yes | keyed rendering: focus, selection and scroll survive a live update | ~4 kB dependency, renderers rewritten |
| Lit, vendored | none | yes | a component model, scoped styles | shadow DOM makes the aria work harder |
| Web Awesome | none | yes, if vendored | a designed look in a day | a component library in a dependency-free product |
| Alpine | none | yes | declarative bindings in markup | `unsafe-eval` or a restricted CSP build |
| htmx + Go templates | none | yes | deletes client rendering | rewrites the SSE contract and the settings guard |
| Svelte / Solid (JSX) | required | yes | the best ergonomics of the set | node in CI, a bundle, and a guard that proves less |

## Counter-arguments that survived

- **The live view is where plain files genuinely hurt.** [EVIDENCE — this
  repository] `render()` re-runs every renderer on every SSE message, the
  panel repaints through `innerHTML` in 33 places (31 assignments; the agent's first count of 82 was wrong and is corrected here from the tree), and `app.js` contains no
  `document.activeElement`, no `selectionStart` and no `scrollTop`
  preservation anywhere. The existing mitigation is a whole-form lock,
  `if (settingsTouched) return;` at the top of `renderSettings`, with two
  hand-written exceptions immediately beneath it — `refreshOverrideModels` and
  `renderBridgeState` — for live state that must keep updating while the lock
  holds. That lock is a coarse hand-rolled diff, and its exception list grows
  with every live field the panel gains; per-field error slots and a live
  per-model map are the next two. This is the one argument for a library that
  survives contact, and the fair version of it does NOT ask for a build step:
  Preact plus htm, vendored, is ~4 kB and keeps every other constraint intact.
  Against it: it means rewriting roughly 2,500 lines of renderers in one
  movement, which is the opposite of a script-first MVP, and it puts a second
  templating primitive beside the `innerHTML` strings until the last renderer
  is converted.
- **"Modularity" might be read as "components".** It is better served here by
  ES modules with clear seams — formatters, a state store, one module per tab —
  than by a component abstraction. A module boundary is testable by node
  today; a component boundary would need a DOM harness this repository does
  not have.
- **Without a framework the accessibility work is never done.** True in
  general; falsified in particular by APG, which specifies the tabs pattern
  completely, and by the panel's size. The risk is that it is never STARTED,
  which is an intent-and-test problem, not a stack problem: the same archtest
  approach that proves every setting has a control can prove every tab has
  `role="tab"` and every panel an `aria-labelledby`.
- **A build step would let the panel use TypeScript.** Real, and the largest
  thing given up. `// @ts-check` with JSDoc plus a type check run by hand
  recovers most of the checking without shipping compiled output — but it
  still puts node in the loop, so it is a separate decision, not a rider.

## Not worth adopting

A bundler or minifier for 2,500 lines served over loopback (it costs the
guard its inference and the CI a node toolchain). Tailwind or any CSS build
(the hand-written stylesheet is the product's look, not an accident). Svelte
or Solid-with-JSX (compilers; excluded by the no-build decision). Alpine
(`unsafe-eval`, or a CSP build that restricts what the panel can express).
htmx (rewrites the SSE contract to buy something the panel is not short of).
Shadow DOM (harder aria, for encapsulation a single embedded panel does not
need). A CDN import map (breaks the offline requirement outright — vendor or
do without). DTCG tokens as tooling rather than vocabulary (nothing crosses a
tool boundary here). Cross-document view transitions (Firefox has not shipped
them, and the panel is one document anyway). A service worker (there is no
network to lose on loopback).

## Recommendation

Keep the plain files; the decision taken today survives the challenge. Do it
in this order. First, the stylesheet: `@layer tokens, base, components,
utilities`; `color-scheme: light dark` with `light-dark()` collapsing the two
duplicated dark blocks; container queries plus `auto-fit` grids on the cards,
the override rows and the wide tables; 44 px minimum targets; `:focus-visible`
rings; `dvh` and `env(safe-area-inset-*)` with `viewport-fit=cover`; every
transition inside `prefers-reduced-motion: no-preference`. Second, the markup:
the APG tabs contract with arrow keys and automatic activation; one
`role="status"` region carrying events, never the poll; a per-field error slot
bound by `aria-describedby`; `<dialog>` for the destructive confirmations that
are currently two-click armed buttons. Third, the script: split `app.js` into
ES modules loaded with `type="module"` and relative specifiers, replace the
three inline `onclick` handlers with delegated listeners, and convert the
node-lifted tests to import the DOM-free module instead of extracting its
text. Teach the settings guard to scan the directory of scripts; add a sibling
guard for the tabs contract, so the accessibility promise is armed the way the
surface-sync promise is — a test, not a habit. Leave the renderer question
open and write its trigger down now: when the hand-written exceptions to
`settingsTouched` reach three, or when a live per-model row must keep an
editable field, vendor Preact and htm and convert renderer by renderer. That
reversal costs 4 kB and no build step, which is why it is the one worth
pre-committing to.

## Outcome

The maintainer adopted the recommendation at the third interview of
2026-09-20: itd-2609201315575657 commits to plain files, native modules and
modern CSS, with the Preact+htm trigger written into the intent.
