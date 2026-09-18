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
---

The sidebar card architecture test scans the whole file for six substrings scoped to no block, leaving the card's icon and title unasserted and matching the creation date against the model's own field regardless of what the card draws.
