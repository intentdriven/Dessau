---
schema_version: 1
id: "iss-2610031018046897"
slug: "control-plane-routes-undocumented"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "a peer session's question about unloading from a script"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gateway/control.go"
remedy: "Add a reference page for the control plane's script-facing routes (load, unload, and what each answers), stating that they are reachable only from this Mac and, under itd-2610031004535845's 'this account only', only from the serving account."
related_intents: [itd-2610031024247803]
resolution: "docs/unload-reference.md documents POST /api/models/unload and /api/models/load for a script on this Mac: who can use them, the body, and what each answers."
impact: internal
resolved_by:
  commit: "23933e3519e06f782b76703111c47a3647258139"
---

The control plane's load and unload routes are undocumented for scripts. POST /api/models/load and POST /api/models/unload (internal/gateway/control.go, body {"model": "<repo id>"}) answer from this Mac without a key, and unload waits, bounded, for an idle job holding the model and answers 409 when the model is serving a request or is not loaded, but no page in docs/ names either route, so a client that batch-tests models and wants the memory back after each round finds only the panel's Unload button. Asked by a peer session on 2026-10-03.
