---
id: adr-2609182357322050
slug: paired-clients-speak-to-the-server-over-tls-with-a-pinned-ce
status: accepted
date: 2026-09-19
supersedes: null
superseded_by: null
related_intents: [itd-2609182357325215]
related_rfcs: []
related_adrs: [adr-2609091123526871, adr-2609181004167097]
---

# ADR-2609182357322050: Paired clients speak to the server over TLS with a pinned certificate and a keypair of their own; unpaired OpenAI clients keep plain HTTP on the LAN

## Context

The server speaks plain HTTP on the local network, bound as adr-2609091123526871
decided, and a client proves itself with one shared bearer key typed into
Settings. The maintainer asked (2026-09-19) for each client instance to
register a unique identity with the server and for every interaction to be
encrypted "via this unique hash". The second half cannot be built as asked: an
identifier both sides hold is not a secret key, and one that travels over the
same network to register is not secret at all. What the ask needs is two
standard things — an identity per client the server can name and revoke, and
encryption in transit — and both have one sanctioned shape on Apple platforms.
This is a trust-boundary decision (`internal/gateway`, `internal/config`) and
is taken here before any code.

## Decision

We will:

1. **Encrypt in transit with TLS, not with an application-layer scheme.** The
   server makes itself a certificate on first start (a self-signed leaf for
   its `.local` name and its addresses, kept beside `config.json`, never
   leaving the Mac) and serves HTTPS on a second port beside the plain one.
2. **Pin, not trust.** A client pins the server's certificate at pairing — the
   fingerprint is what it pairs with, advertised in the Bonjour TXT record and
   shown on the panel — and from then on refuses any other certificate for
   that server. No certificate authority is involved and none is asked for.
3. **A keypair per client, made where it lives.** At pairing the client makes
   a keypair in its Keychain (Secure Enclave where the device has one, by the
   Security framework, not CryptoKit) and sends the raw public key; the
   server records it under a name the person chooses and signs the client's
   leaf certificate with its own key, because no public Apple API mints one.
   The client proves itself on every request by TLS client authentication
   (mutual TLS); the server checks the presented public key against the
   paired set and builds no chain. The private key never leaves the client
   and the server never holds a secret for it. The server's own key persists
   across restarts, so its leaf may be reissued without re-pairing; only a
   new key re-pairs every client. The SOTA note of 2026-09-19 is the source.
4. **Pairing is first-come, then revoke.** Any client on the LAN that reaches
   the server's pairing endpoint pairs itself — the maintainer's choice at the
   interview (2026-09-19) over a displayed code or a click on the panel — and
   the panel lists every paired client with its name, its pairing time and
   its last sight, revocable with one click, which the server enforces on the
   next request. What this protects against is stated plainly on the docs
   page: a paired client cannot be impersonated after pairing, but at the
   moment of pairing an impostor on the LAN can pair too and is only caught by
   Alice reading the list. A code or a click may be added by a later ADR.
5. **Unpaired OpenAI clients keep plain HTTP and the shared API key on the
   LAN,** exactly as today, so curl and every script go on working; the
   plain port is the bind mode's and unchanged. A paired client uses HTTPS
   only.
6. **What is recorded stays what it is:** a paired client's name and key
   fingerprint in `config.json` beside the API key, redacted where the key
   is; the log names a client by its pairing name, never its key.

## Alternatives Considered

- **Encrypt with the registered id as a shared secret.** The ask's words; not
  a secret, and not a key. Rejected as unsound.
- **A pre-shared key per client entered by hand (like the API key today) with
  an authenticated-encryption layer over plain HTTP.** Works but invents a
  transport no OpenAI client speaks, breaks every script, and is what TLS
  already does with hardware-backed keys. Rejected.
- **TLS with a certificate authority (a public one, or one the server runs).**
  A public CA cannot issue for a `.local` name; a private CA is a second thing
  to keep and revoke, and pinning at pairing gives the same guarantee for one
  server. Rejected for now; an ADR may supersede this if a fleet of servers
  needs one.
- **A displayed pairing code, or approve-on-panel.** Each protects the first
  pairing against an impostor on the LAN; the maintainer chose first-come with
  revocation for the friction it saves (2026-09-19). Not chosen; not closed.
- **HTTPS only, plain HTTP off.** Considered at routing (2026-09-19); the
  maintainer chose to keep plain HTTP for unpaired clients so scripts keep
  working. Not chosen.

## Consequences

- Easier: a revocable identity per client; the API key never typed on a
  paired client; encryption in transit for the chat client on both
  platforms with the system's own TLS.
- Harder: the server gains a certificate and a second listener to keep
  healthy and to advertise; the client gains a pairing flow and a pinning
  trust evaluation; the panel gains a clients pane; `config.json` gains a
  clients block with the three-surfaces obligation.
- Obligations: an adversarial security review before the pairing intent
  lands; a test that the plain port's behaviour is byte-for-byte unchanged
  for an unpaired client; a test that a revoked client is refused on its
  next request; the docs page that says what pairing protects against and
  what it does not (a compromised Mac on either end).
