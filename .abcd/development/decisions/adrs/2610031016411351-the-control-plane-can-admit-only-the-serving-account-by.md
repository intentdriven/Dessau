---
id: adr-2610031016411351
slug: the-control-plane-can-admit-only-the-serving-account-by
status: accepted
date: 2026-10-03
supersedes: null
superseded_by: null
related_intents: [itd-2610031004535845]
related_rfcs: []
related_adrs: [adr-2609091123526871, adr-2610030906462776]
---

# ADR-2610031016411351: The control plane can admit only the serving account by asking the kernel who owns the connecting socket

**Refines** [adr-2609091123526871](2609091123526871-gropius-binds-loopback-alongside-every-other-address-with-a-p.md)
(its statement that "no check can tell them apart, because from the socket they
are the same connection"), and **supersedes in part**
[adr-2610030906462776](2610030906462776-dessau-serves-from-one-macos-account-the-machine-wide-shared.md)
(Alternative 3's clause that other accounts reach "its control panel" over
`localhost`, which now holds only while the operator leaves the default).

## Context

itd-2610031004535845 (planned 2026-10-03) lets Alice limit the control panel to
the account Dessau serves from. Every check the control plane makes today looks
only at the connection's address, `Host`, `Origin` and `Sec-Fetch-Site`
(`fromThisMachine`, `admitToControlPlane`), none of which carries an account,
and adr-2609091123526871 recorded that no check could tell another account's
request from the operator's. The design review of the intent measured one that
can: the kernel records which process owns each socket, and libproc lets a
process list the sockets of every process it may inspect. A process may inspect
only its own account's processes (another account's answer `EPERM`), so a
loopback connection whose peer socket belongs to one of the serving account's
processes is the serving account's, and every other connection is not. A full
scan of one account's processes took about 1.3 ms on the reviewer's Mac.

## Decision

Decided by the maintainer on 2026-10-03.

**We will admit the control plane, when the operator chooses "this account
only", only to a connection whose peer socket the kernel attributes to a
process of the account Dessau runs as**, refusing every connection it cannot
attribute. The lookup runs once per connection, keyed on the full address pair;
the admit-or-refuse decision is made per request against the live setting, and
open event streams are closed when the setting narrows. The four existing
checks stay in force in front of it. The default is "anyone on this Mac"
(today's rule), so nothing changes until the operator chooses.

Before it ships, a spike proves that connections from Safari, Chrome and
Firefox, from the menu-bar app and from the `dessau` command are attributed;
any that are not would lock the operator out, and are recorded with a remedy
before the choice is offered.

## Alternatives Considered

1. **A secret cookie set by the menu-bar link.** Rejected: browser cookies are
   not scoped to a port, so another account listening on another `localhost`
   port could receive the cookie and replay it; a typed address or a second
   browser would also be locked out.
2. **The panel on a Unix socket.** Rejected: browsers cannot reach one.
3. **A macOS password prompt.** Rejected: the server cannot tell whose login
   session sent a request, so the prompt would appear on the wrong screen.
4. **The kernel's account for the connecting socket (chosen).** Exact for the
   accounts it covers, fails closed, needs nothing from the operator; rests on
   libproc interfaces that `lsof` and Activity Monitor also use.

## Consequences

- The control plane's admission rule gains one clause under "this account
  only"; the docs that say every account can open the panel say it of the
  default.
- An administrator account can act as the serving account and is not kept out;
  anything running as the serving account (a tunnel, a proxy) is admitted as
  the operator.
- On a platform without libproc, the lookup is unsupported and "this account
  only" refuses every connection; Dessau runs only on macOS, where it is
  supported.
- The network option stays declined (decision line of 2026-10-03).
