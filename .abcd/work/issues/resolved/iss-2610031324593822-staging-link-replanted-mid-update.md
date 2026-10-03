---
schema_version: 1
id: "iss-2610031324593822"
slug: "staging-link-replanted-mid-update"
severity: "minor"
category: "security"
source: "impl-review"
found_during: "adversarial re-review of spc-2610030929021692 step 3"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/app/staged.go"
remedy: "Repeat the Lstat check of the staging components immediately before each RemoveAll and Rename, or open a root at the staging org folder itself once it is verified and do every staging operation inside that."
resolution: "Removals under the staging folder go through a root opened at its org folder; swap renames re-check the name first."
impact: fix
resolved_by:
  commit: "517517b965b4a6c4797e1ca6b4c0b1c7006cb2bd"
---

A link re-planted in the staging folder during an update could still reach the served model. os.Root keeps every staging operation inside the models folder but follows a link that stays inside it; realStagingDirs removes such a link before work starts, so only a process of the same account re-planting .dessau+staging/<org> as a link to the model's own org folder between that check and a later removal or rename could reach the served model. Found by the adversarial re-review of spc-2610030929021692 step 3 on 2026-10-03.
