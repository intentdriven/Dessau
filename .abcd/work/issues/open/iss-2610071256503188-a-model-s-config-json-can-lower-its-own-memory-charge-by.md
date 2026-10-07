---
schema_version: 1
id: "iss-2610071256503188"
slug: "a-model-s-config-json-can-lower-its-own-memory-charge-by"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "security review of the prompt-cache bound (iss-2610071035130302), 2026-10-07"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/registry/registry.go"
remedy: "Decide the latent cache type from more than one config key (the architecture's model_type among the latent families mlx-lm implements, with the rank and rope dimensions present and plausible), and charge the key-value figure whenever the evidence disagrees, with a registry test for a config that claims latent attention falsely."
---

A model's config.json can lower its own memory charge by claiming a compressed-latent cache it does not have. The registry reads the cache type from the config (a kv_lora_rank key marks latent attention) and charges a latent model's smaller per-token figure, so an ordinary key-value model whose config adds that key is under-charged against the memory budget, and the pool may load it beside models it does not fit with. The prompt-cache bound is not made worse by it, since mlx-lm measures cache entries by their real size. Found by the security review of the prompt-cache bound; it predates that change.
