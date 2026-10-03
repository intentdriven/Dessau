---
schema_version: 1
id: "iss-2610031757370958"
slug: "chat-check-rests-on-path-compare"
severity: "nitpick"
category: "security"
source: "agent-finding"
found_during: "adversarial review of the fix for iss-2610031010371709"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gateway/gateway.go"
remedy: "Give the chat route its own handler that passes the fact in; make canChat fail closed."
resolution: "handleChatCompletions passes chat=true; canChat returns false on a failed lookup."
impact: fix
resolved_by:
  commit: "a0264236"
---

The gateway decided a request was a chat request by comparing r.URL.Path with the chat route, separately from the mux pattern that routed it, so a future alias route would have skipped the chat check silently; and canChat answered true when the registry lookup failed (harmless only because the acquire then fails).
