---
schema_version: 1
id: "iss-2610031756537487"
slug: "missing-folder-put-back-unchecked"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "adversarial review of the fix for iss-2610031324593822"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/app/staged.go"
remedy: "Lstat the aside entry before the rename and check the moved entry afterwards, as the swap's other renames are; undo on a mismatch."
resolution: "The missing-folder put-back is checked like the swap's other renames."
impact: fix
resolved_by:
  commit: "b650136"
---

When the model's folder was missing, the swap put the newest aside copy back with a rename nothing checked afterwards, so a link planted at that aside name between the listing and the rename was moved in as the model's folder.
