---
schema_version: 1
id: "iss-2609190226069369"
slug: "internal-gateway-pairing-go-s-readpairrequest-matches-the-pa"
severity: "nitpick"
category: "bug"
source: "agent-finding"
found_during: "adversarial security review of the pairing audit fixes branch"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gateway/pairing.go"
---

internal/gateway/pairing.go's readPairRequest matches the pairing request's content type with strings.HasPrefix(ct, "application/json"), which accepts 'application/jsonevil' and refuses 'APPLICATION/JSON' although RFC 9110 makes the media type case-insensitive. Neither is a CSRF gain — no application/json variant is CORS-safelisted, so the preflight the guard relies on still happens and the Origin check beside it still holds — but a client that spells the header in capitals is refused a pairing for no reason. Parse the media type rather than matching its prefix.
