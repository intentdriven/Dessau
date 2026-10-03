---
schema_version: 1
id: "iss-2610031756549441"
slug: "start-clear-removes-through-link"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "adversarial review of the fix for iss-2610031324593822"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/app/staged.go"
remedy: "Open and hold each staging org folder (Lstat, open, same-file check) and remove through the held root."
resolution: "The start's clear holds each staging org folder open and removes through it."
impact: fix
resolved_by:
  commit: "fbf4fa6"
---

The start's clear, as first changed on this branch, listed and removed entries of each staging org folder by name through the models root, which follows a link that stays inside it; a link planted at an org folder's name after its Lstat would have made the start delete served models. Never on main.
