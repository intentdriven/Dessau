---
schema_version: 1
id: "iss-2610030752128514"
slug: "client-named-draft-model-loads-any-directory"
severity: "critical"
category: "security"
source: "impl-review"
found_during: "security review of the fix for iss-2610030709283687"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gateway/gateway.go"
resolution: "both doors closed: the gateway refuses draft_model and adapters (fix/refuse-model-code), and the model server is reachable only by the serving account over its private socket, whose launcher also forces the model and refuses both fields"
impact: fix
resolved_by:
  commit: "a12cf826"
---

A chat request can make the model server load any model directory on disk, outside every check Dessau makes, including running that directory's own Python. The gateway rewrites only the request's model field and relays the rest of the body (internal/gateway/gateway.go handleCompletions and internal/gateway/ask.go), so draft_model and adapters reach mlx-lm 0.31.3 unread. Its server takes draft_model with no validation (server.py: body.get('draft_model'), absent from its _validate list) and calls load(draft_model_path), which reaches load_model and exec_module for any config.json model_file; adapters loads adapter weights from a client-named path and forces a reload of the main model, which re-reads config.json. Verified 2026-10-03 by reading the gateway and the mlx-lm 0.31.3 wheel; the adversarial reviewer also ran ModelProvider._load on a directory naming model_file and saw its .py run. Reach: withAuth admits a loopback connection with a loopback Host without the API key even when one is set, so any account on the serving Mac can point draft_model at a directory it controls and run code as the serving account; from the LAN, a client with the key (or anyone on a keyless install) can name any directory already on disk. A draft model also loads outside the pool's memory budget. Defeats the model-code refusal of iss-2610030709283687, which checks only the model Dessau launches. Surfaced by the adversarial security review of that fix.
