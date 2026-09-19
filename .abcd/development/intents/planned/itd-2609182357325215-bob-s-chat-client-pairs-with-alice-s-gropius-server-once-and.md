---
id: itd-2609182357325215
slug: bob-s-chat-client-pairs-with-alice-s-gropius-server-once-and
spec_id: spc-2609190031320045
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609151701196720, itd-2609170718430553, itd-2609180943290800]
severity: major
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# Bob's chat client pairs with Alice's server once

## Press Release

Bob opens the chat client, finds Alice's server in the picker and taps Pair. He
types a name for this device — "Bob's iPad" — and that is the whole of it. The
client makes a keypair of its own, kept in the Keychain and never sent anywhere,
and the server signs it a certificate under the name Bob chose. From then on the
client speaks to the server over TLS on a second port, proving itself with that
key on every request. It never asks Bob for an API key again, and a lock sits
beside the server's name in the picker to say which connection it is on.

Alice opens the control panel and finds a Clients pane. It lists every client
that has paired: the name it gave, the fingerprint of its key, when it paired
and when the server last heard from it. Carol's laptop is on the list too —
under first-come pairing anything on the network that reaches the pairing
endpoint pairs itself, and reading this list is how Alice finds out. She presses
Revoke beside it, and Carol's next request is refused.

Nothing else changes. The plain port answers exactly as it did: curl with the
shared API key, the Python script Bob wrote last month, every OpenAI client on
the network. Alice's server keeps its key across restarts, so it can reissue its
own certificate — after a rename, or a new address — without a single client
having to pair again.

## Why This Matters

Today every client proves itself with one shared bearer key. It is the same
string on Bob's Mac, on his iPad and in Carol's script; it is typed by hand into
each of them; it travels in the clear on the LAN with every request; and the
only way to take it away from one device is to change it for all of them. Alice
cannot answer "which of my devices is that?" because the server cannot tell them
apart, and she cannot answer "who else has it?" at all.

An identity per client answers both. The server names each one, sees when it was
last heard from, and can revoke it alone. And because the identity is a key the
device holds rather than a string the device was told, it is not something a
person can read off one screen and type into another.

## Mechanism

We expect this to work because every piece is system-provided and none of it is
ours to invent: the Go side is the standard library's own mutual TLS —
`ClientAuth: RequireAnyClientCert` with a `VerifyPeerCertificate` that hashes
`RawSubjectPublicKeyInfo` and looks it up, about forty lines by the SOTA note's
count — and the client side is `SecKeyCreateRandomKey` for the key and
`URLCredential(identity:)` on URLSession's client-certificate challenge, which is
the route Apple's own guidance names. The risk is therefore not the volume of
code but whether two Apple pieces meet, so the spike that makes them meet comes
before the server is written. We are wrong if the identity cannot be presented
through URLSession at all — in which case there is no transport here and the
work stops rather than being built on an unproven one — or if the pinning
delegate cannot accept a self-signed leaf without loosening App Transport
Security, which would make the client's half unbuildable as designed.

## Scope Conditions

- Both clients, one source: the Mac client and the iPad client, macOS 27 and <!-- cond: cond-2609190031321992 -->
  iPadOS 27, differing only under `#if os(macOS)`. Swift stays the client and
  only the client.
- The key is made in the Keychain with the Secure Enclave attributes where the <!-- cond: cond-2609190031325339 -->
  device grants them, and with a permanent non-Enclave Keychain key where it
  does not. Which of the two a given build gets is decided at runtime by the
  Keychain's own answer, not by a compile-time platform test.
- Pairing is first-come: anything on the network that reaches the pairing <!-- cond: cond-2609190031322606 -->
  endpoint pairs itself. No code, no click. The maintainer's decision at the
  interview, 2026-09-19.
- What first-come leaves open is accepted, not closed: in the window before <!-- cond: cond-2609190031320071 -->
  Alice reads the clients list, Carol can pair a client of her own, and nothing
  but that list tells Alice so. A displayed code or a click on the panel is not
  chosen and not closed (adr-2609182357322050, decision 4).
- Revocation is enforced on the next request. An answer already streaming to a <!-- cond: cond-2609190031327891 -->
  revoked client is not cut off.
- The plain port and the shared API key are untouched — same port, same bearer <!-- cond: cond-2609190031321181 -->
  check, same loopback exemption, same refusal text. A paired client speaks
  HTTPS on the second port only.
- The second listener follows the bind plan it is given: the same addresses, <!-- cond: cond-2609190031322420 -->
  on its own port. A bind that narrowed to this Mac narrows the TLS listener
  with it (adr-2609091123526871, rules 1 and 3).
- A paired client's name, key fingerprint and pairing time are settings in <!-- cond: cond-2609190031322346 -->
  `config.json`, reached from the panel's own Clients pane, and so are inside
  the three-surfaces obligation. When the server last heard from a client is
  live state and not a setting: it is held in memory, reported on the panel, and
  not written to disk, because a save per request is not a thing a settings file
  should be asked to be.
- The terminal surface is out. `gropius config show` reports the clients block <!-- cond: cond-2609190031325817 -->
  the way it reports every other setting, with the fingerprints shown and no
  verb that writes one.
- Pre-1.0: the new `config.json` block gets no migration. A tester who <!-- cond: cond-2609190031325530 -->
  downgrades loses their pairings and pairs again.
- A build-time architecture test over the client's source, not a client test <!-- cond: cond-2609190031322324 -->
  suite: the client has no test target, so what is assertable is that the
  delegate methods, the Keychain attributes and the pinning comparison are
  present in the source, never that they behave.

## Acceptance Criteria

**Pairing**

- Given Bob's Mac client and Alice's server on the same network, when Bob
  chooses the server and gives it a name, then a keypair is made in the
  Keychain, the server signs its leaf, and the next request is answered over
  TLS on the second port without Bob being asked for an API key. Checked on the
  maintainer's own Mac by hand and recorded in the shipping decision line; the
  automated half is the architecture test over the client's source and the Go
  test that a paired key is admitted.
- Given the iPad client, when Bob pairs from it, then the same holds. This
  project owns no iPad, so it is checked by hand on the maintainer's device and
  recorded as such; the simulator cannot answer for it.
- Given the spike of the Secure-Enclave-backed identity against a Go standard
  library mutual-TLS server, when this work lands, then its outcome — working,
  or failing with the exact error — is recorded in the spec's Verification and
  in one line of `.abcd/work/DECISIONS.md`, and the fallback the result chose is
  named there.

**What the client shows**

- Given a paired server, when Bob opens the picker, then a lock is drawn beside
  that server's name, and it is drawn for the pairing rather than for the API
  key. Held by an architecture test over the client's source; how it renders is
  an owed hand check, since the client has no test target.
- Given an unpaired server, when Bob opens the picker, then the client uses
  plain HTTP and asks for the API key exactly as it does today.
- Given a paired server the client cannot reach over TLS, when it tries, then
  it reports that it cannot reach the server and does not fall back to plain
  HTTP — a fallback would be an off switch anything on the network could reach
  for.

**The plain port does not move**

- Given the gateway's existing request fixtures, when they are replayed against
  a server with no client paired and against the same server with a client
  paired, then every status, every header the gateway sets and every response
  body is identical byte for byte.
- Given an unpaired OpenAI client on the network, when it sends a request with
  the shared API key, then it is answered, and when it sends one without, then
  it is refused with the message it is refused with today.

**The pairing endpoint is not a way in**

- Given a pairing request, when its body, its name or its public key is longer
  than the endpoint accepts, or the name carries a control character or a line
  break, or the key is not a P-256 public key of exactly the expected length,
  then it is refused and nothing is written.
- Given the paired set at its ceiling, when another client tries to pair, then
  it is refused with a reason, so an unauthenticated endpoint cannot grow the
  settings file until the operator's own saves start failing.
- Given a client that pairs twice with the same key, when the second pairing
  arrives, then its entry is updated rather than duplicated.

**Revocation**

- Given a paired client, when Alice revokes it and it sends its next request,
  then the request is refused — including when that request travels down a
  connection the client already had open, which no handshake-level check sees
  again.
- Given a revoked client, when it appears on the panel, then it does not.

**The Clients pane**

- Given two paired clients, when Alice opens the Clients pane, then each is
  listed with the name it gave, its key fingerprint, when it paired and when it
  was last heard from, and each carries a Revoke button.
- Given two clients that chose the same name, when Alice reads the pane, then
  the fingerprints tell them apart, because the name is what the client called
  itself and never what identifies it.
- Given a client name that is HTML, when the pane draws it, then it is drawn as
  text, because the name is chosen by whoever paired and the panel it is drawn
  on has the whole settings API behind it.
- Given the clients block, when the three-surfaces architecture test runs, then
  it passes without an exemption, because the pane is the control.
- Given any settings save, when it is applied, then the paired clients it
  carries are ignored and what is stored is kept — a settings body is not a way
  to write the list of who may connect.

**The fingerprint travels and the secret does not**

- Given an advertising server, when it registers with Bonjour, then the TXT
  record carries the fingerprint of its public key under `spki=`, and the same
  fingerprint is shown on the panel.
- Given a client that has just paired, when it records what it will pin, then
  it pins the certificate presented at a TLS handshake it made itself, and
  never a fingerprint that arrived in a TXT record or in the pairing response —
  both of which anything on the link can answer with.
- Given a paired client, when Bob asks what it is talking to, then it shows the
  fingerprint it pinned, so that it can be compared with the one on Alice's
  panel.
- Given the panel or `gropius config show`, when the settings are read, then the
  API key and the HuggingFace token are redacted as they are today, and a save
  that does not name a client leaves every paired client exactly as it was.
- Given a request from a paired client, when it is logged, then the line names
  the client by its pairing name and never by its key.

**The server's key persists**

- Given a server that has paired a client, when it is restarted, then that
  client's next request is answered without pairing again — the key is the same
  file, and only a new key re-pairs everybody.
- Given a server whose addresses have changed, when it reissues its own leaf,
  then no client re-pairs.

**The documentation says what this is worth**

- Given the docs, when a reader asks what pairing protects against, then they
  are told: a paired client cannot be impersonated after pairing, and what
  passes between it and the server is no longer readable by everything on the
  network. And when they ask what it does not protect against, then they are
  told that too: an attacker present on the network at the moment of pairing
  can be pinned in the server's place, permanently and with the lock showing,
  and comparing the fingerprint the client shows against the one on the panel
  is what catches it; an impostor can pair in the window before Alice reads the
  list; the shared API key still opens the plain port, so pairing takes nothing
  away from the server's own exposure; and neither end is protected from a Mac
  that is already compromised.

## Open Questions

- Whether a Secure-Enclave-backed identity completes a mutual-TLS handshake
  through URLSession. The SOTA note of 2026-09-19 could find no first-hand
  report of one. The spike answers it for the signing arrangement this project
  actually ships; it cannot answer it for every signing arrangement, so the
  Enclave stays open here whatever the spike returns.
- What authenticates the server's fingerprint before a client pins it. It
  arrives in a Bonjour TXT record, which is unauthenticated multicast, and the
  pairing call that follows is on the plain port. Pinning is therefore
  trust-on-first-use, and the first-come window and this question are the same
  window seen from the two ends. A displayed code or a panel click closes both;
  neither is chosen.
- Whether a code or a click is added later. Deferred here rather than declined:
  the maintainer chose first-come for the friction it saves, the research sided
  with approve-on-panel, and the question returns the first time a tester finds
  a client on the list they did not put there.
- Whether a fleet of servers ever needs a private certificate authority.
  adr-2609182357322050 rejects one for now and says an ADR may supersede it.
- Any macOS 27 or iPadOS 27 change to URLSession, Network framework TLS or
  Local Network privacy. Only WWDC25 material was found.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: we expect a per-client keypair proved by mutual TLS on a second port to give Alice a revocable identity per device, because every piece is system-provided — Go's stdlib mTLS on one side and URLSession's client-certificate challenge on the other — so the risk is whether two Apple pieces meet rather than the volume of code; wrong if a Keychain identity cannot be presented through URLSession at all, or if the pinning delegate cannot accept a self-signed leaf without loosening App Transport Security
