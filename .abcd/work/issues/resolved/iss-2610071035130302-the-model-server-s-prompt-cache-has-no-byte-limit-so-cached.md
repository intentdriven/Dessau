---
schema_version: 1
id: "iss-2610071035130302"
slug: "the-model-server-s-prompt-cache-has-no-byte-limit-so-cached"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "SOTA research on MLX serving performance, 2026-10-07"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/runtime/serve.py"
remedy: "Maintainer's decision 2026-10-07: bound the cache within the existing charge. Pass max_bytes to LRUPromptCache in serve.py equal to one served window's KV for one sequence (the uninflated per-token KV size times the served window), passed by the launcher, and pass --prompt-cache-size explicitly; the charge is not raised. Test that the bound reaches the constructor; verify peak memory on a Mac with footprint sampling."
resolution: "Bounded the model server's prompt cache to one served window of real KV (kv_bytes_per_token, stored beside the charge) via --dessau-prompt-cache-bytes and an explicit --prompt-cache-size 10; serve.py passes max_bytes and refuses a bad bound; a model with no real figure keeps no prompt cache. A real-Mac footprint check (launcher footprint sampling) is still owed."
impact: fix
---

The model server's prompt cache has no byte limit, so cached prompts grow outside the memory budget. internal/runtime/serve.py builds server.LRUPromptCache(cli.prompt_cache_size) with the entry count only, so mlx-lm 0.32.0's default of 10 entries and 2^63 bytes applies; upstream's --prompt-cache-bytes flag never reaches the LRUPromptCache constructor (ml-explore/mlx-lm PR 1392, open, which reports Metal running out of memory on long-running servers). docs/memory-budget-explained.md already says caches stack across requests. Each cached entry can hold a whole served window's KV, so a model serving several long conversations can hold several windows' worth of cache that the charge (one window per sequence) does not count.

Correction (2026-10-07, on fixing): the flag is not entirely unused, as first written. In 0.32.0 it drives a trim on the batched path (server.py around line 761: `self.prompt_cache.trim_to(n_bytes=total - active)` after a request joins the batch), but it is never passed to the constructor, so the cache's own max_bytes stays 2^63 and nothing bounds it on the other paths.
