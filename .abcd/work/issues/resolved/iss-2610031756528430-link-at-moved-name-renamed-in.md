---
schema_version: 1
id: "iss-2610031756528430"
slug: "link-at-moved-name-renamed-in"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "adversarial review of the fix for iss-2610031324593822"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/app/staged.go"
remedy: "After each rename, check through the held folders that the moved entry is a directory, not a link, and the same one the name held before; the path the in-place check reads must name it too; undo and abandon on a mismatch."
resolution: "Each swap rename is checked through the held folders to have moved the same directory, not a link; a mismatch is undone and the update abandoned."
impact: fix
resolved_by:
  commit: "530c9f9"
---

A link planted at the staged version's own name (or at the served model's) just before the rename that moves it was renamed in or aside as though it were the folder: renameat moves whatever stands at the one name, a link included. The in-place check then followed the link and passed on another org's model, the update was recorded as landed, and the real old version was removed.
