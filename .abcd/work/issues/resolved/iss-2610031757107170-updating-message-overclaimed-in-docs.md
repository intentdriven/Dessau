---
schema_version: 1
id: "iss-2610031757107170"
slug: "updating-message-overclaimed-in-docs"
severity: "nitpick"
category: "documentation"
source: "agent-finding"
found_during: "adversarial review of the fix for iss-2610031317470004"
origin: researcher-authored
production_mode: hand-written
found_at: "docs/model-updates.md"
remedy: "Say a 503, and that a client told why reads the updating message; log the refusal class 'updating'."
resolution: "The docs say a 503 and who reads the reason; the log names the refusal 'updating' (edf79db)."
impact: fix
resolved_by:
  commit: "9bede54"
---

docs/model-updates.md said a request refused while a model drains is answered with the message that it is being updated; an unentitled client gets the generic refusal, and only a client told why reads that text. The operator log also read the refusal as plain 'refused'. Never on main.
