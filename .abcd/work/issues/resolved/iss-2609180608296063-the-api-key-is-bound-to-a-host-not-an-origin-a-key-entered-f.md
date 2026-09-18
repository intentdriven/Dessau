---
schema_version: 1
id: "iss-2609180608296063"
slug: "the-api-key-is-bound-to-a-host-not-an-origin-a-key-entered-f"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "adversarial review of feat/client-macos-27, 2026-09-17"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/GropiusChat.swift"
resolution: "The key is bound to and compared against the origin: scheme, lowercased host and port."
impact: fix
---

The API key is bound to a host, not an origin: a key entered for an https address is also sent to a discovered http server on the same host. Bind to scheme, host and port.
