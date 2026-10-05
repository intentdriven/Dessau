---
id: adr-2610042021365934
slug: the-gateway-may-refuse-a-completions-prompt-that-is-not-a
status: accepted
date: 2026-10-04
supersedes: [adr-2610040749545010]
superseded_by: null
related_intents: []
related_rfcs: []
related_adrs: [adr-2610040749545010, adr-2609201008470380, adr-2609061610102325]
---

# ADR-2610042021365934: The gateway may refuse a completions prompt that is not a JSON string, reading its kind and nothing else

**Supersedes in part:**
[adr-2610040749545010](2610040749545010-the-gateway-may-refuse-a-request-whose-prompt-or-messages.md),
on two of its sentences, and for a `/v1/completions` prompt only: that the
check "reads whether the field is empty and nothing else", and that "a prompt
or messages value of another type ... [is] left to the model server". The
check now also reads the prompt's JSON kind. Every other decision of that
record stands: what counts as an empty string prompt, the refusal of empty
`messages` on the network and bridge paths, a missing field left to the model
server, keeping nothing, and the one file the check lives in.

## Context

adr-2610040749545010 granted the gateway one bit about a request's prompt:
whether it is empty. Its adversarial review found that a `/v1/completions`
prompt that is an array of strings reaches the generation thread of mlx-lm
0.32.0 outside its `try` (iss-2610040805353721). The tokenizer treats a list
of strings as a batch, so `["hi"]` tokenizes to `[[72, 73]]` and `[""]` to
`[[]]`, and the nested list raises in the prompt cache's search and in the
batch generator. The emptiness check refused only an empty array and let
`[""]` and `["hi"]` through, because telling them apart from a string reads
more than emptiness.

The maintainer set the condition on 2026-10-03: reproduce it on the Mac first,
and refuse it if it freezes or crashes the server. On 2026-10-04, on Dessau
v0.9.3 with mlx-lm 0.32.0 and a 30B mixture-of-experts 4-bit model, each prompt
on a fresh load:

- `["hi"]` and `[""]` each got no answer, and the ordinary prompt sent after
  each got none either: the model server was frozen, as the empty prompt had
  left it.
- `[72,73]`, an array of token ids, was refused by mlx-lm with a 404 ("text
  input must be of type `str`"), and the ordinary prompt after it was
  answered.

So on the pinned server no prompt that is not a string yields a generation: a
token array is refused upstream, and an array of strings freezes the model for
every client. The health watch on main turns such a freeze into a crash and a
restart (iss-2610040752568866), which still fails every request in flight on
that model.

## Decision

We will refuse, with a 400 and before a model is resolved or loaded, a
`/v1/completions` request whose `prompt` is present and is not a JSON string:
an array of any kind (of strings, of token ids, or empty), a number, an
object, a boolean or `null`. The refusal names the field and its required
kind, `"prompt" must be a string`, and never the value.

The check reads the prompt's JSON kind, from the first byte of its raw value,
and nothing else: it never looks inside an array or an object, never counts
elements, and decodes a value only when it is already a string, to apply the
emptiness test adr-2610040749545010 grants. It keeps nothing. A missing
`prompt` is still left to the model server, whose check for it runs on its
handler thread. The chat path and the bridge's `Ask` path are unchanged: they
read `messages`, whose emptiness is all they read.

The check stays in `internal/gateway/emptyprompt.go`, in the same function as
the emptiness check, and its entry on the prompt-content readers' list in
`internal/archtest/prompt_content_test.go` names the wider reading.

We will treat any further reading of prompt content by the gateway as a new
decision that supersedes this record, as the records before it said.

## Alternatives Considered

1. **Refuse any prompt that is not a JSON string (chosen).** It closes a
   reproduced freeze that any client on the network can cause, and refuses
   nothing that works on the pinned server. The grant grows from one bit to
   the value's JSON kind, read from its first byte.
2. **Refuse only an array whose elements include a string.** It is the
   narrowest refusal of the reproduced freeze, but it reads inside the array,
   element by element, which is more of the prompt than its kind. A token
   array and a number would still reach the model server, which refuses them
   with a 404 that says nothing a client can act on. Rejected as a wider read
   for a smaller result.
3. **Leave it to the health watch.** The watch restarts a frozen model server
   within a tick, but every request in flight on that model fails, and any
   client can repeat the request at will. Rejected as the only defence, as
   adr-2610040749545010 rejected it for the empty prompt.
4. **Accept arrays and fan them out as one request per element, as OpenAI
   does.** That reads every element, writes new requests from client text and
   adds batching the gateway does not otherwise have. Rejected as a feature
   nobody has asked for, and a far wider grant than the defect calls for.

## Consequences

- adr-2610040749545010 changes only its status fields, to say it is
  superseded in part by this record; its text is not edited.
- A client that sends an array, number, object, boolean or `null` prompt gets
  a 400 naming the field. Before, an array of strings froze the model for
  everyone and the rest were refused by the model server with a 404. That is
  a breaking change on the wire, recorded as `impact: breaking`, though no
  such request produced an answer before.
- A client written for OpenAI's batched or tokenized `prompt` forms cannot use
  them through Dessau. It could not before either.
- If a later mlx-lm pin accepts a list prompt and generates from it, this
  refusal would withhold a working feature. Moving the pin is the moment to
  check, and lifting the refusal is a new decision.
- Obligations: the adversarial security review of the change that lands this,
  the gateway being a trust boundary; a test that each non-string kind is
  refused before the pool or the model server is reached, and that a string
  prompt is not; and the readers' list entry updated to name the kind read.
