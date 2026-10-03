---
schema_version: 1
id: "iss-2610031018231170"
slug: "hybrid-pattern-as-a-list-not-read"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "fix for iss-2610031010368266"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/registry/registry.go"
remedy: "Read hybrid_override_pattern as either a string or a list of one-character strings in the KV-layer reader (internal/registry/registry.go), with a test for each shape."
resolution: "The KV-layer reader counts the \"*\" entries of hybrid_override_pattern written as a list as well as a string, before layers_block_type (TestAHybridPatternWrittenAsAListIsReadToo)."
impact: fix
resolved_by:
  commit: "10116dd"
---

Dessau reads hybrid_override_pattern only as a string, but mlx-lm 0.31.3 types it as Optional[List[str]] (nemotron_h.py), so a model whose config.json writes the pattern as a JSON list falls back to charging every layer for KV cache, the same over-charge iss-2610031010368266 fixed for layers_block_type. Found 2026-10-03 while fixing that issue; no such config is known to be on a server yet.
