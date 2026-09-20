---
schema_version: 1
id: "iss-2609200830185068"
slug: "the-server-advertises-no-stable-identity-of-its-own-in-its-b"
severity: "minor"
category: "future-work-seed"
source: "agent-finding"
found_during: "decomposition of the model-picker rethink, 2026-09-20"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/discovery/discovery.go"
---

The server advertises no stable identity of its own in its Bonjour TXT record, so a client can only tell two advertisements apart by display name, which Bonjour itself rewrites on a conflict ('(2)') and which a restart without deregistering leaves as a stale twin. The TXT record carries txtvers, api, path, auth, models and, when TLS is on, the certificate's spki fingerprint and tlsport; the fingerprint is stable across restarts because the server's key persists (adr-2609182357322050) but is absent on a plain-HTTP server. A stable per-install server id in the TXT record, minted once and kept beside config.json, would let every client deduplicate by identity rather than by name and recognise a paired server under a new name. Server side, internal/discovery; the client-side symptom is iss-2609200822241060 and the picker rethink itd-2609200829199959 lists the deduplication as an open question.
