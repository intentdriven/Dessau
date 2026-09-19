# Memory graduates to the record

## The rule

> Memory graduates to the record: a lesson whose 'why' is a correction any
> agent should receive belongs in the repo's committed record — the AGENTS.md
> conventions section, a custom domain in .abcd/rules.json, or a ledger capture
> — never only in one user's local agent memory, which is for facts about that
> user and machine; a memory item recalled twice is a graduation candidate —
> see .abcd/development/principles/memory-graduates-to-record.md.

## Why

The repository has done the graduation and recorded it. The
`.abcd/work/DECISIONS.md` line of 2026-09-18 on the 0.7.1 cut states that the
queue lesson of PR 61 is graduated into AGENTS.md. The lesson itself is the
conventions bullet that a queued PR is never pushed to again: `main` sits
behind a merge queue, the queue merges the head it enqueued, and a later push
can be dropped without a word. The bullet cites the PR and the date it was
learned on, so a reader can check the claim against the forge rather than
taking it on trust.

That is the whole argument for the rule, made concrete: the lesson cost a
dropped push once, and every agent working in this repository afterwards reads
it from the committed conventions rather than from whichever machine happened
to witness it.

Records drawn on: the `.abcd/work/DECISIONS.md` line of 2026-09-18 (the 0.7.1
cut), and the AGENTS.md conventions bullet it names.

## How to apply here

- A lesson about how this repository works goes to the AGENTS.md conventions
  section. The merge-queue bullet, the release-and-tag bullet and the
  branch-deletion bullet are all graduated lessons, each carrying the evidence
  that taught it.
- A lesson about a defect goes to the ledger as a capture under
  `.abcd/work/issues/`, and a lesson that shapes the architecture goes to an
  ADR under `.abcd/development/decisions/adrs/`.
- A lesson about the abcd tooling itself is the one exception: it goes to the
  gitignored `.abcd/.work.local/for_abcd/` notes, never to this repository's
  ledger.
- Local agent memory keeps facts about one person and one machine. A memory
  item recalled twice is asking to be written down here.
