---
id: spc-2609201501565589
slug: alice-shares-her-brave-search-with-the-clients-she-chooses-a
intent: itd-2609201407587936
origin: researcher-authored
production_mode: hand-written
---

# The search sidecar: a separate process, granted per paired client

## Summary

This spec delivers itd-2609201407587936: an opt-in search process that Dessau
Server launches and supervises the way it does a model server, on its own
port, speaking the Model Context Protocol, holding Alice's Brave key, and
answering only the paired clients she has granted Search in the Clients tab.
Grants ride the pairing identity the clients already have; a total monthly
cap and a per-client cap bound the spend, and a newly granted client starts
on a trial allowance of fifteen searches. The sidecar's presence and port
join the Bonjour TXT record and the Connect tab. The gateway is untouched:
it holds no key, links no sidecar code, reads no query and logs none. Dessau
Chat gains an MCP client so a granted client searches with no key of its own.
Impact: additive.

## Scope

In: a new entry point `cmd/dessau-search`; a new package `internal/searchd`
(the supervisor, the grant check, the caps and the counts);
`internal/config` (the key, the caps, the grant on a paired client);
`internal/discovery` (two TXT fields); `internal/ui` (the Clients tab, the
Connect tab, a Web search section in Settings); `internal/app` (the
supervisor's lifetime); `internal/archtest`; `client/DessauChat/MCP.swift`;
`docs/web-search.md`. Out: `internal/gateway`, which gains no line of code;
the completions path; the client-held-key intent itd-2609201407580721, which
this one composes with rather than replaces.

## Approach

**A separate process, supervised like a model server.** `cmd/dessau-search`
is a second binary in the same module, built and shipped inside the app
bundle beside the server. `internal/searchd` launches it the way
`internal/runtime`'s `ExecLauncher` launches a model server: a `Spec`, a port
from the same `freePort()` helper, a `Process` with `Done()`, `Stop(ctx)` and
`LogPath()`, and a supervisor that restarts it on exit with backoff and
reports its state the way the pool reports a resident server
(cond-2609201501563457). It is started only when the operator's switch is on
and at least one client is granted, and stopped when neither holds. The key
is passed to the child on its standard input at launch, never in `argv`,
where every other account on the Mac would read it in `ps`.

**How it stays inside the two ratified records.** adr-2609061503319212 — now
restated in whole by adr-2609201008476813 — fixes that nothing about an
install leaves the machine to the project or to a vendor, and that the
outbound connections Dessau makes are the ones that fetch models and
provision the runtime. The sidecar adds an outbound host, and this spec does
not pretend otherwise: it is a host reached only after Alice sets a key and
turns the switch on, only with a query a granted client supplied, only under
Alice's own account with Brave, from a process that carries none of the
app's own state. It reports nothing about the install, which is the clause
the ADR actually protects. The architecture test that walks outbound hosts
gains `cmd/dessau-search` as a named exception carrying exactly that
sentence, so the widening is a reviewed diff rather than a quiet one; and the
2026-09-10 ideate verdict is what admits the shape, having killed the
in-gateway key and left this one standing.

adr-2609061610102325 — restated by adr-2609201008470380 — grants the gateway
one reading of prompt content and no more. The sidecar reads no prompt at
all: it receives a query string from an MCP client that composed it, on a
listener the gateway does not own, in a process the gateway does not link.
The admitted-readers list in `internal/archtest/prompt_content_test.go` is
therefore unchanged, and an architecture test asserts no file under
`internal/gateway` imports `internal/searchd` or names the search port
(cond-2609201501567875).

**The grant, on the pairing identity.** The sidecar's listener is TLS with
client certificates required, built from the same `pairing.Registry.TLSConfig`
the paired endpoint uses, so a caller proves itself with the key it paired
with. `pairing.PeerFingerprint` yields the SPKI; `internal/searchd` looks it
up in the granted set and, on a miss, answers exactly what it answers an
unknown client — one refusal shape for the ungranted, the unknown and the
revoked, so a caller cannot tell from the answer whether the pairing exists
(cond-2609201501562976). Revocation is immediate: the grant is read per
request from the live config, never cached for the connection's lifetime.

**Caps and counts.** `internal/searchd` holds a monthly counter in a bounded
JSON file in the app's own support directory, written through the discipline
`internal/selftest/results.go` uses — `{month, total, per_client: {spki:
count}}` and nothing else. No query, no result, no timestamp per search.
Before a search the supervisor checks the total cap and the caller's cap; the
total being reached tells every caller that search is paused until the month
turns, a per-client cap being reached tells that caller alone. A client
granted with no cap of its own gets the trial allowance, fifteen searches
unless Alice has changed the default in Settings (cond-2609201501561535), and
the Clients tab shows the figure beside the grant.

**Configuration, and the three surfaces.** `config.Config` gains a `Search`
block — `enabled`, `key`, `monthly_cap`, `trial_allowance` — and
`config.Client` gains `search bool` and `search_cap *int`. The key inherits
the API key's rules exactly: a `MaxSearchKeyBytes` ceiling with a
`sanitizeSearchKey` that trims and says so, so a hand-edited file still
loads; the same file mode, so it never crosses accounts under the shared
cache; and the anti-wedge rule this repository has built three times — a save
that does not touch the key is never refused over it, and the panel, which
never renders the key back, sends the field absent to mean unchanged
(cond-2609201501568354). Because each of the four settings carries a JSON
tag, `internal/archtest/settings_surface_test.go` fails in both directions
until the Settings pane names every one of them and the script's submit body
matches: that test is the arming of the three-surfaces rule for this change,
not a habit. The Clients tab's per-client grant and cap are covered by their
own `internal/ui` tests, since they are per-row rather than pane settings.

**Being found.** `discovery.Advertiser` gains a `SearchPort func() int`
callback and `txtRecord()` gains two keys on the both-or-neither rule the
`spki`/`tlsport` pair already follows: `search` reading `mcp`, and
`searchport` reading the port, present only while the sidecar is up. `txtvers`
stays `1`: the record is additive and a reader that does not know the keys
ignores them, which is the rule the record already lives by
(cond-2609201501562586). The Connect tab shows the same address in the
same form it shows the API endpoint.

**The panel.** The existing `search` tab is Find Models and is not touched;
the sidecar's controls are a **Web search** section in Settings (the switch,
the key, the two caps, the trial default) and a Search column in the Clients
tab (a toggle and a cap per paired client, with the month's count beside
it). Settings carries, once, the sentence the intent requires: that Alice is
Brave's customer for the clients she grants, that the terms she accepted bind
the people she grants it to, and that what she grants she can take back
(cond-2609201501561311). What a granted client renders is the client's.

**Dessau Chat's MCP client.** A new `client/DessauChat/MCP.swift` speaks
JSON-RPC to the sidecar over the pairing session — the same `URLSession`
delegate that answers the server-trust and client-certificate challenges
today, so no second identity and no key of its own. It discovers the sidecar
from the TXT record the client already browses, or from an address pasted in
Settings. Composed with itd-2609201407580721 the rule is one line: Bob's own
Brave key wins where he has set one; where he has not and a grant is
advertised, the sidecar answers; the client's web-search switch is off by
default either way (cond-2609201501569102).

## Acceptance Criteria, and what holds each

- **A grant in the Clients tab lets exactly that client reach the sidecar;
  every other is refused as an unknown client is**: `internal/searchd` tests
  over the listener with three peers — granted, paired-but-ungranted, unknown
  — asserting one answer shape for the latter two.
- **A granted call is authenticated by the pairing identity, searches under
  Alice's key, and the gateway's listeners carry none of it**: an
  `internal/searchd` test against a fake Brave endpoint asserting the
  subscription header is the configured key and the peer is resolved through
  `pairing.PeerFingerprint`; plus the architecture test that no gateway file
  imports the package.
- **The gateway's outbound hosts are unchanged and no request log line
  carries a query**: the outbound-host architecture test, with
  `cmd/dessau-search` as its one new named exception and the gateway's own
  list unchanged; and `TestOnlyTheMergeReadsPromptContent`, green and
  untouched.
- **The panel shows the month's count, the caps and the customer sentence**:
  `internal/ui` tests over the Settings pane and the Clients tab markup, and
  `internal/archtest/settings_surface_test.go` in both directions for the
  four configuration settings.
- **A revoked client is refused at once**: an `internal/searchd` test that
  revokes between two calls on one connection and asserts the second is
  refused.
- **The total cap pauses everyone; a per-client cap pauses that client
  alone**: `internal/searchd` cap tests asserting the two refusal texts and
  that the counter rolls at the month boundary.
- **A grant with no cap gets the trial allowance, fifteen unless Settings
  says otherwise, shown beside the grant**: a config default test, a cap test
  over the effective allowance, and an `internal/ui` test that the Clients
  row renders the figure.
- **The sidecar's presence and port are in the TXT record, and the Connect
  tab shows the same address**: `internal/discovery` tests for the
  both-or-neither pair and their absence while the sidecar is down; an
  `internal/ui` test for the Connect tab; and a `make run` hand check with a
  Bonjour browse, recorded in the shipping line.
- **A crash is restarted and reported the way a model server's is, and the
  gateway serves on**: an `internal/searchd` supervisor test over a fake
  launcher whose process exits, asserting the restart, the backoff and the
  reported state; and a hand check killing the child while a completion
  streams.
- **A granted Dessau Chat searches through the sidecar with no key of its
  own, off by default**: an architecture test over the Swift source —
  `MCP.swift` uses the pairing session, names no Brave host and no key store,
  and the switch's literal default is false — plus a recorded hand check on a
  granted client and on an ungranted one. The client has no XCUITest target
  (itd-2609170718438919), so the behaviour on screen is the hand check and
  the structure is the architecture test.
- **The key is stored the way the API key is, never crosses accounts, and a
  save that does not touch it is never refused over it**: `internal/config`
  tests for the ceiling, the sanitiser and the load of a hand-edited file; a
  save-path test in `internal/gateway`'s control plane asserting an absent
  key field leaves the stored key intact; and the shared-cache file-mode test
  the API key already has.

Before the pull request: the adversarial security review, which this change
needs on four counts — a new network listener, a subprocess, a third-party
secret and a new outbound host — and a docs-currency review of
`docs/web-search.md`. One dated line goes in `.abcd/work/DECISIONS.md` when
the outbound-host exception is written, because it is the first widening of
that list since the record was set.

## Departures

None at the time of writing.
