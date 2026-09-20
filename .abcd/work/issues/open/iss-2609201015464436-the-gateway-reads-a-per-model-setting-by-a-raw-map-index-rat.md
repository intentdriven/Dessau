---
schema_version: 1
id: "iss-2609201015464436"
slug: "the-gateway-reads-a-per-model-setting-by-a-raw-map-index-rat"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "spec spc-2609201007367486 design, 2026-09-20"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gateway/gateway.go"
---

The gateway reads a per-model setting by a raw map index rather than the folded reader: cfg.Models[model].MergeSystemMessages is indexed on the request's model string in internal/gateway/gateway.go and ask.go, while the registry, the pool and the models listing all key on config.FoldRepoID, so a request whose spelling differs from the stored key by case or by a trailing slash misses the setting and takes the default. Found while writing the no-transcript spec, whose folded reader must not inherit the gap.
