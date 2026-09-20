---
id: itd-2609201407580721
slug: bob-searches-the-web-from-dessau-chat-with-his-own-brave-key
spec_id: spc-2609201459377326
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609151701196720, itd-2609201445423499]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# Bob searches the web from Dessau Chat with his own Brave key: the client runs the search and shows where each answer came from

## Press Release

Bob turns on **Web search** in Dessau Chat's Settings, which is off until
he does, and pastes his own Brave Search API key, where a line tells him
the key is his, stays in his Mac's Keychain, and that Brave bills his
account for what he searches; a "Powered by Brave" mark sits on that
screen. From then on, when he asks a model something it cannot know, a
model the server has recorded as able to call tools can ask for a web
search: Dessau Chat runs the search, hands the results back to the model,
and the answer arrives with the sources it drew on listed beneath it, each
with its title and its host. For a model that never asks, or whenever Bob
wants to be sure, a search-first switch on the composer fetches results for
his question and sends them with it: Brave's snippets by default, the full
pages when he flips a per-question toggle. The on-device model gets the
same only when a second switch, also off by default, says so. Nothing about
this touches the server: the key never leaves Bob's device, the server
relays the conversation as it always has, and Alice's Mac holds no key and
reaches no new host. What Dessau Chat keeps in the conversation is the
answer and the links, never the snippets Brave returned. The iPad does the
same, with the key in its own Keychain.

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

Confirmed by the maintainer at the 2026-09-20 interview: we expect a
per-person key with the client running the loop to be the shape that
survives because it is what every peer client does and what Brave's own
guides sanction, and because it touches no server record. What would show
this wrong: Brave ruling that keys inside third-party apps are not
sanctioned, or the served models never calling the search tool.

## Scope Conditions

Confirmed by the maintainer at the 2026-09-20 interview:

- The chat client only; the key lives in the Keychain, never in a file and <!-- cond: cond-2609201459373651 -->
  never on the server.
- The person is Brave's customer; the maintainer is not a party. <!-- cond: cond-2609201459371730 -->
- Off by default. <!-- cond: cond-2609201459378409 -->
- Whether a model calls tools is read from the server's recorded probe field <!-- cond: cond-2609201459379646 -->
  (itd-2609201445423499); search-first is always available.
- Search-first sends Brave's snippets by default and the full pages on a <!-- cond: cond-2609201459374164 -->
  per-question toggle.
- The on-device model gets search only when a Settings switch, off by <!-- cond: cond-2609201459374478 -->
  default, says so.
- The stored conversation keeps the answer and the links, never the <!-- cond: cond-2609201459378573 -->
  snippets.
- The Brave mark sits in Settings only. The record notes that Brave's terms <!-- cond: cond-2609201459376027 -->
  speak of conspicuous attribution where results are shown, so this is a
  reading a reviewer may challenge.

## Acceptance Criteria

Confirmed by the maintainer at the 2026-09-20 interview, every bullet walked
and accepted:

- Given Settings, when Bob pastes a Brave key, then it is stored in the
  Keychain under the client's own service, never written to a file, and the
  screen says the key is his and that Brave bills his account.
- Given a key is set and search is on, when a served model recorded as
  tool-capable asks for a search through a tool call, then Dessau Chat runs
  it, the results go back to the model as a tool message, and the reply
  lists each source's title and host.
- Given a reply that used search, when Bob relaunches, then the answer and
  the links are still there and no result snippet is stored anywhere.
- Given the search-first switch is on, when Bob sends a question, then the
  client searches first and sends the snippets with the question, or the
  full pages when the per-question toggle is on, so a model that never
  calls tools still answers with sources.
- Given search is off or no key is set, when Bob sends a message, then the
  request carries no tools and nothing about search appears in the reply.
- Given the server, when tool messages and streamed tool-call deltas pass
  through it, then they arrive at the client byte-identical, held by
  passthrough tests beside the existing one for the tools field, and the
  server logs no query text.
- Given the on-device model, when its search switch is off, then it is
  never given search; when on, it gets search-first the same way.
- Given Settings, when web search is on, then the "Powered by Brave" mark
  is shown there.
- Given the iPad, when Bob sets a key there, then it lives in that device's
  Keychain and the same search behaviour holds.

## Open Questions

All four were put to the maintainer at the 2026-09-20 interview and are
decided: tool-call ability is probed once server-side and read by the
client (its own intent, itd-2609201445423499); search-first sends snippets
by default and full pages on a per-question toggle; the on-device model is
behind a switch, off by default; the Brave mark sits in Settings only.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: a local model cannot look anything up, and the client-held key is the one shape the record and Brave's guides both allow; wrong if testers with a key never search, or if Brave changes its terms on third-party apps
