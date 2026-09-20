# The user's directory is theirs

## The rule

> The user's directory is theirs: a tool never creates directories beside the
> user's projects or anywhere the user did not hand over; session and agent
> scratch (a worktree, a verifier's copy, an export) goes under ~/.abcd/ keyed
> on the root-commit SHA, or the checkout's .abcd/.work.local/ tier — see
> .abcd/development/principles/the-users-directory-is-theirs.md.

## Why

`iss-2609081427104462` records what happens when agent scratch lands inside the
tree the repository tests. Two of the four tree-walking architecture tests
scanned worktrees checked out under `.claude/`, so their result depended on
untracked files that are not repository content: a walk excluding only `.git`,
`node_modules`, `dist`, `bin` and `site` saw twelve shell scripts, eight of them
inside `.claude/worktrees/`, and three copies of `install.sh` where exactly one
is tracked. The record states the two-sided cost precisely — a non-compliant
line written in any agent worktree turns the main checkout's test red for a
file nobody can locate in the repository, and a worktree copy can hold a test
green. One shared walker in `internal/archtest` owns the skip rule and every
full-tree scan goes through it, with `.github` and `.githooks` named back in
because excluding every dot-directory would otherwise drop tracked content.

The separation is also a committed rule of the layout. The
`.abcd/work/DECISIONS.md` line of 2026-07-29 records `.abcd/.work.local/` being
gitignored, and the AGENTS.md conventions section states the three tiers:
`.abcd/development/` is the durable committed record, `.abcd/work/` is
committed shared working state, and `.abcd/.work.local/` is gitignored
per-machine ephemera where runtime artefacts — logs, traces, scratch output —
go, never in tracked directories.

Records drawn on: `iss-2609081427104462`, the `.abcd/work/DECISIONS.md` line of
2026-07-29, and the AGENTS.md three-tier convention.

## How to apply here

- Agent worktrees live under `.claude/worktrees/`, which is gitignored and
  which every full-tree scan skips through the shared walker in
  `internal/archtest`. A new scan uses that walker rather than its own
  `WalkDir`.
- Logs, traces, handover notes and scratch output go under
  `.abcd/.work.local/`; notes about the abcd tooling go to
  `.abcd/.work.local/for_abcd/`. Nothing runtime is written into a tracked
  directory.
- The product side holds the same line: Dessau writes into the account's own
  Application Support folder and the shared cache root it was given, and
  nowhere else.
