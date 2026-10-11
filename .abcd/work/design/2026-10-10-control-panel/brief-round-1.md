# Brief: Dessau control-panel mockups

> The brief given to the agents that generated round 1, on 2026-10-10. Machine
> paths are replaced by repository paths; nothing else is changed.

You are designing static HTML mockups of the redesigned web control panel of
Dessau, a macOS menu-bar app that downloads MLX models from HuggingFace and
serves them to the local network over an OpenAI-compatible API. The operator
(Alice) opens the panel in a browser on her Mac. The maintainer will choose one
mockup as the visual reference for the redesign intent, so each must look like
a real, finished product screen, not a wireframe.

## Hard rules

- Write ONLY the files named in your task, under the operator's Desktop. Read nothing outside
  the paths listed under "Sources", and write nothing anywhere else (your own
  temp files under $TMPDIR are fine). Before writing, check that the target file
  does not already exist; if it does, stop and report rather than overwrite.
- NO browser, NO server, NO opening tabs or apps. Do not use any browser or
  computer-use tool. Check your HTML by reading it, and if you want, with a
  quick syntax check (for example `python3 -I -c "import html.parser"` style
  parsing, or `xmllint --html --noout` if available).
- Each mockup is ONE self-contained .html file: inline CSS and inline JS, no
  external script, stylesheet, font, image or CDN. Use system fonts
  (-apple-system, BlinkMacSystemFont, "SF Pro", system-ui), inline SVG for icons
  and charts. It must work opened straight from disk with networking off.
- Mock data only, realistic: models such as mlx-community/Qwen3-8B-4bit,
  mlx-community/gemma-3-12b-it-4bit, mlx-community/Llama-3.2-3B-Instruct-4bit,
  a 64 GB Apple Silicon Mac, memory figures, tokens/s, request counts. Any
  address shown uses documentation values only (192.0.2.x, port 8080, or
  `alices-mac.local`). People: only Alice, Bob, Carol (e.g. "Bob's laptop",
  "Carol's iPad" as clients). No real names, paths or hostnames.
- British English in all visible text ("colour", "organise", "licence").
- Respect `prefers-color-scheme` (light and dark both designed) and
  `prefers-reduced-motion`. Keyboard: tabs are real buttons with
  role="tab"/aria-selected and arrow-key movement; a visible :focus-visible ring.
  Works from about 360 px wide (no sideways scroll) to wide screens.

## What every mockup shows (clickable, via a small inline script)

The panel has eight tabs: **Overview** (the default, read-only live summary),
**My Models**, **Find Models**, **Statistics**, **Connect**, **Clients**,
**Posture**, **Settings**. Every tab must have believable content; Overview,
My Models, Settings and Statistics in real depth:

- **Overview:** one-sentence summary ("Serving 2 models to 3 devices on your
  network"), health, memory in use against the budget (a meter), what is loaded,
  connections, recent changes (download finished, model unloaded after idle…).
- **My Models:** the resources roll-up at the head (memory budget, used, free),
  then model cards/rows: name, size, quantisation, state (loaded / idle /
  downloading 63 % / update available — "3 files changed"), Load/Unload, a
  per-model Settings entry, Delete with an inline, untimed confirmation (no
  alert/confirm dialogs).
- **Find Models:** a line about this Mac ("This Mac can run models up to about
  40 GB") above a search box and a few results.
- **Statistics:** a request/tokens chart (inline SVG), per-model table, the
  store line and a "Clear records" action with inline confirmation.
- **Connect:** the endpoint URL to copy (http://alices-mac.local:8080/v1), API
  key state, a short code example.
- **Clients:** devices seen (Bob's laptop, Carol's iPad…), last seen, revoke.
- **Posture:** what the server exposes and to whom, stated as facts.
- **Settings, by task, eight groups, each opening with a live "what is in
  force" line**, e.g. "Reachable from your network on port 8080". Groups (use
  these titles or close equivalents):
  A Who can reach this server · B Memory and how many models stay loaded ·
  C How requests are handled · D When idle models are unloaded ·
  E Checks and measurements (self-test, context probe) ·
  F Statistics and logging · G Discord bridge · H Downloading and updating models.
  Show a per-field error slot on one field with a server-style refusal
  ("The grace period cannot exceed the maximum wait of 300 seconds."), and a
  save bar whose Save button is never disabled.

## Sources (read-only)

- internal/ui/static/index.html — today's panel:
  the real settings, labels and hint wording. Reuse real setting names.
- internal/ui/static/index.html in the snapshot of lab lab-261009093853-9fed999
  — a lab prototype of the redesign with the eight Settings groups and an
  Overview pane. Content reference only; your visual design must be your own.
- .abcd/development/intents/planned/itd-2609100519003748-*.md
  and itd-2609201315575657-*.md — the intents (press release, scope conditions):
  system colours and the Mac's accent; the brand triad yellow #F5C518, red
  #E63329, blue #2B5DAA appears only in the logo mark and as the three status
  meanings (ready/working/failed). The mark: copy the inline SVG logo mark
  from today's index.html header verbatim; do not redraw it.

## Report

Final message: for each file, its path, the design direction in one sentence,
its size in bytes, and anything from this brief you could not do.
