# Evaluator outside the loop

## The rule

> Evaluator outside the loop: the reviewer/challenger is independent of whoever
> proposed the option — see
> .abcd/development/principles/evaluator-outside-the-loop.md.

## Why

The phrase appears in the record under its own name.
`.abcd/development/research/notes/2026-09-06-local-telemetry-sota.md` states
that it is produced by an independent research pass — "evaluator outside the
loop" — and that the maintainer's decision is the gate.

The rest of the record is the practice rather than the phrase. The
`.abcd/work/DECISIONS.md` line of 2026-09-06 on the GitHub issue triage records
sixteen maintainer-filed issues "each adversarially verified by an independent
reviewer before capture". The lines of 2026-09-07 on the local statistics store
carry corrections "from a second pair of independent reviews of the branch
(design and adversarial security)", written as superseding lines rather than as
edits to what the proposer wrote. The installer-gate lines of 2026-09-08 are
three successive rounds, each closing every route the previous review named and
each followed by a review finding more — which is the argument for the rule
stated as a measurement: the proposer's own confidence was wrong three times
running, and only an outside reader found it out.

The repository's conventions arm the rule where it matters most. A change under
`internal/gateway`, `internal/hub`, `internal/runtime`, `internal/config` or
`internal/capability` needs an adversarial security review before it lands, and
the 2026-09-18 line on the pairing resolve records a security review's blocking
finding reversing the reasoning of the line above it.

Records drawn on: `.abcd/development/research/notes/2026-09-06-local-telemetry-sota.md`,
the `.abcd/work/DECISIONS.md` lines of 2026-09-06 (triage), 2026-09-07 (local
statistics), 2026-09-08 (installer gate) and 2026-09-18 (`iss-2609181052437306`).

## How to apply here

- A diff touching a trust boundary gets a hostile read that is a separate pass
  from the one that wrote it, and what that pass finds is captured before it is
  fixed.
- A review's corrections are appended to `.abcd/work/DECISIONS.md` as a new
  dated line that governs where it differs; the superseded line stands as
  written. The ledger is append-only, so the disagreement stays legible.
- A research pass that feeds a decision says in the note that it is
  independent of the decision it feeds.
