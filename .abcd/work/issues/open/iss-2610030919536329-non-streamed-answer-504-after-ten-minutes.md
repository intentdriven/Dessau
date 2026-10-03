---
schema_version: 1
id: "iss-2610030919536329"
slug: "non-streamed-answer-504-after-ten-minutes"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "peer-session report while consuming the gateway"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gateway/gateway.go"
remedy: "For a non-streamed request, ask the model server for a stream and assemble the JSON answer in the gateway, so the header wait bounds only admission and a disconnect stops generation; correct the 504 text and document the ceiling meanwhile."
---

A non-streamed chat answer that takes longer than ten minutes ends in a 504 that blames the prompt, and the model server keeps generating for nobody. The gateway's upstream header wait (prefillBudget in internal/gateway/gateway.go: the larger of 10 minutes and a prompt-size estimate, or upstream_header_timeout_sec) is meant to bound prefill, but mlx-lm 0.31.3 sends no headers for a non-streamed request until generation ends (server.py flushes end_headers after the token loop), so for those requests it bounds prefill plus the whole answer. The 504 text says the server 'did not finish reading the prompt' and to 'raise upstream_header_timeout_sec in Settings', which has no Settings control (config.json only); statistics file it as unreachable; the limit is published nowhere and docs/getting-started.md calls it a wait for the first header. After the 504 the gateway closes the upstream connection, but mlx-lm's non-streaming loop never writes before the end, so it decodes to max_tokens, slowing the next request including a retry. Reported 2026-10-03 by a peer session: Nemotron-3.5-Lightning-30B-A3B-4bit, temperature 0, non-streaming, about 2.5k prompt tokens, max_tokens about 53k, HTTP 504 with a 3,600 s client timeout (elapsed time unknown; the live config.json could not be read, so a lowered override is not ruled out). Streaming is unaffected: its headers go out before prefill. Reach widens with the fix for iss-2610030652514762: a non-streaming client that omits max_tokens now generates up to the served window instead of 512 tokens.
