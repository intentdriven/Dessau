---
schema_version: 1
id: "iss-2610040805353721"
slug: "completions-prompt-array-reaches-the-generation-thread"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "the adversarial review of fix/refuse-empty-prompt"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gateway/emptyprompt.go"
remedy: "Maintainer's decision needed: may the gateway refuse a /v1/completions prompt that is not a JSON string? That reads the field's JSON kind, as badStop already does for stop, and refuses nothing that works on the pinned server, but it widens adr-2610040749545010's grant, so it would be a new ADR. Reproduce on the Mac first if the decision is to follow the earlier REPRODUCE FIRST practice."
---

A /v1/completions prompt that is an array of strings reaches mlx-lm 0.32.0's generation thread outside its try (medium confidence: transformers 5.14.1 verified locally, the mlx-lm path read from source). encode treats a list of strings as a batch, so ["hi"] tokenizes to [[72,73]] and [""] to [[]]; _tokenize returns the nested list, and fetch_nearest_cache, the cache trie's search, insert_segments and batch_generator.next() then run on the generation thread outside the try, where the nested list raises. The empty-prompt check refuses only an empty array and leaves [""] and ["hi"] alone, because looking inside the array, or at the prompt's JSON kind, reads more than the emptiness adr-2610040749545010 grants. On 0.32.0 a list prompt never yields a valid generation.

## Reproduced on the Mac 2026-10-04, and the maintainer's decision

Run on the installed v0.9.3 (mlx-lm 0.32.0) with Nemotron-3.5-Lightning-30B-A3B-4bit,
each prompt on a fresh load, a normal prompt before and after, unloaded after
each:

| prompt | result | the normal prompt after it |
|--------|--------|----------------------------|
| `["hi"]` | no answer, 30 s client timeout | no answer, 30 s timeout: server frozen |
| `[""]` | no answer, 30 s client timeout | no answer, 30 s timeout: server frozen |
| `[72,73]` | refused by mlx-lm, HTTP 404, "text input must be of type `str` …" | HTTP 200: server healthy |

The array-of-strings case freezes the server exactly as the empty prompt
did. On main the health watch would turn that into a crash and a restart
(iss-2610040752568866), still failing every request in flight on that model.

**Decision (maintainer, 2026-10-04, recorded as the condition set on
2026-10-03 is met):** the gateway refuses a `/v1/completions` prompt that is
not a JSON string, with a 400. This reads only the field's JSON kind, as
`badStop` does for `stop`, and refuses nothing that works on the pinned
server: no list prompt yields a valid generation there, and token arrays are
already refused upstream. It widens adr-2610040749545010's grant, so it lands
with a new ADR that partly supersedes it.
