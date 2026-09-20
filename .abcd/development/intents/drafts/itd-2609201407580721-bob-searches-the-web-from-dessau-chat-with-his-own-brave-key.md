---
id: itd-2609201407580721
slug: bob-searches-the-web-from-dessau-chat-with-his-own-brave-key
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609151701196720]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# Bob searches the web from Dessau Chat with his own Brave key: the client runs the search and shows where each answer came from

## Press Release

Bob pastes his own Brave Search API key into Dessau Chat's Settings, where
a line tells him the key is his, stays in his Mac's Keychain, and that Brave
bills his account for what he searches. From then on, when he asks the model
something it cannot know, the model can ask for a web search: Dessau Chat
runs the search, hands the results back to the model, and the answer arrives
with the sources it drew on listed beneath it, each with its title and its
host, and a "Powered by Brave" mark beside them. When a model on the server
does not ask for searches on its own, Bob can ask for one himself: a
search-first switch on the composer fetches results for his question and
sends them with it. Nothing about this touches the server: the key never
leaves Bob's device, the server relays the conversation as it always has,
and Alice's Mac holds no key and reaches no new host. What Dessau Chat keeps
in the conversation is the answer and the links, never the snippets Brave
returned. The iPad does the same, with the key in its own Keychain.

## Why This Matters

A local model knows nothing after its training date and cannot look
anything up. Every peer chat client that offers search puts the key on the
person's own machine and runs the loop there, and Brave's own guides put an
individual's key into a third-party desktop app in exactly that shape. The
alternative, a key on the server, was weighed and killed on 2026-09-10:
Brave's terms would land on the operator, queries would be retained by a
third party on the operator's account, and the gateway would have to read
prompt content and reach a new host, which two ratified decisions forbid.
The research note of 2026-09-20 (`research/notes/2026-09-20-brave-search-key-custody-sota.md`)
ranks the shapes and this is the first.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

Seeded by the filing session, unconfirmed; every bullet is a proposal for the
planning interview:

- Given Settings, when Bob pastes a Brave key, then it is stored in the
  Keychain under the client's own service, never written to a file, and the
  screen says the key is his and that Brave bills his account.
- Given a key is set and a served model asks for a search through a tool
  call, when Dessau Chat runs it, then the results go back to the model as a
  tool message and the reply lists each source's title and host with the
  Brave mark beside them.
- Given a reply that used search, when Bob relaunches, then the answer and
  the links are still there and no result snippet is stored anywhere.
- Given the search-first switch is on, when Bob sends a question, then the
  client searches first and sends the results with the question, so a model
  that never emits tool calls still answers with sources.
- Given no key is set, when Bob sends a message, then the request carries no
  tools and nothing about search appears in the reply.
- Given the server, when tool messages and streamed tool-call deltas pass
  through it, then they arrive at the client byte-identical, held by
  passthrough tests beside the existing one for the tools field, and the
  server logs no query text.
- Given the iPad, when Bob sets a key there, then it lives in that device's
  Keychain and the same search behaviour holds.

## Open Questions

- Which served models emit tool calls reliably on the pinned mlx-lm, and
  how the client tells Bob when a model will only work with search-first.
- What "search first" sends: the top results' snippets as context, or the
  full pages fetched by the client.
- Whether the built-in on-device model gets search too.
- The exact attribution the terms require and where it sits on the iPad.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
