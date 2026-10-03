---
schema_version: 1
id: "iss-2610031756525719"
slug: "swap-rename-misdirected-to-another-org"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "adversarial review of the fix for iss-2610031324593822"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/app/staged.go"
remedy: "Rename relative to the model's org folder and the staging org folder held open, and check each rename moved the directory it meant to."
resolution: "The swap renames with renameat relative to the held org folders (d146f31); each rename is checked afterwards (530c9f9)."
impact: fix
resolved_by:
  commit: "d146f31"
---

A link planted at the staging org folder's name after its last check, pointing at another org's folder, misdirected the swap's rename: Dessau moved that org's model in as this one, served it, and then removed the real old version as the swap's leftover. Name checks before each rename only narrowed the window; two reviews built interleavings through them.
