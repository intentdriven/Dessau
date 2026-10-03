---
schema_version: 1
id: "iss-2610031811496593"
slug: "bridge-skips-generation-refusal"
severity: "nitpick"
category: "security"
source: "agent-finding"
found_during: "adversarial review of the fix for iss-2610031444343397"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gateway/ask.go"
remedy: "Apply generationRefusal in Ask beside the other refusals."
resolution: "Ask applies generationRefusal beside the other refusals, with a test."
impact: fix
resolved_by:
  commit: "7853fbde51368cf570418cb5074ca627fc01b51d"
---

Ask, the bridge's in-process path, applies loadField, emptyAnswerBudget and badStop as the rule held on the second path, but not generationRefusal. Its body is built by Dessau today and carries none of the fields, but Ask takes arbitrary bytes, and parity is what the second path promises.
