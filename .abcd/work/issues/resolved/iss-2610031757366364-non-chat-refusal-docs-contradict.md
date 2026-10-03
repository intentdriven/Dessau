---
schema_version: 1
id: "iss-2610031757366364"
slug: "non-chat-refusal-docs-contradict"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "adversarial review of the fix for iss-2610031010371709"
origin: researcher-authored
production_mode: hand-written
found_at: "docs/chat-models.md"
remedy: "Reword all three: a chat request is refused, /v1/completions still serves the model."
resolution: "docs/chat-models.md, README.md and the picker string say chat is refused and /v1/completions still serves the model."
impact: fix
resolved_by:
  commit: "a0264236"
---

With chat requests refused for a model the chat rule marks not chat-capable, docs/chat-models.md, README.md and Dessau Chat's picker string still said the rule filters nothing and every model stays callable over the API by name, and docs/chat-models.md did not name the new 400 on chat and 409 on Load. Never on main.
