# Brief, round 2: Dessau control-panel mockups (cards, larger, intuitive)

> The brief given to the agents that generated round 2, on 2026-10-10. Machine
> paths are replaced by repository paths; nothing else is changed.

Read the round-1 brief first and follow every one of its hard rules and its
content list: brief-round-1.md
(one self-contained file each, no browser, no server, nothing loaded from the
network, system fonts, mock data with Alice/Bob/Carol and documentation
addresses only, British English, light and dark, reduced motion, keyboard,
360 px to wide, the eight tabs by name, the logo mark copied verbatim from
today's index.html). Do not overwrite any existing file.

## What the maintainer said about round 1

"None of them are great. Try five more, this time using larger text, a
card-based metaphor, using current SOTA design principles.
dessau-mockup-5-master-detail.html goes in the right direction but is
confusing, too small, and not intuitive enough."

Read round-1/dessau-mockup-5-master-detail.html before you start: keep what
is right about it (each model is the centre: pick a model, see everything about
it) and fix what is wrong (confusing, too small, not intuitive). Reuse its mock
data so all mockups describe the same server.

## Design principles for this round (apply all)

- **Larger type, fluid.** Body text at least 17 px (1.0625rem), rising with
  the window via clamp(); card titles 20–24 px; key figures 32–56 px. Line
  length about 60–75 characters. Never below 14 px for anything a person must
  read (captions included).
- **Cards as the metaphor.** Every unit of meaning is a card: a model, a
  device, a setting group, a fact. One idea per card, a clear title, the
  state in plain words, and at most one primary action on its face; secondary
  actions behind a "More" disclosure on the card. Generous padding (20–28 px),
  large radius (16–24 px), a soft elevation or a subtle border, an 8-point
  spacing grid. Bento-style grids where cards of different sizes show what
  matters most.
- **Progressive disclosure.** The first screen answers "Is everything OK, and
  what needs me?" Detail opens in place (a card expands) or in a sheet/side
  panel, with an obvious way back (Escape, a visible Close or Back). Nothing
  advanced shows until asked for.
- **Plain language, answer-first.** Card titles are what a person wants to
  know ("Who can reach this server", "Ready to answer"), not internal names.
  Statuses are words plus a shape/colour, never colour alone.
- **Obvious affordances.** Buttons look like buttons; the whole card is a
  target where it opens detail; hover and pressed states; focus rings.
  Touch/click targets at least 44 px (above WCAG 2.2's 24 px minimum).
- **Calm hierarchy.** One accent colour (the Mac's AccentColor where the
  browser has it, a system blue otherwise), neutrals for everything else, the
  brand triad only in the mark and the three statuses. Few lines and dividers:
  space separates. Empty, loading and error states for at least one card each.
- **Navigation stays simple.** The eight tabs exist by name (Overview, My
  Models, Find Models, Statistics, Connect, Clients, Posture, Settings), but you
  may group them visually (e.g. Overview · Models (My/Find) · Statistics ·
  Sharing (Connect, Clients, Posture) · Settings) so a newcomer sees about five
  places, not eight.
- **Accessible by construction.** WCAG 2.2 AA contrast, a visible
  :focus-visible ring not hidden by sticky bars, aria roles for tabs and
  dialogs/sheets, a polite live region for state changes.

## Report

Final message: for each file, its path, the design direction in one sentence,
its size in bytes, and anything from either brief you could not do.
