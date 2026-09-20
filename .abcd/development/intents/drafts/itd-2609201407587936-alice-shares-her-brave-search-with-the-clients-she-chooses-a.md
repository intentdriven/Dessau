---
id: itd-2609201407587936
slug: alice-shares-her-brave-search-with-the-clients-she-chooses-a
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609182357325215, itd-2609201407580721]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# Alice shares her Brave search with the clients she chooses: an opt-in search sidecar beside Dessau Server, granted per paired client

## Press Release

Alice has a Brave Search API key and wants Bob and Carol, whose clients are
paired with her server, to search the web through it without holding a key
of their own. In the control panel's Clients tab she turns on **Search** for
exactly the paired clients she chooses; the others see nothing new. For a
granted client, search is a small sidecar that runs beside Dessau Server on
its own port, speaking the Model Context Protocol, so any MCP-capable client
Bob uses can call it, and Dessau Chat, when it grows an MCP client, does the
same. The gateway itself is untouched: it holds no key, reads no
conversation, and reaches no new host; only the sidecar talks to Brave, only
for the clients Alice granted, and the panel shows her what it has spent.
Alice is Brave's customer for those searches, and the panel says so in one
sentence when she turns it on: the terms she accepted bind the people she
grants it to, and what she grants she can take back.

## Why This Matters

Some of the people on Alice's network are hers to look after, and a key per
person is the wrong shape for a household or a small team. The 2026-09-10
verdict on server-side search killed a key inside the gateway and left one
shape standing: a separate, opt-in sidecar outside the gateway for
MCP-capable clients. This intent is that shape, with the one addition the
maintainer asked for on 2026-09-20: it is shared with some clients, not all,
and the grant rides the pairing the clients already have.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

Seeded by the filing session, unconfirmed; every bullet is a proposal for the
planning interview:

- Given the Clients tab, when Alice turns Search on for one paired client,
  then that client can reach the sidecar and every other client is refused
  with the same answer an unknown client gets.
- Given the sidecar is on, when a granted client calls it, then the request
  is authenticated by the client's pairing identity, the search goes to Brave
  under Alice's key, and the gateway's own listeners carry none of it.
- Given the gateway, when the sidecar is running, then the gateway's outbound
  hosts are unchanged and no request log line carries a query.
- Given the panel, when Alice looks at Search, then she sees the month's
  search count and the sentence saying she is Brave's customer for the
  clients she grants.
- Given Alice revokes a client, when it calls again, then it is refused at
  once.
- Given `config.json`, when the key is set there or in the panel, then it is
  stored the way the API key is, never crosses accounts under the shared
  cache, and a save that does not touch it is never refused over it.

## Open Questions

- Whether the sidecar is a second listener of the same binary or a separate
  process, and where its port lives in the config.
- How a granted client discovers the sidecar (a field in the Bonjour TXT
  record, the panel's Connect tab, or both).
- A per-client or total monthly spend cap, and what the client sees when it
  is reached.
- Attribution: the terms want it displayed conspicuously, and an MCP client
  the product does not own renders what it likes.
- Whether Dessau Chat gains an MCP client in this intent or a later one.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
