---
id: adr-2610040749545010
slug: the-gateway-may-refuse-a-request-whose-prompt-or-messages
status: accepted; superseded in part by adr-2610042021365934 (a /v1/completions prompt that is not a JSON string is refused too, reading its kind)
date: 2026-10-04
supersedes: [adr-2609201008470380, adr-2609061610102325]
superseded_by: adr-2610042021365934
related_intents: []
related_rfcs: []
related_adrs: [adr-2609201008470380, adr-2609061610102325]
---

# ADR-2610040749545010: The gateway may refuse a request whose prompt or messages are empty, reading emptiness and nothing else

**Supersedes in part:** [adr-2609201008470380](2609201008470380-the-gateway-may-retain-both-sides-of-a-conversation-in-a-tra.md),
on its closing rule that no reader of prompt content exists beyond those the
readers' list admits. That rule is restated there from
[adr-2609061610102325](2609061610102325-the-gateway-may-rewrite-prompt-content-only-to-merge-system.md),
which is why both records carry a status note. Every other decision of both
records stands.

## Context

adr-2609201008470380 admits a short list of readers of a client's prompt
content — the system-message merge, the Discord bridge and the transcript
retainer — and says any further reading by the gateway is a new decision that
supersedes it. This is that decision, and it is the narrowest one the boundary
can take.

A static audit of mlx-lm 0.32.0 found that a `/v1/completions` prompt that
tokenizes to nothing reaches `insert_segments` on the model server's one
generation thread and raises there (iss-2610031758029994). The audit was
medium-confidence and tokenizer-dependent, so on 2026-10-03 the maintainer
decided to reproduce it on the Mac first, and settled in advance what the
gateway would get if it reproduced.

It reproduced on 2026-10-03, on Dessau v0.9.3 with mlx-lm 0.32.0 and a 30B
mixture-of-experts 4-bit model, and the result was worse than the audit
predicted. One request with `"prompt": ""` got no answer. Every request to
that model after it, including an ordinary one, hung until the client gave up
at 120 seconds. Throughout, the model server's process stayed alive, and Dessau
reported the model loaded with nothing in flight. Only an unload recovered it.
Any client that can reach the server can send that request.

The gateway cannot see what the prompt tokenizes to, and nothing short of
reading the prompt can tell it that a request has none.

## Decision

We will refuse, with a 400 and before a model is resolved or loaded:

- a `/v1/completions` request whose `prompt` is a string that is empty or only
  whitespace, or an empty array;
- a `/v1/chat/completions` request whose `messages` is an empty array;
- a bridged request (the gateway's `Ask` path) whose `messages` is an empty
  array, for parity with the network path.

The check reads whether the field is empty and nothing else: never what a
message says, never how many messages there are, never any other field of a
message. It keeps nothing: no log line, statistics record, transcript entry or
error message carries any of the prompt, and the refusal names the field, not
its value. A prompt or messages value of another type, and a missing field,
are left to the model server, whose checks for them run on its handler thread.

The check lives in one file, `internal/gateway/emptyprompt.go`. It is entered
by name on the prompt-content readers' list in
`internal/archtest/prompt_content_test.go`. The scan learns the `prompt` field
in the same change, so a later reader of a completions prompt is caught as a
reader of messages already is.

We will treat any further reading of prompt content by the gateway as a new
decision that supersedes this record, as the records before it said.

## Alternatives Considered

1. **Refuse emptiness at the gateway (chosen).** It closes a reproduced freeze
   that any client on the network can cause. The grant is one bit per request,
   held in one file and named on the readers' list.
2. **Leave it to the pool's health watch.** The watch on main restarts a
   model server whose `/health` answers 503. The reproduction shows a server
   that stayed up and reported nothing wrong. The reproduction ran on
   v0.9.3, which predates the watch, and whether `/health` answered 503 was
   not measured, so the watch is not shown to catch this freeze; the model
   server's source predicts a 503 and an immediate error rather than the hang
   that was seen, and the difference is recorded on iss-2610040752568866.
   Rejected as the only defence; the watch stays as the net under raisers the
   gateway cannot see.
3. **Ask mlx-lm to fix it upstream and wait.** That is worth doing, and does
   not depend on this record. Rejected as the only remedy: the freeze is live
   on the pinned version and needs nothing but one request to trigger.
4. **Tokenize the prompt at the gateway to see whether it is empty.** That
   reads far more than emptiness and needs the model's tokenizer in Go.
   Rejected as a much wider grant than the defect calls for.

## Consequences

- adr-2609201008470380 changes only its status fields, to say it is superseded
  in part by this record. adr-2609061610102325, whose rule it restates, gets a
  status note too. Neither record's text is edited.
- A client that sends an empty prompt gets a 400 naming the field, where before
  it froze the model for everyone. That is a breaking change on the wire for
  such a request, recorded as `impact: breaking`.
- Whitespace-only prompts and empty `messages` are refused, though the
  reproduction could not show whether they freeze the server alone: the server
  was already frozen when they were sent. They are refused on the maintainer's
  decision of 2026-10-03, which named them, and because a prompt of only
  whitespace is not a request anyone means to send.
- A model server that freezes for some other reason still goes unnoticed: the
  process is up and nothing marks the model unhealthy. That is a separate
  defect, captured on its own, and this record does not address it.
- Obligations: the adversarial security review of the change that lands this,
  the gateway being a trust boundary; a test that each empty shape is refused
  before the pool or the model server is reached, and that a request with
  anything in it is not; and the readers' list entry naming this file and why.
