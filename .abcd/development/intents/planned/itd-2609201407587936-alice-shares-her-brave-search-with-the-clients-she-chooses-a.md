---
id: itd-2609201407587936
slug: alice-shares-her-brave-search-with-the-clients-she-chooses-a
spec_id: spc-2609201501565589
kind: standalone
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
of their own. Search is off until she turns it on. In the control panel's
Clients tab she turns on **Search** for exactly the paired clients she
chooses; the others see nothing new. For a granted client, search is a
small separate process that Dessau Server launches and supervises the way
it does a model server, on its own port, speaking the Model Context
Protocol, so any MCP-capable client Bob uses can call it, and Dessau Chat,
which gains an MCP client with this, does the same with no key of its own.
A client finds the sidecar in the server's Bonjour announcement or copies
its address from the Connect tab. The gateway itself is untouched: it holds
no key, reads no conversation, and reaches no new host; only the sidecar
talks to Brave, only for the clients Alice granted, and the panel shows her
the month's count against a total cap she sets and a per-client cap she
may set. A newly granted client starts with a trial allowance, fifteen
searches unless Alice changes the default in Settings, so a person can try
it before she decides how much of her key they get. Alice is Brave's customer for those searches, and the panel says
so in one sentence when she turns it on: the terms she accepted bind the
people she grants it to, and what she grants she can take back.

## Why This Matters

Some of the people on Alice's network are hers to look after, and a key per
person is the wrong shape for a household or a small team. The 2026-09-10
verdict on server-side search killed a key inside the gateway and left one
shape standing: a separate, opt-in sidecar outside the gateway for
MCP-capable clients. This intent is that shape, with the one addition the
maintainer asked for on 2026-09-20: it is shared with some clients, not all,
and the grant rides the pairing the clients already have.

## Mechanism

Confirmed by the maintainer at the 2026-09-20 interview: we expect a
separate search process granted per paired client to give a household one
key without breaking the gateway's boundary because the pairing identity
already tells the server who a client is, and a process the gateway never
links cannot read its memory or logs. What would show this wrong: Brave's
terms on end users proving unworkable for a home operator, or the grant
being bypassed by an unpaired client.

## Scope Conditions

Confirmed by the maintainer at the 2026-09-20 interview:

- A separate process the server launches and supervises like a model <!-- cond: cond-2609201501563457 -->
  server, on its own port, holding the only key.
- Off by default; a grant is per paired client in the Clients tab, and <!-- cond: cond-2609201501562976 -->
  revocable.
- Found through a Bonjour TXT field and the Connect tab. <!-- cond: cond-2609201501562586 -->
- A total monthly cap and a per-client cap; a newly granted client's cap <!-- cond: cond-2609201501561535 -->
  defaults to a trial allowance of 15 searches, the default configurable in
  the server's Settings; the panel shows the counts.
- Alice is Brave's customer for the clients she grants, said once in the <!-- cond: cond-2609201501561311 -->
  panel; what a client renders is the client's.
- The gateway holds no key, adds no outbound host, reads no query and logs <!-- cond: cond-2609201501567875 -->
  none.
- Dessau Chat gains an MCP client in this intent, so a granted Dessau Chat <!-- cond: cond-2609201501569102 -->
  searches with no key of its own.
- The key is stored like the API key: it never crosses accounts under the <!-- cond: cond-2609201501568354 -->
  shared cache and is never a reason to refuse a save.

## Acceptance Criteria

Confirmed by the maintainer at the 2026-09-20 interview, every bullet walked
and accepted:

- Given the Clients tab, when Alice turns Search on for one paired client,
  then that client can reach the sidecar and every other client is refused
  with the same answer an unknown client gets.
- Given the sidecar is on, when a granted client calls it, then the request
  is authenticated by the client's pairing identity, the search goes to
  Brave under Alice's key, and the gateway's own listeners carry none of it.
- Given the gateway, when the sidecar is running, then the gateway's
  outbound hosts are unchanged and no request log line carries a query.
- Given the panel, when Alice looks at Search, then she sees the month's
  search count, the caps, and the sentence saying she is Brave's customer
  for the clients she grants.
- Given Alice revokes a client, when it calls again, then it is refused at
  once.
- Given the total cap is reached, when any granted client calls, then it is
  told search is paused until the month turns; given a per-client cap is
  reached, that client alone is told so.
- Given Alice grants a client, when she sets no cap for it, then its cap is
  the trial allowance, 15 searches unless Settings says otherwise, and the
  Clients tab shows it beside the grant.
- Given the server is announcing, when a client browses, then the sidecar's
  presence and port are in the TXT record, and the Connect tab shows the
  same address.
- Given the sidecar runs, when it crashes, then it is restarted and reported
  the way a model server is, and the gateway serves on.
- Given a granted Dessau Chat, when its own web search switch is on, then it
  searches through the sidecar with no key of its own; off by default.
- Given `config.json` or the panel, when the key is set, then it is stored
  the way the API key is, never crosses accounts under the shared cache,
  and a save that does not touch it is never refused over it.

## Open Questions

All five were put to the maintainer at the 2026-09-20 interview and are
decided: a separate process the server launches; found through a TXT field
and the Connect tab; a total cap and a per-client cap with a trial default
of 15 searches, configurable in Settings (the maintainer, 2026-09-20);
attribution stated once to Alice in the panel; Dessau Chat gains its MCP
client in this intent.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: some of the people on the network are the operator's to look after, and a key per person is the wrong shape for a household; wrong if the terms on end users make the operator's position untenable, or if nobody on the network uses it
