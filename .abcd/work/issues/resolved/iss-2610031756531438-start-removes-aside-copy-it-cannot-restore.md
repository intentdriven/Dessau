---
schema_version: 1
id: "iss-2610031756531438"
slug: "start-removes-aside-copy-it-cannot-restore"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "adversarial review of the fix for iss-2610031324593822"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/app/staged.go"
remedy: "Keep an aside copy the start cannot put back unless the model's folder holds a version that checks out, and name it in the log."
resolution: "A start keeps an aside copy it cannot put back while the model's folder does not check out (b650136, refined in fbf4fa6)."
impact: fix
resolved_by:
  commit: "b650136"
---

A start removed the whole staging folder after putting back what it could, including an old version left aside that could not go back because something else stood in its model's folder: a swap whose put-back failed logged that the old version would be put back at the next start, and the start then deleted what may have been the only copy. Predates this branch.
