---
id: adr-2609202200197975
slug: who-counts-as-one-client-for-scheduling-a-paired-client-is-o
status: accepted
date: 2026-09-20
supersedes: null
superseded_by: null
related_intents: [itd-2609202108180709, itd-2609202108185678, itd-2609182357325215, itd-2609061441285238]
related_rfcs: []
related_adrs: [adr-2609182357322050, adr-2609061503319212, adr-2609201008476813, adr-2609121450000000]
---

# ADR-2609202200197975: Who counts as one client for scheduling: a paired client is one person by its pairing, unpaired clients are one class served last, and what is written down is the contention, never the client

## Context

Two intents planned on 2026-09-20 need the pool to know whose request it is
holding. itd-2609202108180709 gives the next free turn at a busy model to the
client that has had the fewest answers while others waited, which is not a
decision the pool can take about requests it cannot tell apart.
itd-2609202108185678 answers a client's question about where its own request
stands, which needs "its own" to mean something.

The record already holds two positions on identity, and both are kept here.
The pool's load-waiter queue is shared out per source, and its source is the
presented API key or `""` for a loopback or unkeyed caller — one shared bucket,
because "the pool cannot tell two anonymous callers apart, and an address is
not a client" (`internal/runtime/pool.go`, `loadWaiter.source`). The
security decision that produced that rule, iss-2609070252377294 (resolved),
found that per-caller fairness state on an open endpoint is a denial-of-service
lever for a caller who churns addresses and a persistent per-address record for
everyone else, and closed it by keying on the API key and requiring one before
grace may be switched on for a LAN-exposed server.

Since then the record has gained an identity that IS a client:
adr-2609182357322050 gives a paired client a keypair of its own, proved on
every request by TLS client authentication, revocable by name on the panel;
`pairedOnly` in `internal/gateway/pairing.go` already tags a paired request's
context with `client:<spki fingerprint>` for the pool. What the record lacked
was a decision on whether that identity, and only that identity, is what
scheduling may see — and what happens to the unpaired clients that have none.

The privacy posture bounds the answer. adr-2609061503319212, restated by
adr-2609201008476813, decides that local telemetry never leaves the Mac and
that the statistics store holds "no prompt, no answer, no key, no client
address", held by byte-scanning tests; `docs/logging.md` promises the log
names no client address either. Two adversarial reviews of the drafts
(2026-09-20) flagged that a draft's "unpaired client told apart in memory only"
could only mean an address, and that this would reverse both the pool's rule
and the resolved issue in the very file the feature extends. The maintainer
settled it at the planning interview.

## Decision

We will:

1. **A paired client is one person, by its pairing identity and nothing
   else.** The identity the pool schedules on is the SPKI fingerprint the
   TLS handshake proved, carried as the source tag `pairedOnly` already sets.
   It is never a name (two clients may choose one name), never a header a
   plain-port client could also send, never an address. The Discord bridge's
   requests are one paired client's (the bridge is one identity of the
   server's own, tagged where `Gateway.Ask` acquires the pool).
2. **Unpaired clients are one class.** Every request that arrives without a
   pairing — the shared API key on the plain port, a loopback client, a
   keyless install's LAN — is scheduled as the unpaired class. Members of the
   class are never told apart from each other by the pool, by any means,
   and no address is used or kept for the purpose. This is the existing
   `""`/key bucket of the load-waiter queue, given a name.
3. **Unpaired requests are served only when no paired request is waiting for
   that model**, in arrival order among themselves, and an unpaired request
   that has waited longer than the queue's maximum wait is refused with the
   existing `waited` state and queue time rather than left waiting. A paired
   person is what the server is for; a script or an unknown caller is served
   from what is left.
4. **The debt lives in memory only.** What a paired client is owed or owes
   is a counter on the pool's own structures, per model, gone when the model
   leaves memory and when the server stops. It is never written to
   `config.json`, the registry, the statistics store or the log, and nothing
   reads it but the pool's admission decision.
5. **What is written down is the contention, never the client.** The
   operational log carries one line each time a request is held back for
   another, naming the model, whether the held request was paired or
   unpaired, and how many were waiting. The statistics carry, per model per
   minute, how many requests waited, for how long, and how many unpaired
   requests were deferred. Neither carries a client's name, fingerprint or
   address, which keeps the store's promise and the logging page's exactly
   as they stand.

This REFINES, and does not reverse, the pool's rule that an address is not a
client and the resolution of iss-2609070252377294: both said the pool may
key fairness only on an identity a caller has proved, and both are kept —
there is still no per-address state anywhere. What is new is that a proved
pairing identity is now such a key beside the API key, and that the class of
callers without one is placed after the callers with one rather than
interleaved with them.

## Alternatives Considered

1. **Tell unpaired clients apart by remote address, in memory only.** The
   drafts' wording. Rejected: it reverses the pool's stated rule and reopens
   the closed security question; an address is reassigned by DHCP, shared by
   a NAT and changed by a roaming device, so it is a weak key as well as a
   forbidden one; and "never written down" would rest on a habit rather than
   on there being nothing to write.
2. **Treat every client alike and schedule on the API key only.** The
   pre-existing shape. Rejected: every unpaired LAN client holds the one
   shared key, so this cannot separate Bob from a script and fairness among
   people is impossible on it. A paired identity exists and is already
   plumbed; not using it would be the gap.
3. **Interleave the unpaired class with paired clients as one more
   participant with its own debt.** Considered: it is what round-robin would
   do. Rejected by the maintainer: a class that any number of callers share
   would receive one person's share between them, which is neither fair to
   them nor safe for the people the server is for, and a burst from the
   class would still take turns from paired people. Paired first, the class
   from what is left, bounded by the queue's wait.
4. **Require an API key before the fair turn may run, as eviction grace
   does.** Rejected as unnecessary: the fair turn keys on nothing an open
   endpoint could be made to fabricate — a pairing is proved by the
   handshake and the class needs no key — so there is no lever the key
   requirement would be fencing off.

## Consequences

- Easier: the pool has one answer to "who is this" for every scheduling
  decision, and it is the answer the pairing ADR already made revocable and
  visible on the panel; a fairness rule can be built at the slot-admission
  layer without a second identity scheme.
- Harder: unpaired clients on a busy model are deliberately behind every
  paired one, and a person using curl on a shared Mac is in the class; the
  docs page for the fair turn must say so plainly, and the intent's refusal
  bound is what keeps the class from waiting for ever.
- Obligations: an adversarial security review before either intent lands
  (`internal/gateway` and `internal/runtime` are trust boundaries); a test
  that no per-address state exists — `RemoteAddr` is read nowhere in the pool
  and reaches no scheduling structure; the statistics store's byte-scanning
  tests extended to the new fields; the log line's shape held by a test; the
  bridge's identity tagged and tested as one paired client.
