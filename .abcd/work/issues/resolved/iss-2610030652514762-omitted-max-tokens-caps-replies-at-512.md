---
schema_version: 1
id: "iss-2610030652514762"
slug: "omitted-max-tokens-caps-replies-at-512"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "peer-session report while consuming the gateway"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/runtime/launcher.go"
resolution: "the launcher passes the served window as --max-tokens when the operator has set no default"
impact: fix
resolved_by:
  commit: "27dac19b"
---

A chat request that omits max_tokens is cut off at 512 completion tokens, not at the model's served context. Observed 2026-10-03 on loopback: POST /v1/chat/completions to mlx-community/NVIDIA-Nemotron-3.5-Lightning-30B-A3B-4bit, temperature 0, no max_tokens, returned finish_reason length, usage.completion_tokens 512 and no message content, because the reasoning model was cut off mid-reasoning; /v1/models lists served_context 57384 for that model. Likely cause, not yet verified against the mlx-lm source: samplingArgs (internal/runtime/launcher.go) passes --max-tokens only when the operator has set a max_tokens default, so the model server falls back to its own built-in default. Expected: a request that leaves the limit to the server gets the model's served context minus the prompt. Impact: reasoning models silently return empty answers to any client that leaves max_tokens unset.

Confirmation from the reporter (same day): the identical request with
`max_tokens` set from the published `served_context` finished normally,
with `finish_reason` stop and 7,886 completion tokens. Only the omitted limit
causes the cut-off.
