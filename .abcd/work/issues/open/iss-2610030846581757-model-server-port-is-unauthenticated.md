---
schema_version: 1
id: "iss-2610030846581757"
slug: "model-server-port-is-unauthenticated"
severity: "critical"
category: "security"
source: "impl-review"
found_during: "adversarial re-review of the fixes for iss-2610030709283687 and iss-2610030752128514"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/runtime/launcher.go"
remedy: "Serve each model child on a Unix socket in a directory only the serving account can open, through a small Dessau-owned launcher, and proxy to it over that socket; no TCP port."
---

Each model server child listens on a loopback TCP port with no authentication, so any account on the serving Mac can bypass the gateway and make it load any directory, running that directory's own Python as the serving account. The launcher starts mlx-lm 0.31.3's server on a free 127.0.0.1 port (internal/runtime/launcher.go launchArgs, freePort); the child answers GET /health and GET /v1/models (which returns the resolved model path) to anyone, and a POST naming another directory in model or draft_model makes it call load() on that path (server.py ModelProvider.load, utils.py load_model, exec_module for config.json model_file). Its --allowed-origins defaults to *. Found 2026-10-03 by the adversarial re-review of the fixes for iss-2610030709283687 and iss-2610030752128514: the gateway-side refusal of draft_model and adapters holds, but this door stays open, so iss-2610030752128514 stays open until it is closed. The maintainer decided the remedy the same day.
