---
id: itd-2609201012167478
slug: gropius-becomes-dessau-alice-installs-dessau-server-bob-s-de
spec_id: spc-2609201245369767
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
impact: breaking
origin: researcher-authored
production_mode: hand-written
---

# Gropius becomes Dessau: Alice installs Dessau Server, Bob's Dessau Chat finds her Mac by its own name

## Press Release

Alice runs the documented one-line install command and what lands in her
Applications folder is Dessau Server, a menu-bar app that downloads MLX
models and serves them to her network exactly as before, under a new name.
Bob opens Dessau Chat on his Mac and it lists Alice's server as "Alice's
Mac", the name her Mac already has in System Settings and on the tailnet,
with no product prefix in front of it, because the service type already
says what it is. He pairs, and everything he could do yesterday he can do
today. Carol, whose coding tool only wants a base URL and a model name,
reads the models list and sees `owned_by: dessau`; the response headers she
graphs are the `X-Dessau-` family. The repository, the module path, the
release assets, the bundles, the config directory, the Bonjour service type
`_dessau._tcp`, the site at intentdriven.sh/Dessau/ and every page of the
documentation carry the one name, and the name guard refuses the old one in
user-facing content from this change on. The change is breaking: it is
pre-1.0, there is no migration, and Alice re-creates her server state and
Bob re-pairs his clients once.

## Why This Matters

The current family name collides with an active, published developer tool
aimed at the same developers, and the moment the wire identifiers freeze at
1.0 the family name is fixed with them. adr-2609200729102059 decided the
name, the component naming, the frozen wire identifiers and the
instance-name rule; this intent is the one change that carries that decision
into the tree, the forge, the release and the site, while the pre-1.0 rule
still makes it cheap.

## Mechanism

We expect one mechanical rename to hold across the whole product because
every place the family name lands is already pinned by a test or a guard:
the architecture tests hold the service type to one value across server and
client, the identity check holds every rendered surface to the identity
block, the name guard refuses a superseded name in user-facing content, and
the site test renders the page against the identity block. So the sweep is
complete when those checks are green with the new name and the old one is
banned, and a stray mention is a red check rather than a reader's discovery.
What would show this wrong: a user-facing surface no guard covers that still
says the old name after the checks pass, or a paired client that finds the
server under the old type because a second advertisement was left behind.

## Scope Conditions

- The rename lands as one branch and one pull request, followed by release <!-- cond: cond-2609201245364256 -->
  0.8.0 as a breaking pre-1.0 release; nothing is migrated, and no
  compatibility shim exists for the old service type, the old config
  directory, the old headers or the old bundle identifiers.
- User-facing content, the code, the build, the forge and the site are <!-- cond: cond-2609201245362115 -->
  swept; the dated record under `.abcd/` (shipped intents, closed specs,
  resolved captures, ratified ADRs) is history and keeps the name it was
  written under; the living records (`CONTEXT.md`, `IDENTITY.md`, the
  principles, open captures, drafts, planned intents, open specs, the ADR
  index rows) are swept.
- The Bonjour instance name is the Mac's Computer Name from System Settings <!-- cond: cond-2609201245363671 -->
  (the value `scutil --get ComputerName` reports), not the DNS-safe
  LocalHostName the advertiser uses today; the address-record host label
  stays a DNS-safe derived label.
- The IANA service-name registration and the trademark database search stay <!-- cond: cond-2609201245368242 -->
  with the maintainer; the type is used unregistered until then, as the old
  one was.
- The repository rename on the forge happens before the module path changes, <!-- cond: cond-2609201245361496 -->
  so the forge's redirect exists before any reference points at the new name.

## Acceptance Criteria

- Given a Mac at the floor, when Alice runs the documented install command,
  then the release it resolves from the new repository name is 0.8.0, the
  asset it verifies is named for Dessau Server, and the menu-bar app that
  launches names itself Dessau Server.
- Given Dessau Server is running, when Bob browses the local network for
  `_dessau._tcp`, then he finds one instance whose name is the Mac's
  Computer Name with no prefix, whose TXT record still carries `txtvers`,
  and no instance of the old type is advertised.
- Given a request to the models list, when Carol reads a model entry, then
  `owned_by` is `dessau`, and the queue headers on a proxied response are
  `X-Dessau-State` and `X-Dessau-Queue-Time`.
- Given the module path, the binary, the bundles and the config directory,
  when the tree is built, then the module is `github.com/intentdriven/Dessau`,
  the bundle identifiers, the config directory and the shared root carry the
  new name, and `make test`, `gofmt -l .`, `go vet ./...` and
  `client/build.sh` all pass.
- Given the name guard, when a docs page or the README names the old family
  name, then `abcd docs lint` fails on it as a superseded product name, and
  `abcd identity` reports zero surfaces adrift from the new identity block.
- Given the site workflow, when release 0.8.0 publishes, then
  intentdriven.sh/Dessau/ answers with the page and a request for the old
  path is redirected to it.
- Given the changelog, when 0.8.0 is cut, then the entry says
  `impact: breaking` and tells Alice she re-creates state and Bob re-pairs.

## Open Questions

All four were put to the maintainer at the 2026-09-20 planning interview and
are decided (DECISIONS.md, 2026-09-20):

- Bundle and asset naming: `DessauServer.app` and `DessauChat.app`, assets
  `DessauServer.app.zip` and `DessauChat.app.zip`, binary `dessau`, display
  names "Dessau Server" and "Dessau Chat". No space in any path a script,
  workflow or checksum line handles.
- Bundle identifiers: `sh.intentdriven.dessau.server` and
  `sh.intentdriven.dessau.chat`, rooted in a domain the project serves from;
  the client's Keychain service, log subsystem and queue label follow.
- Record sweep: living records only. `CONTEXT.md`, `IDENTITY.md`, the
  principles, open captures, drafts, planned intents, open specs and the ADR
  index rows are swept; shipped intents, closed specs, resolved and wontfix
  captures and ratified ADR bodies keep the name they were written under.
- Release codename: Prellerhaus, and a codename spans a major line rather
  than one minor: Prellerhaus names every release up to 1.0.0, the next
  codename names everything up to 2.0.0. It appears in the release title
  only, never in an identifier.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: one mechanical rename holds across the product because every place the family name lands is pinned by a test or a guard (service type, identity block, name guard, site render); what would show it wrong is a user-facing surface no guard covers still naming the old family, or a client finding the server under the old type
