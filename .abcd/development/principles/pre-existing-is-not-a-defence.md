# Pre-existing is not a defence

## The rule

> Pre-existing is not a defence: a defect confirmed while doing other work is
> fixed, or deferred OUT LOUD as a recorded decision naming the finding and the
> reason — never filed and stepped over, and never carried past a release about
> that class of defect. The release cut refuses on a major/critical record
> captured since the anchor tag and still open; the recorded deferral
> (`deferred_after` + `deferral_reason` on the record) is the only way past —
> see .abcd/development/principles/pre-existing-is-not-a-defence.md.

## Why

Both of the repository's most recent release cuts exercised the mechanism, and
the record shows it working rather than being argued for.

The `.abcd/work/DECISIONS.md` line of 2026-09-18 on the 0.7.0 cut opens by
naming what the cut carries past it: `iss-2609161712555136`, a queued "Measure
now" whose model fails the fit check and is dropped silently. The reason is
stated — the defect is in the self-test runner, which that release does not
touch — the record carries `deferred_after: v0.7.0` and its reason in the
frontmatter, the fix is named as the next item of the probe lane, and the line
ends with its own falsifier: the drop biting a 0.7.0 user before the next cut.
The record's resolution later states that the deferral recorded for v0.7.0 is
discharged, so the deferral is a debt with a settlement, not an exemption.

The line of 2026-09-18 on the 0.7.1 cut applies the same rule to a whole class:
any major observation from the fidelity audits still open carries
`deferred_after: v0.7.1` with its reason.

Records drawn on: the `.abcd/work/DECISIONS.md` lines of 2026-09-18 (the 0.7.0
and 0.7.1 cuts), `iss-2609161712555136`.

## How to apply here

- A defect found while doing other work is captured before it is fixed, with
  its category, severity, what you were doing, and the file it lives in.
- A deferral is a dated heading appended to the record's body saying why it
  waits and what decision it needs. A major additionally gets `deferred_after`
  and `deferral_reason` in the frontmatter, immediately after `found_at`.
- The deferral is named in the release's own dated line, with the falsifier
  that would show it wrong, and is discharged in the record when the fix lands.
- "It was already broken" is not a reason to leave it; it is a reason to say so
  in writing.
