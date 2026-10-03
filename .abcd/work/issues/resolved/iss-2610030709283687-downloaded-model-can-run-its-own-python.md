---
schema_version: 1
id: "iss-2610030709283687"
slug: "downloaded-model-can-run-its-own-python"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "adversarial design review of itd-2610030656210408"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/hub/hub.go"
resolution: "a model whose config.json names model_file is refused in Go before the model server starts"
impact: fix
resolved_by:
  commit: "ffca3dc2"
---

Loading a downloaded model can execute Python shipped inside it. The pinned mlx-lm 0.31.3 imports and runs the file a model's config.json names in model_file (mlx_lm/utils.py lines 325-331, spec_from_file_location then exec_module, inside the model server child at load time), and the hub's download filter (wantedFile in internal/hub/hub.go) does not exclude .py files, so a repository that ships a .py file and names it in model_file gets its code run under the operator's account the first time the model loads, with the runtime's full file access. Verified 2026-10-03 by reading the mlx-lm 0.31.3 wheel; mlx-vlm 0.7.4 has the same path. Reach: downloads are started only from the loopback control plane (Control.Handler wraps loopbackOnly), and HF_HUB_OFFLINE plus the model-field rewrite stop a chat client from fetching a repo by name, so the trigger is the operator downloading a hostile repository; nothing tells them a model carries code. The tokenizer path is not exposed: the server passes trust_remote_code only on its own flag, which the launcher never sets. Surfaced by the adversarial design review of itd-2610030656210408, whose 'never runs Python from a download' promise this also blocks.
