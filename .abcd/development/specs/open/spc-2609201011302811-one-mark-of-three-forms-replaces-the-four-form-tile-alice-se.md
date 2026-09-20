---
id: spc-2609201011302811
slug: one-mark-of-three-forms-replaces-the-four-form-tile-alice-se
intent: itd-2609200827202340
origin: researcher-authored
production_mode: hand-written
---
# One mark of three forms, on every surface

## Summary

This spec delivers itd-2609200827202340 by replacing the four-form tile with
one drawing in three renderings, each rendering a committed SVG that every
raster is derived from: the default mark ("G3 inverse") at `build/icon.svg`,
the same mark inside a bubble frame at `client/icon/icon.svg`, and a
single-colour template glyph at `cmd/gropius/icon.svg`. The geometry is copied
from the three accepted drawings under
`.abcd/development/research/evidence/2026-09-20-mark-icons/` and is not
redrawn. The rasters stay by-hand work for `make icon` through the two scripts
that already exist; what is new is that each raster carries a recorded source
hash and an architecture test that fails when art and raster disagree — the
server has that pin today, the chat client and the menu bar do not. The
website's hero and wordmark, the control panel's header mark and favicon and
the README all take the same drawing, the grey square disappears, and the
site test that asserts four forms in four quadrants is replaced. Impact:
additive; one deliberate breaking change to a drawing nobody depends on
programmatically.

## Scope

In: `build/` (`icon.svg`, `AppIcon.icns`, its hash pin, `mkicon.sh`);
`client/` (`icon/icon.svg`, `icon/AppIcon.icns`, a new hash pin, `mkicon.sh`,
the icon fallback in `build.sh`); `cmd/gropius/` (a new `icon.svg` source,
the embedded `icon.png`, a new hash pin); `internal/archtest` (the source-SVG
test, the client's pin test, the template-image test, the control panel's
existing mark test); `internal/sitetest` (the mark test, replaced);
`site-src/` (`index.html.tmpl`, `site.css`); `internal/ui/static/index.html`
(header mark and favicon); `README.md`; the changelog.

Out: the rename to the new family name and every file or bundle name that
carries the current one (adr-2609200729102059, cond-2609201011306035); the
chat client's answer-style control, which spc-2609200945518522 owns
(cond-2609201011309024); any new rasterising dependency
(cond-2609201011302195); any compatibility rendering of the old tile
(cond-2609201011300020).

## Approach

### The three sources, copied rather than redrawn

Each rendering has exactly one committed source, and each source is the
evidence drawing's numbers (cond-2609201011305232). All three share one cube:
the faces `#3d3e46` (top), `#26272d` (left) and `#2f3037` (right) as the
polygons `0,-300 260,-150 0,0 -260,-150`, `-260,-150 0,0 0,300 -260,150` and
`0,0 260,-150 260,150 0,300`, the whole group rotated −8°.

**`build/icon.svg` — the default mark.** Replaced outright by
`g3-inverse-default-mark.svg`: the `#17181d` tile full-bleed on the 1024 grid
(macOS applies its own corner mask), the cube at `translate(512,540)
rotate(-8)`, and one form cut into each visible face — the yellow `#F5C518`
triangle `0,-105 95,60 -95,60` at `translate(0,-150)` under
`matrix(0.866,0.5,-0.866,0.5,0,0)`, the red `#E63329` 156-unit square at
`translate(-130,75)` under `matrix(0.866,0.5,0,1,0,0)`, and the blue `#2B5DAA`
circle of radius 86 at `translate(130,75)` under `matrix(0.866,-0.5,0,1,0,0)`.
The four-quadrant drawing and its grey square go; nothing keeps a copy.

**`client/icon/icon.svg` — the chat client's mark.** Replaced by
`chat-app-icon-bubble-frame.svg`: the same tile, a speech bubble drawn as a
`#ffffff` stroke of width 28 with `stroke-linejoin="round"` and no fill, its
path inset from the tile's edge with 90-unit corners and the tail dropping
from `H520` to `-160,140`, and the mark inside it at `translate(512,500)
rotate(-8) scale(0.66)` — six tenths, so Bob's chat tile carries a cube the
size of Alice's server tile inside a frame rather than a shrunken one. The
bubble-with-three-bars drawing goes.

**`cmd/gropius/icon.svg` — the menu-bar glyph, a new file.** The menu bar has
had a committed PNG and no source; this gives it one, from
`server-menu-bar-cube-template.svg`: 44×44, the cube at `translate(22,23)
rotate(-8) scale(0.062)`, the three faces in `#000000`, and the top face drawn
a second time for the lift. One restatement, and only one: the evidence draws
that lift as `#ffffff` at 35 % opacity, and a template image carries no white
(`cmd/gropius/icon.go`: pure black plus an alpha mask). The source therefore
draws the lifted top face as `#000000` at `opacity="0.65"` over transparency,
which is the same composite — a face 35 % of the way towards whatever the bar
is drawn on — expressed in the one channel a template image has. This is the
whole reason the menu-bar rendering drops the coloured forms: there is no
colour to cut them into.

### The rasters, by the scripts that exist

No new dependency (cond-2609201011302195). `build/mkicon.sh` keeps its
`rsvg-convert` → `sips` → `iconutil` path for `AppIcon.icns` and gains one
step: it rasterises `../cmd/gropius/icon.svg` to `cmd/gropius/icon.png` at
44×44 and records that source's hash beside it. One embedded PNG, not two —
`systray.SetTemplateIcon` takes a single byte slice and AppKit scales the
template to the bar's height, which is what the committed 44 px file already
relies on. `client/mkicon.sh` keeps its path and gains the hash record its
server counterpart has had all along. The Makefile's `icon` target runs both
scripts, so `make icon` remains the single by-hand verb; `make app`, `make
build` and the release workflow still reach neither, which
`TestAppIconIsCommittedAndBuildable` already enforces through `make -n app`.

One fallback goes with this. `client/build.sh` runs `./mkicon.sh` when
`icon/AppIcon.icns` is absent — a path the release workflow reaches when it
builds `GropiusChat.app`, on a runner with no librsvg, after the tag is
pushed. It becomes the same refusal the server's Makefile gives: the committed
`.icns` is restored, or `make icon` is run by hand.

### The pins and their tests

Three recorded hashes on one pattern: `build/AppIcon.icns.source-sha256`
(exists, rewritten by the script), `client/icon/AppIcon.icns.source-sha256`
(new) and `cmd/gropius/icon.png.source-sha256` (new). In `internal/archtest`,
`TestAppIconMatchesTheCommittedArt` gains the client's and the menu bar's
pins beside the server's — one table, three rows of source and pin, each
failing with the same sentence naming `make icon`.

A second new test, over the sources themselves, holds them to the evidence:
for `build/icon.svg` and `client/icon/icon.svg` it asserts that every shape
element of the drawing it was copied from is present with the same fill and
the same transform, normalised for whitespace and quoting the way the control
panel's test already normalises; for `cmd/gropius/icon.svg` it asserts the
cube's geometry and that no fill in the file is anything but `#000000`. The
evidence folder is committed, so the drawings are readable from the test.

### The menu bar is a template image

A new architecture test asserts both halves of what makes Alice's menu-bar
glyph legible in a light bar and a dark one without a second drawing: that
`cmd/gropius/menubar.go` installs the image through
`systray.SetTemplateIcon` at every call site that sets it, and that the
embedded `cmd/gropius/icon.png` decodes to pixels that are pure black
(R=G=B=0) wherever they are not fully transparent, with at least two distinct
non-zero alpha values — the faces at full alpha and the top face at its lift.
A single alpha value would mean the lift was lost; a non-zero colour channel
would mean a brand colour had been baked in and macOS would stop recolouring
it.

### The website, the control panel and the README

**Site.** `site-src/index.html.tmpl`'s hero mark (`svg class="mark"`) and
wordmark dot (`svg class="dot"`) are redrawn as the cube, both on
`viewBox="0 0 1024 1024"` so they carry `build/icon.svg`'s own numbers — the
same device that makes the control panel's drift a textual difference a test
can see. `site.css` loses `--mark-grey` from all three theme blocks and gains
`--face-top`, `--face-left` and `--face-right` for the cube's greys; its
header note, which today says the four mark colours are not free choices, is
rewritten to the three forms' colours and the three faces. The pillar glyphs
take the mark's own pairing — triangle yellow, square red, circle blue — in
place of today's square-blue and circle-red.

**The site test, replaced not extended (cond-2609201011300020).**
`TestMarkMatchesTheAppIcon` and its four-quadrant `shapesIn` helper go.
`TestMarkCarriesTheThreeForms` takes their place: it reads `build/icon.svg`,
asserts it carries exactly three cut forms — a polygon, a rect and a circle —
in `#F5C518`, `#E63329` and `#2B5DAA` on three face polygons, then asserts
the hero mark and the wordmark dot each carry the same three shape kinds with
the same coordinates, that each fill is a theme token whose value in both dark
blocks equals the icon's fill, and that no grey form and no `--mark-grey`
token survives anywhere under `site-src/`.

**Control panel.** Its header mark and favicon are redrawn from the same file.
`TestControlPanelMarkMatchesTheAppIcon` fails the moment `build/icon.svg`
changes, so it moves with this change rather than after it: its `len(want) !=
4` becomes the drawing's six shapes above the ground, and its shape regexp
learns the per-form `<g transform=…>` wrappers, since a form's transform is
part of what must not drift.

**README.** The centred header gains the mark itself — `<img
src="build/icon.svg">` at 128 px, alt text naming the three forms — so Carol
meets on the forge the drawing that later lands in her Dock, and there is no
second copy of the art to go stale. No user-facing line anywhere describes a
four-form tile, a grey square or a bar-chart chat icon; `docs/` carries no
such prose today and gains none.

### The styles' glyphs, and the rename

Bob's three answer styles are drawn with these three forms and these three
fills. This spec owns the forms; spc-2609200945518522 (for
itd-2609200850330402) owns the control they sit in and is where their shapes
and colours are specified — nothing is restated here, and no fourth form and
no alternative palette is minted (cond-2609201011309024, and decision 5 of
adr-2609200729102059, which put a shape from the school's vocabulary where a
person's name would otherwise go).

Where a file or bundle name carries the family name it follows the rename
when that lands on its own branch, and not before (cond-2609201011306035).
The drawing is unchanged by the rename: the same three sources, the same
geometry, on both sides of the move. The website likewise keeps its current
path until then.

## How each acceptance criterion is held

The intent's criteria carry no stamped ids of their own — only its six scope
conditions do — so they are mapped here in the order they are written. Every
holder is watched failing before the change and passing after.

| # | Criterion | Held by |
|---|---|---|
| 1 | The two coloured sources carry the accepted drawings' shapes and fills | The new source-SVG architecture test in `internal/archtest`, beside `icon_test.go`, comparing `build/icon.svg` and `client/icon/icon.svg` with the evidence drawings shape by shape |
| 2 | `make app` on a clean checkout with no librsvg; the server's pin agrees | `TestAppIconIsCommittedAndBuildable` (unchanged, `make -n app` reaches no `mkicon`) and `TestAppIconMatchesTheCommittedArt`'s server row |
| 3 | The chat client's bundle carries the committed `.icns` and its recorded hash equals `client/icon/icon.svg` | The new `client/icon/AppIcon.icns.source-sha256` and its row in `TestAppIconMatchesTheCommittedArt`; `client/build.sh`'s fallback replaced by a refusal |
| 4 | The menu-bar image is a template image and carries no brand colour | The new template-image architecture test: `SetTemplateIcon` at every call site in `cmd/gropius/menubar.go`, and `cmd/gropius/icon.png` decoding to black-plus-alpha with at least two non-zero alpha values |
| 5 | Hero mark and wordmark dot carry the three forms in the cube's arrangement, with the three fills and no grey | `TestMarkCarriesTheThreeForms` in `internal/sitetest`, replacing `TestMarkMatchesTheAppIcon`; `TestControlPanelMarkMatchesTheAppIcon` holds the panel's two marks to the same file |
| 6 | No user-facing line describes a four-form tile, a grey square or a bar-chart chat icon; the README shows the mark under `build/` | `abcd docs lint` with the documentation architecture tests in `internal/archtest`; the README references `build/icon.svg` itself, so there is no second drawing to check |
| 7 | All three renderings read as their drawings on a real Mac | The named hand check `hand-check: mark at 32 px and 18 pt, light and dark` — the server tile and the chat tile at 32 px in the Dock and in Finder, the menu-bar glyph at 18 pt, in both appearances — dated and recorded beside the drawings in the evidence folder, since no automated test sees what a menu bar renders |

## Verification

`make test` green, `gofmt -l .` empty, `go vet ./...` clean, `abcd docs lint`
clean. Every test above watched red before the change: the source-SVG test
against today's four-form art, the two new pins against absent files, the
template test against a PNG carrying colour, and the replaced site test
against the four-quadrant mark.

The rasters are regenerated once, by hand, on a Mac with librsvg: `make icon`,
then `make test`, which is where a stale pin shows. `make app` builds the
bundle and the hand check follows — the bundle's tile in Finder and in the
Dock at 32 px, the running app's menu-bar glyph at 18 pt, the chat client's
tile beside the server's, each in light and dark appearance. Its result is
recorded beside the drawings and in the shipping decision line.

## What is deliberately not built

- No rename, and no file or bundle name anticipating one.
- No answer-style control: the forms are minted here, the control is
  spc-2609200945518522's.
- No second menu-bar raster, no new rasterising dependency, and no build-time
  rasterisation: `make icon` stays by-hand work.
- No compatibility rendering of the four-form tile anywhere, and no migration
  of anything that referenced it.

## Falsifiers, from the intent's Mechanism claim

The cut forms reading as noise rather than three named shapes at 32 px; the
bubble outline breaking up at the Dock's smallest size; the menu-bar
silhouette reading as a filled blob or losing its lifted top face once macOS
recolours it for a dark bar; or any surface needing a hand-tuned second
drawing to be legible, which would mean there is no longer one source. Any of
these reopens the intent rather than being patched on the surface that shows
it.
