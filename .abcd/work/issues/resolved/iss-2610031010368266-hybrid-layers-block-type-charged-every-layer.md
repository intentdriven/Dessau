---
schema_version: 1
id: "iss-2610031010368266"
slug: "hybrid-layers-block-type-charged-every-layer"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "investigation of a peer report about the memory line"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/registry/registry.go"
remedy: "Count the attention entries of layers_block_type as the cache-bearing layers, beside the three spellings already read, and confirm the served window with the context probe on a real Nemotron load."
resolution: "The KV-layer reader now counts the attention entries of layers_block_type, read after hybrid_override_pattern as the pinned mlx-lm does (nemotron_h.py gives a KVCache to attention entries only; mamba gets a fixed-size ArraysCache, moe and mlp none). Nemotron-3.5-Lightning is charged 30,720 bytes a token (6 of 52 layers) instead of 266,240; at the 76.8 GiB default budget of a 128 GB Mac it is served at its declared 262,144 tokens at one or four batched requests. The corrected charge is 2.5 times the 12 KB a token the 2026-09-06 campaign measured, inside the safety factor; the Mamba layers' fixed state (about 2 MB a layer a sequence) is covered by the weights' headroom. The context-probe confirmation on a real load remains for the maintainer."
impact: fix
resolved_by:
  commit: "ca629a54"
---

A hybrid model whose layout is spelled layers_block_type is charged cache memory for every layer, not only its attention layers. Dessau sizes the per-token KV charge from config.json and recognises a hybrid layout only through layer_types, hybrid_override_pattern or full_attention_interval (the KV-layer reader in internal/registry/registry.go), so mlx-community/NVIDIA-Nemotron-3.5-Lightning-30B-A3B-4bit (52 layers, layers_block_type: 23 mamba, 23 moe, 6 attention; 2 KV heads, head dim 128) is charged 52 x 2 x 128 x 2 x 2 x 5 = 266,240 bytes per token instead of 30,720. With 17,792,502,843 bytes on disk and 4 decode sequences, the derived served window is exactly 57,384, the figure the models list publishes, and its charge fills the whole 76.8 GiB budget, so nothing else can load beside it, though the 2026-09-06 measurements put the real cost near 12 KB per token. The pinned mlx-lm reads layers_block_type itself (nemotron_h.py). The 2026-09-20 decision line already noted the model is charged for every layer and was worth a measurement-backed revisit; no record carried it. Found 2026-10-03 investigating a peer report that the panel read '76.8 GB of 76.8 GB budget resident' with only Nemotron loaded.
