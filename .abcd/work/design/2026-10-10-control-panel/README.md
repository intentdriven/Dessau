# Design history: the control panel's look

- **Question:** which design should the redesigned control panel follow?
- **Serves:** itd-2609100519003748 (an intuitive control panel), and the window intent itd-2609201315575657 for how it looks and feels.
- **Status:** decided on 2026-10-11, after two rounds. Design 8 is the base, with the maintainer's changes and three borrowings (see [Decision](#decision)).
- **Judge:** the maintainer, as product thinker.

## Criteria

Fixed before any design was judged in the assessment, each scored 1 (poor) to 7 (excellent):

| Criterion | What it means |
|---|---|
| Fits the brief | Does what the panel is for: is all well, models, sharing, settings. |
| Easy to find things | You can tell where to go for a task without hunting. |
| Clear what matters | The most important thing on each screen stands out. |
| Comfortable to read | Text size, contrast and spacing are easy on the eye. |
| Looks right | Feels like a finished, trustworthy Mac product. |

## Round 1 (2026-10-10): five directions

Brief: [brief-round-1.md](brief-round-1.md). Each design is one self-contained page; open it straight from disk.

| # | Design | File |
|---|---|---|
| 1 | Native sidebar | [round-1/dessau-mockup-1-native-sidebar.html](round-1/dessau-mockup-1-native-sidebar.html) |
| 2 | Ops console | [round-1/dessau-mockup-2-ops-console.html](round-1/dessau-mockup-2-ops-console.html) |
| 3 | Calm editorial | [round-1/dessau-mockup-3-calm-editorial.html](round-1/dessau-mockup-3-calm-editorial.html) |
| 4 | Bauhaus cards | [round-1/dessau-mockup-4-bauhaus-cards.html](round-1/dessau-mockup-4-bauhaus-cards.html) |
| 5 | Master–detail | [round-1/dessau-mockup-5-master-detail.html](round-1/dessau-mockup-5-master-detail.html) |

Verdict, in the maintainer's words: "None of them are great. Try five more, this time using larger text, a card-based metaphor, using current SOTA design principles. [Design 5] goes in the right direction but is confusing, too small, and not intuitive enough."

## Round 2 (2026-10-10): cards, larger, intuitive

Brief: [brief-round-2.md](brief-round-2.md). All five rework design 5's idea (pick a model, see everything about it) under the new principles.

| # | Design | File |
|---|---|---|
| 6 | Model cards with a side sheet | [round-2/dessau-mockup-6-model-cards-sheet.html](round-2/dessau-mockup-6-model-cards-sheet.html) |
| 7 | Bento home | [round-2/dessau-mockup-7-bento-home.html](round-2/dessau-mockup-7-bento-home.html) |
| 8 | Tahoe glass | [round-2/dessau-mockup-8-tahoe-glass.html](round-2/dessau-mockup-8-tahoe-glass.html) |
| 9 | Questions first | [round-2/dessau-mockup-9-questions-first.html](round-2/dessau-mockup-9-questions-first.html) |
| 10 | Two-pane cards (5 redone) | [round-2/dessau-mockup-10-two-pane-cards.html](round-2/dessau-mockup-10-two-pane-cards.html) |

## Assessment (2026-10-11)

All ten designs, both rounds, were judged together in one sitting, with names hidden behind letters until the end. The method: criteria fixed first; a first impression of each (good, maybe, no); 30 side-by-side pairs ("which better serves the brief?"), 5 of them unannounced repeats with the sides swapped; the top three scored against the criteria with a reason each; a ranking fitted from the pairs (Bradley-Terry, with a 90% range from 500 resamples). The full result is [assessment/2026-10-11-protocol-a.json](assessment/2026-10-11-protocol-a.json).

| Rank | Design | Round | Could rank | Won–lost | First impression | Mean score |
|---|---|---|---|---|---|---|
| 1 | 8 · Tahoe glass | 2 | 1–3 | 5–0 | good | 5.8 |
| 2 | 7 · Bento home | 2 | 1–4 | 5–1 | good | 5.2 |
| 3 | 6 · Model cards with a side sheet | 2 | 2–5 | 5–1 | maybe | 5.8 |
| 4 | 10 · Two-pane cards (5 redone) | 2 | 2–5 | 6–1 | good | — |
| 5 | 9 · Questions first | 2 | 4–8 | 3–3 | maybe | — |
| 6 | 5 · Master–detail | 1 | 5–8 | 2–5 | maybe | — |
| 7 | 2 · Ops console | 1 | 4–9 | 2–3 | maybe | — |
| 8 | 1 · Native sidebar | 1 | 5–9 | 2–4 | maybe | — |
| 9 | 3 · Calm editorial | 1 | 7–10 | 0–6 | no | — |
| 10 | 4 · Bauhaus cards | 1 | 8–10 | 0–6 | no | — |

The top three against the criteria:

| Design | Fits the brief | Easy to find things | Clear what matters | Comfortable to read | Looks right | Why, in the maintainer's words |
|---|---|---|---|---|---|---|
| 8 · Tahoe glass | 6 | 4 | 6 | 6 | 7 | "appears most polished, Mac like. Sidebar useful, but needs to be toggable; buttons don't feel native" |
| 7 · Bento home | 6 | 4 | 6 | 6 | 4 | "menu bar breaks on phone" |
| 6 · Model cards with a side sheet | 6 | 6 | 6 | 6 | 5 | "overall simplicity; works on phone well" |

What the numbers say, and how far they go:

- **Round 2 beat round 1 outright.** All five round-2 designs rank above all five round-1 designs, and design 10, the rework of 5, beat 5 both times they met.
- **The judge was consistent:** all 5 repeated pairs were answered the same way, and the pairs hold no loops (A beat B, B beat C, C beat A). The repeats came 4 to 10 pairs after their originals, so memory may account for some of that.
- **The pairs did not separate the top four.** 8, 7, 6 and 10 could each rank anywhere from first to fifth.
- **The pairs were judged at a glance:** about 2 seconds each, side by side at desktop size, where text shows at half size. The pairs follow looks; the criteria showed the rest. Design 8 scored highest on looks (7) and lowest of the three on finding things (4), and design 6 matched its mean score.

## Decision

**Design 8 (Tahoe glass) is the base.** "8 is the most polished overall."

The maintainer's changes to it, in their words, then as requirements:

> Sidebar must be toggable; main menu bar wasts space on lables (Models & sharing), could be icons instead; buttons don't feel 'native'; they should be placed lower left corner etc.; when cards expand, this should be smoothly animiated, not abrubt; fixed status bar (e.g. Not saved) should be more distinct from cards; cards should ideally use more screen real estate (on larger screens).

1. The sidebar can be shown and hidden.
2. The top navigation drops its group labels ("Models", "Sharing") in favour of icons, so it takes less room.
3. Buttons look and behave like a Mac's own, and sit where 9 places them (see the borrowings).
4. A card that expands animates smoothly rather than jumping (and still respects a reduced-motion setting).
5. The fixed save bar ("Not saved…") is clearly set apart from the cards beneath it; in 8 it reads as one more card laid over the settings.
6. On larger screens, cards use more of the width.

Borrowed from other designs:

| From | Part | In the maintainer's words |
|---|---|---|
| 9 · Questions first | The detail sheet that opens on the right (e.g. Settings → "Who can reach this server"), and the position and style of its buttons | "copy from 9: Details sidebar to the right & position/style of buttons" |
| 9 · Questions first | The statistics bar chart: one bar a day, weekends paler, today darkest, a labelled scale | "bar charts also nicer than 8" |
| 7 · Bento home | The "More" button on a card that pops out a short menu (everything about it, its settings, delete) | "copy from 7: More buttons pop-out" |

The maintainer marked these parts with screenshots of the designs during the debrief; the screenshots were not kept, so each part is named above by design and screen.

## What this teaches the next round

The lasting lessons, with their reasons, are in [the taste file](../../../development/design/taste.md).

## Known gaps

- **Nothing here was checked in a browser by an agent.** The designs were read, not rendered, by the agents that made them; the maintainer's own viewing is the only rendering check. The agents' own notes say 8's glass was checked for contrast only with the default blue accent, and 9 and 10 are tight at phone width.
- **The assessment rests on one judge in one sitting.** Its consistency figures are good, but the top four were not separated.
- **The status shapes** (ready a blue square, working a yellow triangle, failed a red circle) are the agents' choice, carried in every round-2 design, and have not been put to the maintainer.

## Next steps

- The panel spec (spc-2609201007368529) is re-planned to carry the Overview and this reference.
- The window intent's "native feel" criteria take the button and sidebar requirements above.

## Provenance

- **Designs:** generated on 2026-10-10 by Claude agents, one design each, from the two briefs; the model was not recorded per design.
- **Assessment:** the maintainer, 2026-10-11, in an HTML assessment page (design-assess 0.1.0) built in abcd lab lab-261011111706-9fed999, which also holds the tool, its self-test and the session's measures. The judge's own debrief of the tool is recorded there.
