---
id: spc-2609190031320045
slug: bob-s-chat-client-pairs-with-alice-s-gropius-server-once-and
intent: itd-2609182357325215
origin: researcher-authored
production_mode: hand-written
---
# Bob's chat client pairs with Alice's server once

## Summary

The server makes itself a P-256 key and a self-signed leaf on first start, keeps
them beside `config.json` at 0600, and serves the same handler a second time
over TLS on a second port, on the addresses the bind plan already resolved.
`POST /pair` on the plain port takes a name and a raw public key, mints the
client's leaf with the server's key, and records the name, the key's SPKI
fingerprint and the pairing time in a `clients` block in `config.json`. Both
listeners check a presented client key against that block — once at the
handshake, and again on every request, which is what makes a revocation take
effect on the next request rather than at the next connection. A paired client
is exempt from the API key on the TLS port and on that port alone; the plain
port does not move. The panel gains a Clients pane listing each client with its
fingerprint, its pairing time and when it was last heard from, and a Revoke
button. The Bonjour TXT record gains `spki=`. Both chat clients make a keypair
in the Keychain, pair, pin the server by SPKI on the server-trust challenge and
present the identity on the client-certificate challenge. Stdlib only on the Go
side; no new dependency.

## Scope

In scope: a new package `internal/pairing` (the server's key and leaf, the
paired-set lookup, the SPKI fingerprint, the leaf minting); `internal/config`
(`tls_port`, the `clients` map and its `Client` struct, their validation and
bounds, the ceiling); `internal/gateway` (the `POST /pair` handler, the
`VerifyPeerCertificate` callback, the per-request paired check and the API-key
exemption on the TLS listener only, the clients block on the snapshot and its
revocation through the settings save); `cmd/gropius` (the TLS listener set
acquired from the same bind plan, the `tls.Config`, the second `http.Server`);
`internal/discovery` (`spki=` in the TXT record, from a live callback);
`internal/ui` (the Clients pane and its tab); `internal/archtest` (the client's
new shape: the delegate methods, no ATS pinning, the Enclave-or-fallback
attributes); `client/GropiusChat` (a `Pairing` source file and a
`URLSessionDelegate`); `docs/pairing.md`; `README.md`; the changelog.

Out of scope: a certificate authority and chain verification; a displayed
pairing code or a click on the panel; certificate rotation on a schedule;
revoking by fingerprint from the terminal; pairing more than one server at a
time from one client; `gropius clients` as a verb (the clients block is read
through `gropius config show` like every other setting); any migration for the
new `config.json` block.

## Approach

### The server's identity — `internal/pairing`

`Identity` holds a P-256 private key and a self-signed leaf. `Load(paths)` reads
`server-key.pem` from `Paths.Account` through the package's regular-file reader,
so a symlink planted where the key goes is refused rather than followed, and a
file whose mode is not 0600 is refused rather than repaired — a key another
account could have read is not a key, and the shared install leaves the root
group-writable at 3775. When the file is absent the key is generated and written
the way `config.Save` writes: a random temp name in the target directory,
`Chmod(0600)`, write, sync, rename.

The leaf is derived, never stored: it is minted from the key at every start, for
the SANs the bind plan names plus this Mac's `.local` name and loopback, valid
for a year. So a changed address or a renamed Mac reissues the leaf and no
client re-pairs — the pin is on the SPKI, which is the key's, and the key
persists. Only deleting the key file re-pairs everybody, and the docs say so.

`Fingerprint(der []byte) string` is `base64(sha256(RawSubjectPublicKeyInfo))`,
RFC 7469, and is the one place that hash is computed.

`MintLeaf(pub crypto.PublicKey, name string)` signs a client leaf with
`x509.CreateCertificate`, `ExtKeyUsageClientAuth`, the chosen name as the common
name. There is no CSR: proof of possession is the handshake.

### The paired set — `internal/config`

```go
type Client struct {
    Name     string `json:"name"`
    SPKI     string `json:"spki"`
    PairedAt int64  `json:"paired_at"`
}
```
keyed in `Config.Clients map[string]Client` by the SPKI fingerprint, so a client
that pairs twice with the same key updates its entry rather than making a second.
`Validate` bounds the map at `MaxClients` (64, the same reasoning as `MaxModels`)
and each name at `MaxClientNameBytes` (128), refuses a name carrying a control
character or a line break, and refuses an SPKI that is not 44 base64 characters
decoding to 32 bytes. `Config.Clone` copies the map, or a posted body would reach the live
authorization list before `Validate` ran and stay there when the save was
refused — the hazard `Clone`'s own comment describes.

The clients block is excluded from the settings round trip altogether. A
collection cannot round-trip through the secret placeholder the way a string
does: the form would be served redacted fingerprints and would post them back,
destroying every pairing. And a settings body carrying a clients map would make
`POST /api/settings` a second pairing endpoint, loopback-only but asking for no
credential, through which another account on this Mac could write itself into
the authorization list with any pairing time it chose. So `redactConfig` drops
the block before the settings GET, and `applySettings` always restores what is
stored: **no settings save can add, alter or remove a client.** That is the
strongest form of the rule that a save is never refused over something the
operator did not touch — it cannot touch them at all. Pairing and revoking are
routes of their own.

The fingerprint is shown in full and is not redacted. It is a hash of a public
key, so there is nothing in it to keep, and Alice needs the whole of it to
compare against what her client shows. That narrows adr-2609182357322050's
"redacted where the key is" to what it is about — the API key — and is recorded
as a decision.

A malformed clients entry is **sanitized** at load with a notice, never refused.
A refusal makes `Load` return `Default()`, which carries no API key, which makes
`secureExposedBind` mint and save a new one — breaking every OpenAI client on
the network over one bad row. This follows `sanitizeModels` and `sanitizePreload`
exactly.

Where the last sighting lives is a decision and not an omission. It is held in
memory by the gateway, reported on the snapshot, and never written: persisting
it would fsync the operator's settings file on every request and race every save
they make. The panel says "not since this server started" for a client that has
not been heard from.

`tls_port` is an ordinary `int` setting, defaulting to `port + 1`, validated in
the same range as `port`. A value equal to `port` is **sanitized to "no TLS
listener" with a notice**, never refused: refusing it would refuse a save over a
field the operator did not touch, which is the wedge this repository has built
three times.

### The second listener — `cmd/gropius`

After `secureExposedBind` has had its say and before anything is served, the TLS
listener set is acquired from the *same* plan: `plan.Addrs(cfg.TLSPort)`, in the
same order, loopback first. A TLS address that will not bind narrows the TLS
surface and logs why; it never widens, and it never makes the process exit,
because the plain port is the singleton's contention point and the TLS port is
not. The listeners are wrapped with `tls.NewListener` over one `tls.Config`
carrying the identity's certificate, `ClientAuth: tls.RequireAnyClientCert` and
a `VerifyPeerCertificate` that looks the presented SPKI up in the live paired
set. The check is in `VerifyConnection` and not in
`VerifyPeerCertificate`, and the code comment says why so that nobody
"simplifies" it back: Go does not call `VerifyPeerCertificate` on a resumed
connection at all — it restores the peer certificates from the session ticket
and checks only `NotAfter`, and under `RequireAnyClientCert` not even the chain
check runs. Measured against the standard library: a revoked client was served
on every resumed request while `VerifyPeerCertificate` ran once in four.
`SessionTicketsDisabled` is set as well, so nothing rests on one hook.

A second `http.Server` serves them. It carries the gateway handler and `/pair`
and **never `Control.Handler()`**: the control plane's cross-origin reasoning
was written for one plain-HTTP loopback origin, and serving it on a second
origin is a change nobody has reviewed.

`secureExposedBind`'s lockdown path closes the TLS listeners along with the
plain ones. `closeExtra` closes `lns[1:]` today; a TLS set held separately would
survive it, leaving an HTTPS socket open on the LAN in the exact branch where
the API key could not be saved and the panel has been told "serving this Mac and
nothing else".

### Admission — `internal/gateway`

Three changes, and the third is the one the obligation turns on.

`POST /pair` is mounted on the handler both listeners share, and refuses
anything but a plain connection — the pairing call is what a client makes before
it has an identity. Body, name and key are bounded before they are parsed; the
key must be exactly a P-256 SPKI; the name must pass the grammar `Validate`
holds. The leaf is minted, and the entry is written by a method on `Control` that takes
`settingsMu` and then calls `App.SetConfig` — the same order the settings
handler uses (adr-2609091239058072, rule 1), so pairing, revoking and a save
serialise against each other and no new lock enters the order. The answer
carries the leaf DER and the server's own fingerprint, the second for display
only.

The client leaf is minted for ten years, and the reason is written down: the
leaf carries the key, the *paired set* carries the authorization, and revocation
is Alice's and takes effect on the next request. A short leaf would de-pair
every client at expiry with no renewal path, because the client cannot mint its
own.

`withAuth` is left byte for byte alone on the plain port. The exemption for a
paired client is a wrapper on the TLS server's handler only, and it decides from
`r.TLS.PeerCertificates[0]`'s SPKI and from nothing else — never a header, never
the remote address, never a context value the plain handler could also carry.

The per-request check is the revocation mechanism. `VerifyPeerCertificate` runs
once per handshake; an HTTP/1.1 keep-alive and every HTTP/2 stream on one
connection reuse it, and a chat client holds a connection open precisely because
it streams. So the same lookup runs again on every request from the presented
certificate, and a client revoked between two requests on one connection is
refused on the second. The same read records the sighting and the name the log
line uses.

The lookup is keyed on the SPKI and on nothing else — never the leaf's common
name or serial, which are attacker-chosen under `RequireAnyClientCert` because
it verifies no chain. A test presents a self-signed leaf carrying a paired
client's name and watches it refused.

The log names a client by its pairing name, as a `slog` attribute — which quotes
it — with the first eight characters of its fingerprint beside it, because two
clients may share a name and that is precisely what an impostor does. Never the
key.

### The panel — `internal/ui`

A Clients tab of its own, beside Connect and Posture, so that it does not
collide with the Settings pane. Each row is built with `textContent` and never
`innerHTML`: the name is chosen by whoever paired, and the control plane it is
drawn on has the whole settings API behind it. A row carries the name, the
fingerprint, the pairing time, the last sighting and a Revoke button that posts
that client's fingerprint to `POST /api/clients/revoke` — its own route under
`loopbackOnly`, not a settings save. The server's own fingerprint is shown above
the list, which is the only way Alice can check out of band what a client
pinned.

The `clients` block is a setting and reaches the panel as one, so the
three-surfaces enumeration passes with a control rather than an exemption.

### Discovery — `internal/discovery`

`txtRecord` gains `spki` and `tlsport`, from live callbacks on `Advertiser`,
wired in `cmd/gropius` the way `AuthRequired` already is. `spki=` is 49 bytes
against RFC 6763's 200, and it changes only when the server's key changes, which
is the same event that re-pairs every client anyway.

`spki=` is **display-only and is never what a client pins.** mDNS is
unauthenticated multicast: anything on the link can answer with a record of its
own, so a value taken from it pins whatever the loudest responder said. What the
client pins is the certificate presented at a TLS handshake it performed itself,
which is stated in the client's section below.

### The clients — `client/GropiusChat`

One new source file, `Pairing.swift`, shared by both platforms.
`PairingStore.makeKey()` attempts `SecKeyCreateRandomKey` with
`kSecAttrTokenIDSecureEnclave` and falls back, on `errSecMissingEntitlement`, to
a permanent non-Enclave Keychain key — one path, two attempt dictionaries, so a
team-signed build gets the Enclave without a second design. The public key is
exported with `SecKeyCopyExternalRepresentation`, which returns X9.63 and not
SPKI, so the fixed 26-byte P-256 header is prepended before it is sent. The
returned leaf is stored with `SecItemAdd`, and the identity is found by
`kSecAttrApplicationLabel` — the key's public-key hash — because the macOS file
keychain overwrites a certificate's `kSecAttrLabel` with its common name.

**What the client pins, and where it comes from.** Not the TXT record, and not
the fingerprint in the pairing response — both arrive over channels anything on
the link can answer on, and a pin taken from one of them pins whatever an
attacker said. Immediately after pairing the client opens a TLS connection to
the server's TLS port and records the SPKI of the certificate presented at *that*
handshake. It then shows the fingerprint, and the panel shows the server's, so
Alice can compare the two. That comparison is the only thing that turns
trust-on-first-use into a check, and the docs page says so: an active attacker
present at the moment of pairing is pinned in the server's place, permanently,
with the lock showing, and nothing but the comparison catches it.

Once a server is pinned, a certificate that does not match is a refusal with a
reason and never an offer to pair again.

`PinningDelegate` answers both challenges. On server trust it reads the leaf's
key out of `SecTrustCopyCertificateChain` — checking the chain is non-empty
before indexing it, since the far end chooses what it sends — prepends the same
header, hashes, and compares against the pinned fingerprint; the comparison is the gate and the
system's own trust result is never consulted, because App Transport Security
cannot be made to accept a self-signed leaf and `NSPinnedDomains` cannot loosen
trust. On the client-certificate challenge it answers with
`URLCredential(identity:)`. The client builds one long-lived `URLSession` with this delegate and uses it at
the three call sites that use `URLSession.shared` today — one session, because a
session retains its delegate until `invalidateAndCancel` and a per-request
session leaks one every time. The pinned fingerprint is stored in the Keychain
beside the identity and not in `UserDefaults`, for the reason the client already
records about the API key: a preferences plist is rewritable by anything running
as the user, and a pin anything can rewrite is not a pin.

A paired server is spoken to over HTTPS on its TLS port or not at all: the
client never falls back to plain HTTP for a server it has paired, because a
fallback on a TLS error is an off switch anything on the network could reach
for. A lock is drawn beside a paired server's name in the picker, distinct from
the lock that already means "this server wants an API key".

## How the acceptance criteria are met

1. Pairing from the Mac client — the pairing flow above; the Go half by a test
   that a minted leaf's SPKI is admitted; the end-to-end half by hand.
2. Pairing from the iPad client — the same source, `#if`-free; an owed hand
   check, this project owning no iPad.
3. The spike's outcome — recorded in Verification below and in one
   `DECISIONS.md` line.
4. The lock — an architecture test over the client's source; how it renders is
   an owed hand check.
5. An unpaired server — the client's existing path, untouched.
6. No downgrade — the stored server carries its pairing; the architecture test
   pins that no plain-HTTP fallback is written for a paired server.
7-9. The pairing endpoint's bounds, the ceiling and idempotence — `Validate`'s
   new rules and the handler's bounds, each with a table test.
10. The plain port byte for byte — a test that replays the gateway's existing
   request fixtures against a paired-off and a paired-on server and compares
   status, headers and body.
11. An unpaired OpenAI client — the same replay, with and without the key.
12. A revoked client on an open connection — a test that holds one HTTP/1.1
   connection open, revokes between two requests on it, and asserts the second
   is refused. This is the criterion every handshake-level hook fails; a test
   that closes and reopens between the two requests tests nothing.
13. A revoked client off the panel — the settings save's map reset.
14-17. The Clients pane — the pane and its rows; a test that plants a name
   containing markup and asserts it is drawn as text; the three-surfaces
   enumeration passing with a control.
18-19. `spki=` and the panel — the TXT callback and the pane's header.
20. Redaction and an untouched save — the existing secret handling, and a
   `control_untouched_test` row for `clients`.
21. The log — the `slog` attribute, and the existing content tests.
22-23. The server key persisting, and a reissued leaf — the key file is read
   before it is written, and the leaf is derived at every start; tested by
   loading twice from one directory and asserting one key and two leaves.
24. The docs — `docs/pairing.md`.

## Verification

`make test`, `gofmt -l .`, `go vet ./...`, `abcd docs lint`; `client/build.sh`
builds, and `SIM=1 client/build-ipad.sh` if the simulator is present. The
adversarial security review that adr-2609182357322050 obliges runs over the
whole diff before the branch is presented, and its findings are captured before
they are fixed.

**The spike, run before any of the server was written (2026-09-19).** A
throwaway Go server with `ClientAuth: RequireAnyClientCert` and a
`VerifyPeerCertificate` hashing `RawSubjectPublicKeyInfo`, against a throwaway
Swift command-line program built with `xcrun swiftc` and signed exactly as
`client/build.sh` signs the shipping Mac client — `codesign --sign -`, no
entitlements.

- **The Secure Enclave identity does not form under that signature.**
  `SecKeyCreateRandomKey` with `kSecAttrTokenIDSecureEnclave` returns nil with
  OSStatus `-34018` (`errSecMissingEntitlement`): *"failed to add key to
  keychain: `<SecKeyRef:('com.apple.setoken')…>`"*. The Enclave key is made and
  only the Keychain add is refused, for want of a `keychain-access-groups`
  entitlement. Giving the spike that entitlement ad hoc does not help: the
  kernel kills the process on launch, exit 137, because it is a restricted
  entitlement needing an Apple-issued identity and a provisioning profile.
  `kSecUseDataProtectionKeychain` fails the same way on macOS, Enclave or not.
- **The fallback completes the whole chain.** A permanent non-Enclave Keychain
  key pairs, forms the identity, pins the server's SPKI, answers the
  client-certificate challenge with `URLCredential(identity:)` and is admitted:
  the Go side logs `VERIFY ok name="Spike keychain" spki=… sigalg=ECDSA-SHA256`
  and the request returns 200.
- **Decided here, and recorded:** the client attempts the Enclave and falls
  back to a permanent Keychain key on `errSecMissingEntitlement`. The Enclave
  stays an open question on the intent, because the iPad build is team-signed
  from a provisioning profile and may get it where the ad-hoc Mac build cannot.

What the spike did not show, and what stays owed by hand: a real iPad; the Local
Network prompt on a device whose grant has been reset; a second Mac on the LAN.
Everything above ran over loopback on one Mac.
