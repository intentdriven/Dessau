---
id: itd-2609200827202340
slug: one-mark-of-three-forms-replaces-the-four-form-tile-alice-se
spec_id: spc-2609201011302811
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# One mark of three forms replaces the four-form tile. Alice sees the same mark everywhere the product shows its face: the server's app icon, the chat client's app icon, the website's hero and wordmark, and the README. The mark is the triad of the school at Dessau: a yellow triangle, a red square and a blue circle on the dark tile, the grey square of today's icon dropped, so a three-year-old can name the parts and a designer can name the source. The chat client's three response styles, Square, Circle and Triangle, are drawn with the same three forms, so a style's mark is a piece of the logo rather than a separate glyph. The website keeps its current path until the rename lands and moves with it; the mark is the same on both.

## Press Release

Alice installs Dessau on the Mac under her desk and drags it to
`/Applications`. The tile that appears in the Dock is a small dark cube, tilted
a little off the vertical, with one form cut into each of its three visible
faces: a yellow triangle on the top, a red square on the left, a blue circle on
the right. She has never read a word about the product's design and she can
still describe the icon to Bob over the phone in one sentence. When Dessau
starts, the same cube appears in the menu bar — this time as a plain
silhouette in one colour, its top face a shade lighter, sitting quietly among
the system's own glyphs and turning white when she switches the Mac to dark
mode, because macOS draws it in the bar's own colour rather than the product's.

Bob, on the Mac in the next room, installs the chat client. Its tile in the
Dock is the same cube, drawn a little smaller inside the outline of a speech
bubble. He does not have to be told the two applications belong together: one
is the cube, the other is the cube in a bubble, and the family resemblance does
the work that a shared word would otherwise have to do. When Bob later picks
how he wants the model to answer, the three marks he chooses between — a
square, a circle, a triangle — are the same three forms, taken out of the cube
and set side by side. Nothing new has been invented for them.

Carol, who has installed nothing, meets the product first on its landing page.
The hero shows the mark; the wordmark in the top bar shows it small beside the
name; the README on the forge shows the same drawing again. By the time she
downloads the server, the tile that lands in her Dock is one she has already
seen three times. Previously each of these surfaces carried its own variation
of a four-square tile — a grey square that meant nothing, a bar-chart bubble
on the chat client, a menu-bar glyph with no relation to either — and a person
moving between them had to be told, each time, that this was the same product.

There is one drawing now, in three renderings, and each rendering is a file in
the repository that every raster is built from.

## Why This Matters

The argument is one mark, three surfaces, one source.

**One mark.** A product that shows four different faces is a product a person
has to learn four times. Today the server's tile, the chat client's tile, the
menu-bar glyph and the landing page's mark share a palette and nothing else;
the grey square in the server's tile is the clearest evidence that the drawing
was assembled rather than designed, because it is the only element nobody can
say the meaning of. Recognition is the whole job of a mark, and recognition is
cumulative: the same drawing seen on the page, in the README, in the Dock and
in the menu bar compounds, where four related drawings do not.

**Three forms, and their source.** The triangle, the square and the circle with
yellow, red and blue are the teaching triad of the school at Dessau, the school
this product and its component names are named after (adr-2609200729102059).
They earn their place twice over: a child can name a triangle, a square and a
circle, and a designer can name where they come from. That is a rare
combination — a mark that is legible without explanation and still has an
answer when someone asks why these shapes. The cube is what makes it a mark
rather than a poster: it gives the three forms a single object to live on, and
that object survives being shrunk to a menu bar as a silhouette when the forms
themselves would not.

**The styles' glyphs are pieces of it.** Decision 5 of adr-2609200729102059
settled that a person-facing mode carries a shape from the school's vocabulary
and never a person's name, and itd-2609200850330402 builds the three answer
styles on exactly that: Square, Circle, Triangle. If the mark is the four-form
tile, those glyphs are a separate small alphabet that happens to rhyme with the
logo. If the mark is this cube, they are the logo taken apart — Bob choosing a
style is choosing a face of the icon he already has in his Dock. The mark stops
being decoration on the outside of the product and starts doing work inside it.
That is only true if the drawing is decided first, which is what this intent
decides.

**One source, or it drifts.** The reason this is worth writing down rather than
simply drawing is that the product has already proved it drifts: the control
panel's header mark once carried a red circle where the icon had a yellow
triangle, and it took an architecture test to notice. Four surfaces drawn by
hand from a shared idea diverge; four surfaces derived from one committed file,
with a test holding each to it, cannot.

## Mechanism

We expect the four surfaces to stay identical to the accepted drawings, and to
stay identical to each other over time, because each rendering has exactly one
committed SVG source and every raster is derived from it by the icon script
that already exists — `build/mkicon.sh` for the server's tile and
`client/mkicon.sh` for the chat client's — so an icon that differs from its
drawing can only come from someone editing the drawing.

We expect an edited drawing with a stale raster to be caught rather than
shipped, because the hash-pinning architecture test already holds this seam:
`build/mkicon.sh` records the SHA-256 of the art it rasterised in
`build/AppIcon.icns.source-sha256`, and `TestAppIconMatchesTheCommittedArt` in
`internal/archtest/icon_test.go` fails when that recorded hash and the current
`build/icon.svg` disagree. The same shape extends to the client's icon, which
has the script but not yet the pin.

We expect the menu-bar cube to be readable in a light bar and a dark one
without a second drawing, because it is installed as a macOS *template* image:
`cmd/dessau/menubar.go` calls `systray.SetTemplateIcon`, and `cmd/dessau/icon.go`
documents the embedded art as pure black plus an alpha mask, which macOS
recolours for the bar it is drawn in. This is also why the menu-bar rendering
drops the three coloured forms: a template image has no colour to drop them
into, so the silhouette with its top face lifted by opacity is the only
rendering that can carry the mark there at all.

We expect the three cut forms to be legible at 32 px and the silhouette at
18 pt because the accepted board shows them at those sizes and the maintainer
accepted them as drawn (DECISIONS.md, 2026-09-20): the forms are cut at roughly
a third of each face and filled with a fully saturated colour against
near-black, and the silhouette is a single convex outline with one internal
opacity step.

What would show this wrong: the cut forms reading as noise rather than three
shapes at 32 px in the Dock; the bubble outline breaking up at 16 px at the
Dock's smallest size; the menu-bar silhouette reading as a filled blob, or its
lifted top face vanishing, once macOS recolours it for a dark bar; or any
surface needing a hand-tuned second drawing to be legible, which would mean
there is no longer one source.

## Scope Conditions

- The geometry is the three accepted drawings and nothing else: <!-- cond: cond-2609201011305232 -->
  `g3-inverse-default-mark.svg`, `chat-app-icon-bubble-frame.svg` and
  `server-menu-bar-cube-template.svg` under
  `.abcd/development/research/evidence/2026-09-20-mark-icons/`. The
  implementation copies their numbers — the −8° rotation, the face fills
  `#3d3e46` / `#26272d` / `#2f3037` on the `#17181d` tile, the cut forms in
  `#F5C518`, `#E63329` and `#2B5DAA`, the chat mark at six tenths inside a
  28-unit white outline on the 1024 grid, the menu-bar top face at 35 %
  opacity — rather than redrawing them.
- The surfaces this intent changes are exactly six and no others: the server's <!-- cond: cond-2609201011307657 -->
  app icon (`build/icon.svg` and the committed `build/AppIcon.icns`), the chat
  client's app icon (`client/icon/icon.svg` and its committed `AppIcon.icns`),
  the menu-bar template glyph embedded by `cmd/dessau`, the landing page's
  hero mark and its wordmark dot (`site-src/index.html.tmpl`), and the README.
  The control panel's header mark and favicon follow the same source and are
  already held to it by an architecture test.
- The three answer styles reuse these three forms and their fills; this intent <!-- cond: cond-2609201011309024 -->
  owns the forms, and itd-2609200850330402 owns the control they sit in. No
  fourth form and no alternative palette is minted for the styles.
- The website keeps its current path until the rename to Dessau lands on its <!-- cond: cond-2609201011306035 -->
  own branch, and moves with it; the mark is the same drawing on both sides of
  that move. Prose written here uses the current name, Dessau, and where a
  file or bundle name carries the family name it follows the rename rather
  than anticipating it.
- No new dependency is added to rasterise anything. `rsvg-convert` stays a <!-- cond: cond-2609201011302195 -->
  by-hand tool invoked through `make icon`, never reached by `make app` or by
  the release workflow, which is what the existing architecture test already
  enforces.
- Pre-1.0: the grey square is dropped outright and the four-quadrant <!-- cond: cond-2609201011300020 -->
  arrangement disappears with it. The existing site test that asserts four
  forms in four quadrants is replaced, not extended; no surface keeps a
  compatibility rendering of the old tile.

## Acceptance Criteria

- Given the three accepted drawings in the evidence folder, when the
  architecture tests run, then `build/icon.svg` carries the default mark's
  shape set and fills and `client/icon/icon.svg` carries the chat rendering's,
  each matching the drawing it was copied from — held by a new architecture
  test over the source SVGs in `internal/archtest`, alongside the existing
  icon test.
- Given a clean checkout with no librsvg, when `make app` builds the bundle,
  then the bundle carries the committed `AppIcon.icns`, the recorded source
  hash in `build/AppIcon.icns.source-sha256` equals the current
  `build/icon.svg`, and nothing in the build reaches `mkicon.sh` — held by
  `TestAppIconMatchesTheCommittedArt` and `TestAppIconIsCommittedAndBuildable`
  in `internal/archtest/icon_test.go`.
- Given the chat client's bundle, when its build runs, then its
  `AppIcon.icns` is the committed one and its recorded source hash equals
  `client/icon/icon.svg` — held by extending the hash pin and its architecture
  test to the client's icon, which today has the script but no pin.
- Given the menu-bar item, when the server starts, then its image is installed
  as a template image and the embedded art carries no brand colour — held by
  an architecture test over `cmd/dessau` asserting the `SetTemplateIcon` call
  and that the embedded glyph is single-channel black plus alpha.
- Given the rendered landing page, when the site tests compare its hero mark
  and its wordmark dot with the mark's source file, then both carry the three
  forms in the cube's arrangement with the three fills, and neither carries a
  grey form — held by `internal/sitetest`, replacing
  `TestMarkMatchesTheAppIcon`'s four-quadrant assertion.
- Given the README, when the documentation-currency check runs, then no
  user-facing line describes a four-form tile, a grey square or a bar-chart
  chat icon, and the mark the README shows is the one under `build/` — held by
  `abcd docs lint` together with the docs architecture tests.
- Given a real Apple Silicon Mac, when the maintainer looks at all three
  renderings — the server tile and the chat tile at 32 px in the Dock and in
  Finder, and the menu-bar glyph at 18 pt — in both light and dark appearance,
  then each reads as its drawing: three named forms on the cube, an
  unbroken bubble outline, and a silhouette whose top face is still visibly
  lighter. Held by a named hand check, `hand-check: mark at 32 px and 18 pt,
  light and dark`, dated and recorded beside the drawings in the evidence
  folder, since no automated test sees what a menu bar renders.

## Open Questions

**Answered.** Every question this draft held is decided: the three renderings
and their geometry by the maintainer's decisions of 2026-09-20 in
`.abcd/work/DECISIONS.md`, bound below to the three drawings under
`.abcd/development/research/evidence/2026-09-20-mark-icons/`, which are the
reference geometry the criteria above name. Nothing is left open.

**Decision 2026-09-20 (maintainer, on the mock's icon boards):** the "G3 inverse" variation is the default mark; the chat client's app icon uses a smaller bubble frame; the server's menu-bar icon is the solid cube. Bound to the mock (Model Picker Mock, board 11 `Decided.dc.html`, added 2026-09-20): "G3 inverse" is board 10's G3 variation (`IconsG.dc.html`: dark faces, the three forms cut to colour); the chat icon is that mark inside a tighter white bubble outline than board 10's Chat B; the menu-bar icon is board 10's cube silhouette template glyph (one colour, no cuts, the top face lifted). One acceptance criterion per rendering when planned. Recorded in `.abcd/work/DECISIONS.md`.

### The three renderings, bound to their drawings (2026-09-20)

The maintainer's decision names three renderings; each is bound here to the
drawing under `.abcd/development/research/evidence/2026-09-20-mark-icons/`
so a criterion can name it when this draft is planned:

- **The default mark, "G3 inverse":** `g3-inverse-default-mark.svg`, a tilted
  dark cube with one form cut into each visible face and filled with its
  colour (yellow triangle on top, red square left, blue circle right). This
  is the server's app icon and the mark on the website and the README.
- **The chat app icon, smaller bubble frame:** `chat-app-icon-bubble-frame.svg`,
  the same mark inside a speech bubble drawn as a white outline, the frame
  kept small so the cube stays the size of the server's.
- **The server's menu-bar icon, solid cube:** `server-menu-bar-cube-template.svg`,
  the cube's silhouette as a single-colour template glyph at 18 pt, the top
  face lighter by opacity; no forms, since the menu bar has one colour.

One criterion per rendering when the draft is planned. The drawings are the
mock's boards 8 and 10 reduced to the three chosen; the board files
themselves are the maintainer's evidence, not the record's.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

**Accepted 2026-09-20:** the renderings on board 11 (`Decided.dc.html`) are accepted as drawn and are the reference drawings for the acceptance criteria. Recorded in `.abcd/work/DECISIONS.md`.

## Grounds

- pursued: the maintainer answered the open questions at the second interview of 2026-09-20 and the itd-1 sections were written from the answers; what would show this wrong is a criterion that cannot be held by the holder it names
