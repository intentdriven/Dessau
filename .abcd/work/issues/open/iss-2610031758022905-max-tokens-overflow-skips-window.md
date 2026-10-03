---
schema_version: 1
id: "iss-2610031758022905"
slug: "max-tokens-overflow-skips-window"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "audit of mlx-lm 0.32.0's generation thread for iss-2610031444343397"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gateway/gateway.go"
remedy: "Saturate to MaxInt64 when the number overflows."
---

asTokenCount returns 0 for a max_tokens too large for a float64 (a 401-digit integer fails both Int64 and Float64), so such a request skips the served-window check and the server takes it as an effectively unlimited budget.
