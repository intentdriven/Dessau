---
schema_version: 1
id: "iss-2610042036094209"
slug: "a-request-body-carrying-invalid-utf-8-inside-a-json-string"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "adversarial security review of fix/refuse-non-string-prompt, 2026-10-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gateway/gateway.go"
remedy: "Refuse a request body that is not valid UTF-8 with a 400 at the gateway (utf8.Valid on the bytes it already holds), confirming under the prompt-reading ADRs that a validity check reads no content."
resolution: "A body that is not valid UTF-8 is refused with a 400 at the gateway before it is parsed or forwarded; the check reads the body's encoding, names no field and keeps nothing."
impact: fix
---

A request body carrying invalid UTF-8 inside a JSON string reaches the model server and fails on its handler thread with a 502. Go's json.Unmarshal into RawMessage does not validate UTF-8 inside strings, and json.Marshal forwards those bytes unchanged. mlx-lm 0.32.0's request handler then calls raw_body.decode(), which raises UnicodeDecodeError, and its except clause catches only json.JSONDecodeError. The handler thread errors, the connection closes, and the client gets a 502 rather than a clear 400. The generation thread is not affected, so other clients are unharmed. Found by the security review of the non-string prompt refusal; predates it.
