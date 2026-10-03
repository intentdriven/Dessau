---
schema_version: 1
id: "iss-2610031010371709"
slug: "load-offered-for-a-non-chat-model"
severity: "minor"
category: "ux"
source: "agent-finding"
found_during: "investigation of a peer report about the Clef card"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gateway/control.go"
remedy: "Refuse or relabel Load and refuse chat requests for a model the chat rule marks not chat-capable, and add a Load-button criterion to itd-2610030656210408."
resolution: "Chat requests, the bridge and the panel's Load now refuse a model CanChat rejects; /v1/completions still serves every model; a Load criterion is added to itd-2610030656210408."
impact: breaking
resolved_by:
  commit: "e061bd3"
---

The control panel's Load starts a chat model server for a model the chat rule marks not chat-capable. Load appears on every model that is not downloading or failed (internal/ui/static/app.js), and the control plane's load handler asks the pool directly without consulting CanChat (internal/gateway/control.go); a chat request naming such a model is passed through too. For mlx-community/clef-4bit, a decision model shown READY, mlx-lm loads the Qwen3.5 backbone without its vision weights and joint head (its own README says plain mlx-lm 'will load the backbone but produce meaningless text'), so the load likely succeeds, its derived window fills the budget and can evict an idle model, the tool-call probe then sends it a chat request, and chat answers are nonsense, where the GLM-OCR case at least failed loudly. itd-2610030656210408 promises decision models are marked and refused on chat, but none of its criteria names the Load button. Found 2026-10-03 from a peer's observation; inferred from code, not run.
