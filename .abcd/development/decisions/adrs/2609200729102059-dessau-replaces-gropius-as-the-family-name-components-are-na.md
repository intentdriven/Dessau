---
id: adr-2609200729102059
slug: dessau-replaces-gropius-as-the-family-name-components-are-na
status: accepted
date: 2026-09-20
supersedes: null
superseded_by: null
related_intents: []
related_rfcs: []
related_adrs: [adr-2609182357322050]
---

# ADR-2609200729102059: Dessau replaces Gropius as the family name; components are named descriptively under it, wire identifiers are frozen, and the theme lives in release codenames

## Context

The product is about to stop being one thing. The server will serve a chat
client that is a demonstrator, coding harnesses and other tools that need
only a base URL and model identifiers, and possibly further applications;
and there may be several servers on one tailnet, on different Macs. Every
one of those needs a name, and a name given ad hoc is the hardest thing in
the repository to take back: it lands in the module path, the bundle
identifiers, the Bonjour service type, the response headers, the config
keys, the docs and the release assets. The maintainer asked (2026-09-20)
for a naming convention that holds across all of it, and floated a themed
set: Dessau for the server, Anni for the chat client, Gropius for a further
application, Mies and Flamingo for applications built on one another.

Two facts forced the decision beyond taste. First, the current family name
is taken: an active, published developer tool from a university research
group ships as Gropius, with the `.dev` domain, the GitHub handle, a VS
Code extension and tool papers 2020–2024, aimed at the same developers this
product is for. Second, this is the last cheap moment: the version is
below 1.0.0, breaking changes to stored files and identifiers are accepted
without migration, and the moment the wire identifiers freeze at 1.0 the
family name is fixed with them.

What was already locked: the previous product name is banned from
user-facing content by the committed name guard, so the theme may be drawn
from the school at Dessau but never named as such; the docs personas are
Alice, Bob and Carol; the Bonjour advertiser already uses a dedicated
service type and a machine-derived instance name; and adr-2609182357322050
already advertises pairing material in the TXT record.

The convention was drafted in session, then handed to an independent
research agent briefed to break it. The note is
`.abcd/development/research/notes/2026-09-20-naming-convention-sota.md`;
it broke three of five points and the decision below is the amended form.

## Decision

We will:

1. **Rename the family to Dessau.** The Go module path, the repository, the
   menu-bar bundle, the Swift client bundle, the release asset names, the
   config directory and the identity block all change. The rename lands on
   one branch as one PR. There is no migration: testers re-create state
   and re-pair clients. `impact: breaking`.
2. **Name every component as the family name plus what it does.** Dessau
   Server (binary and bundle `dessau`), Dessau Chat, and for anything
   later Dessau plus a plain noun. The bare family name is never a
   component's display name. The family name spans only components inside
   the product's own trust boundary; a client that can talk to any
   OpenAI-compatible server says so in its docs and never implies it is
   part of the server.
3. **Freeze the wire identifiers and register them.** The service type
   becomes `_dessau._tcp` and is registered with IANA under RFC 6335 as a
   name-only registration; the TXT record keeps the `txtvers` key it
   already carries; the User-Agent, the `owned_by` field of the models list, the
   product-prefixed response headers and the config keys take the family
   name. From 1.0 these never change for branding. The old type is not
   advertised alongside the new one: that is a compatibility shim, and
   pre-1.0 testers reinstall both ends.
4. **Name a server instance after its Mac.** The Bonjour instance name is
   the Computer Name with no product prefix, the type already saying what
   the service is; RFC 6762 §9 handles a duplicate. It matches the
   Tailscale machine name, so a Mac has one identity on the LAN and on the
   tailnet, where discovery is the MagicDNS name and a base URL. Roles are
   aliases, never hostnames.
5. **Put no real person's name on any component or mode.** No persona name
   on the chat client, no historical figure on a style, a daemon or a
   library. Where a person-facing mode needs a label it is a plain word or
   a shape from the school's own vocabulary; the chat client's three
   response styles, if built, are Square, Circle and Triangle.
6. **Keep the theme as release codenames.** Each minor release carries a
   secondary label drawn from the school's buildings, objects and people,
   subordinate to the version number, never appearing in an identifier.
7. **Mint a name only when the thing ships, after a name search.** The
   search covers GitHub, the package registries, the IANA service-name
   registry and a trademark database. A compound that states its layering
   plainly is allowed; a name that encodes layering by allusion is not.
8. **The name guard learns the superseded name.** In the same change that
   lands the rename, Gropius joins the previous product name in the
   committed banned-names layer, and the docs personas stay Alice, Bob and
   Carol.

## Alternatives Considered

- **Keep Gropius and accept the collision.** Zero cost now; the price is
  permanent — shared search results with a tool for the same developers,
  no domain, a handle that is gone, and a family name that can never be
  fully owned. Rejected because the cost only grows from here.
- **A house of brands: Dessau, Anni, Gropius, Mies, Flamingo.** The
  maintainer's first shape. Rejected: every peer developer tool uses a
  family name plus descriptive parts; the projects that carry distinct
  brand names do so with a marketing organisation and a policy that still
  prefixes the family name; and the documented onboarding cost of a themed
  vocabulary scales with the number of names a user must hold. Two of the
  five names collide (an active `anni` media server, a well-known
  vision-language model named Flamingo), and one is a real person whose
  estate licenses the name.
- **A persona-named chat client.** Rejected on the evidence: a woman's
  first name on an assistant is the pattern UNESCO asked the industry to
  end, human-like cues drive over-trust, and the name falsely suggests a
  connection with a person whose rights holder is active.
- **Endorsed double names (Dessau Anni, Dessau Mies).** Rejected: the
  pattern exists to manage marks across thousands of contributors and here
  it doubles the vocabulary for nothing.
- **Cattle-style instance names (`dessau-01`) or a product prefix on the
  instance.** Rejected: RFC 1178 recommends names by machine for a handful
  of hosts, the type already carries the product, and a prefix diverges
  the Bonjour name from the tailnet name of the same Mac.
- **A coined word with no theme.** Most ownable and left open; not chosen
  because Dessau came back clean on every check run and the theme lives
  in codenames either way.
- **Advertising both service types through a transition.** Rejected as a
  migration shim the pre-1.0 rule forbids.

## Consequences

- One rename branch touches the module path, both bundles, the install
  script, the release workflow, the service type in the Go advertiser and
  both client property lists, the docs, the identity block and the name
  guard: roughly 550 files. It is mechanical, and the guard machinery for
  a superseded name already exists.
- The repository rename on the forge redirects the old name, so the
  documented install command keeps resolving the latest release through
  the transition. Existing tags and the current release are untouched.
- The IANA registration is filed once the new type is live; until then the
  type is used unregistered, as the old one was. The IANA search endpoint
  is re-run once before filing.
- Every paired client re-pairs, since the bundle identifier and the
  service type both change; the pre-1.0 rule accepts this and the
  changelog says `impact: breaking`.
- Future components get their names without a discussion: family name plus
  noun, minted at shipping time, after the search. A name proposed before
  the thing exists is parked in a scratch note, never in the record.
- The convention is armed by tests where a test can hold it: the guard
  refuses the superseded name, and the advertiser's instance name is
  asserted to be the Computer Name. The rest is this record.
- A trademark search of Dessau in the USPTO and EUIPO databases is done by
  hand before 1.0; the geographic name is a weak mark and no software use
  was found, but the databases were not queried.
