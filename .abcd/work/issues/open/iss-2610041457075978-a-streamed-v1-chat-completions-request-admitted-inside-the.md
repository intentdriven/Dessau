---
schema_version: 1
id: "iss-2610041457075978"
slug: "a-streamed-v1-chat-completions-request-admitted-inside-the"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "peer session's local brief-to-records run, 2026-10-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/registry/measurement.go"
remedy: "Reproduce on main; if the stall is memory, refuse at admission a request whose projected peak footprint exceeds the memory budget (the projection the probe's memory guard already makes), naming the largest request that fits, instead of admitting it by the served window."
---

A streamed /v1/chat/completions request admitted inside the served window, but above the size the memory guard stopped the context probe at, stalled for 64 minutes and held the model. Qwen3-Coder-Next-4bit on v0.9.3 has served_context 58260 and measured_context 21606, with measured_bound memory_guard. A request with about 29.5k estimated prompt tokens and requested_tokens 48212 was admitted, produced a first token after 44 s, then nothing until the client gave up after 3875 s (class cancelled, 0 completion tokens counted). in_flight stayed 1 throughout, so the model could not be unloaded. The memory-guard figure is a verified floor rather than a ceiling (docs/context-probe.md), so this is evidence that admission by the served window can let through a request whose projected footprint the guard already judged too large. It is not proof that everything above the measured figure fails. Seen by a peer session's scripted run on 2026-10-04; not reproduced on main.

## Second case, 2026-10-04: under the measured window, so not the memory guard alone

From the same peer run, verified against `/api/stats` and `/v1/models`:

- **Ternary-Bonsai-27B-mlx-2bit**, streamed, thinking off. About 29.5k
  estimated prompt tokens, requested_tokens 45065, served_context 55113.
- **Its probe stopped at measured_context 44560 with measured_bound
  `served_window`**, not `memory_guard`. So this prompt sat well inside both
  the served window and the measured one.
- **Same shape as the first case:** first token after 128 s, then nothing
  until the client's 15-minute idle timeout ended it at 1361 s (class
  `cancelled`, 0 completion tokens counted).
- **No other request ran during either stall.** Each was the only request
  in flight, so contention is ruled out.
- **For contrast:** Nemotron-3.5-Lightning completed a 29.5k-token prompt in
  198 s in the same run (first token at 28 s, 17,110 tokens out). The same
  Bonsai served 2–10k-token prompts normally.

So the memory-guard explanation does not cover the Bonsai case. The common
factors are a ~30k-token streamed prompt, a first token, and then silence.
The remedy's first step, reproducing on main, now needs to tell apart three
possibilities: decode stalling at that context in these two models; the
model server's stream stalling after its first chunk; and the gateway's
relay holding back what it receives. The record's remedy may change once
that is known.

## Third case, 2026-10-04: Bonsai wedges on a small prompt and stays wedged

From the peer's chapter-sized re-run (17:06–18:44), verified against `/api/stats`:

- **Qwen3-Coder-Next-4bit:** all 8 chapter requests completed (1.6k–8.6k
  estimated prompt tokens, first token in 2–18 s). For this model the ~30k
  prompt in the first case stays the suspect.
- **Ternary-Bonsai-27B-mlx-2bit stalled on its first chapter:** about 1.8k
  estimated prompt tokens. The first token arrived after 10 s, then nothing
  until the client cancelled at 1180 s.
- **It then answered nothing at all.** Each of the next 7 requests to it
  (17:34–18:34) ended as class `unreachable` after exactly 600 s, with no
  first token, until an unload at 18:44 cleared it.

So, at least for Bonsai, prompt size is not the cause. A streamed request
can leave the model server wedged, and it stays wedged until an operator
unloads it. That is the frozen-server shape of iss-2610040752568866, seen on
v0.9.3, which has no health watch.

Whether main's health watch would catch this depends on what `/health`
answers in this state. The earlier measurement covered only a generation
thread that had died, not one that hangs. That is the first thing to
establish when reproducing. Severity is worth reconsidering: a model that
stops answering until someone notices and unloads it is more than minor.
