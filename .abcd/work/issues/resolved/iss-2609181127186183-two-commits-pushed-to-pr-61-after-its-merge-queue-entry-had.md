---
schema_version: 1
id: "iss-2609181127186183"
slug: "two-commits-pushed-to-pr-61-after-its-merge-queue-entry-had"
severity: "major"
category: "observation"
source: "user-observation"
found_during: "landing PR 61, 2026-09-18"
origin: researcher-authored
production_mode: hand-written
resolution: "The dropped commits were re-landed by PR 66; the rule — never push to a queued PR after arming auto-merge, and verify the merge parent before deleting a branch — is in AGENTS.md's Git conventions and in the maintainer's agent memory."
impact: internal
---

Two commits pushed to PR 61 after its merge-queue entry had been enqueued (the sidebar cards, the search bar, the thoughts rendering, three shipped intents' records) were dropped silently: the queue merged the head it had enqueued, GitHub marked the PR merged with the later head, and the branch was deleted as merged. Rule for this repository: once auto-merge is armed on a queued PR, never push to its branch again; open a new PR. Also worth a check in the landing routine: compare the PR's headRefOid with the merge commit's parent before deleting the branch.

## Grounds

- pursued: we expect a written rule and a head-landed check to prevent a silent drop from recurring; wrong if the queue drops a head that was never pushed after arming
