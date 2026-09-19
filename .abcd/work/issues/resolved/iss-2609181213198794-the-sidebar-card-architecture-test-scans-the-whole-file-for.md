---
schema_version: 1
id: "iss-2609181213198794"
slug: "the-sidebar-card-architecture-test-scans-the-whole-file-for"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "fidelity audits 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/archtest"
resolution: "The sidebar-card test is now scoped to ConversationCard's own body: it asserts the card draws the icon (Image(systemName: icon)), the title (Text(title)) and the summary (Text(summary)), and it matches the date against what the card draws — conversation.createdAt in the day/month/year style — rather than against the model's createdAt field appearing anywhere in the file."
impact: internal
---

The sidebar card architecture test scans the whole file for six substrings scoped to no block, leaving the card's icon and title unasserted and matching the creation date against the model's own field regardless of what the card draws.

## Grounds

- pursued: we expect dropping the card's icon or title, or drawing some other date, to fail the test because the scan reads only the card's block and pins the drawn date's field and style; wrong if the card is split into helper views outside the struct, which the block scan would not see.
