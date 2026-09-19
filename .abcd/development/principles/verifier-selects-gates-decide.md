# Verifier selects, gates decide

## The rule

> Verifier selects, gates decide: a verdict is a proposal; the human's adoption
> is the gate — see
> .abcd/development/principles/verifier-selects-gates-decide.md.

## Why

`.abcd/development/research/notes/2026-09-06-local-telemetry-sota.md` states
the second half of the rule in as many words: the research pass is independent,
and "the maintainer's decision is the gate".

The `.abcd/work/DECISIONS.md` line of 2026-09-07 on `itd-2609061441241254` is
the rule applied to an audit's verdict. Acceptance criterion 5 is adopted as
diverged rather than met, on the design review's finding, and the line says
what the audit is for: the fidelity audit "should report criterion 5 as
diverged for the maintainer to adopt or reject rather than as satisfied". The
verifier's job ends at the verdict; the adoption is a separate, recorded act.

The line of 2026-09-10 shows the same separation on a decomposition: the
decomposition is a proposal, the maintainer adopts it as SPLIT, and the
question the split opens is filed as a decision to be taken before the draft is
planned rather than settled by the draft itself. The line of 2026-09-08 on the
VPN vendor turns on the same distinction — the classifier cannot know what a
person writing the page knows, so the maintainer's decision on
`iss-2609081221484689` is what admits the exception, through a line-scoped allow
marker rather than a blanket exemption.

Records drawn on: `.abcd/development/research/notes/2026-09-06-local-telemetry-sota.md`,
the `.abcd/work/DECISIONS.md` lines of 2026-09-07 (`itd-2609061441241254`),
2026-09-08 (`iss-2609081221484689`) and 2026-09-10 (`itd-2609100519003748`).

## How to apply here

- An audit, a review or a lint run reports; it does not amend the record it
  judges. A criterion that is not met is reported as diverged for the
  maintainer to adopt or reject.
- An adoption is written down where a reader will open it — the intent's Audit
  Notes, the spec's as-built section, and a dated line in
  `.abcd/work/DECISIONS.md`.
- Gates the repository actually holds: the maintainer's sign-off on a new
  dependency, the pre-commit identity pin and name guard in `.githooks/`, and
  the release gate that runs `install.sh` against the artefacts it has just
  built. Each refuses; none of them decides on the maintainer's behalf.
