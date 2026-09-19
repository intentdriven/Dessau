# Script-first MVP

## The rule

> Script-first MVP: earn each tool rung; the smallest documented protocol is
> the MVP before automation — see
> .abcd/development/principles/script-first-mvp.md.

## Why

The repository's record states the cost of a tool rung rather than the
principle in the abstract, and refuses the rung on that cost.

The `.abcd/work/DECISIONS.md` line of 2026-09-10 on `itd-2609100519003748`
records what the control panel is and why: plain JavaScript with no build step
and no dependency, tested by lifting its functions into node, loading nothing
from outside the Mac. The two rungs a redesign would climb are priced on the
spot — an asset fetched from a CDN is a new outbound host, which
`adr-2609061503319212` and the repository's own rule refuse, and a build
toolchain is a new dependency needing sign-off — and the question of whether
either may be climbed is filed as a decision to be taken before the draft is
planned, rather than arriving with the code.

The 0.7.0 cut, on the `.abcd/work/DECISIONS.md` line of 2026-09-18, is the
protocol standing in for the automation: the cut is by hand on the 0.6.0
precedent, with `abcd launch` still refusing this repository, and the 0.7.1 cut
of the same date is by hand on the 0.7.0 precedent in turn. The procedure that
carries a manual step is written down where a reader will find it, as
`.abcd/development/procedures/installer-authorisation-panel.md` does.

Records drawn on: the `.abcd/work/DECISIONS.md` lines of 2026-09-10
(`itd-2609100519003748`) and 2026-09-18 (the 0.7.0 and 0.7.1 cuts),
`adr-2609061503319212`.

## How to apply here

- The shipped tooling is shell the reader can follow end to end: `install.sh`,
  `client/build.sh`, `client/build-ipad.sh`, `client/mkicon.sh`, and the
  `Makefile`. A step that a script and a paragraph can carry does not need a
  framework.
- A rung that costs a dependency or an outbound host is a decision taken before
  the work, not a detail of the branch that wants it.
- A manual protocol earns its automation by being run enough times to have a
  precedent. Until then it is a dated procedure under
  `.abcd/development/procedures/` or a dated line in `.abcd/work/DECISIONS.md`.
