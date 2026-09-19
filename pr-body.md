Bob's chat client pairs with Alice's server once, and from then on the server
knows it by name. The client makes a keypair on the device it is on, the server
signs it a certificate, and every request afterwards proves itself with that key
over TLS on a second port. A paired client is never asked for the API key. Alice
sees every paired client on the control panel's new **Clients** pane — the name
it chose, its key fingerprint, when it paired, when it was last heard from — and
revokes one with a click. curl and every other OpenAI client go on using the
ordinary port and the shared key, unchanged.

Ships `itd-2609182357325215` under `spc-2609190031320045`, implementing
`adr-2609182357322050`. Fidelity review owed: `rcp-fc75998d7eb2`.

## The spike came first, and it changed the design

Before any of the server was written, a throwaway Go mutual-TLS server and a
Swift command-line client — signed exactly as `client/build.sh` signs the
shipping Mac client — were built to find out whether a Secure-Enclave-backed
identity can be presented by URLSession at all. It cannot, under this app's
signature: `SecKeyCreateRandomKey` with `kSecAttrTokenIDSecureEnclave` returns
`errSecMissingEntitlement` (-34018), and the entitlement that would fix it is
restricted and makes the kernel kill an ad-hoc signed binary on launch. A
permanent Keychain key completes the whole chain. So the client attempts the
Enclave and falls back, one path and two attempt dictionaries, and a
provisioning-profile build takes the Enclave with no other change.

Two more findings changed the design rather than the prose:

- **No handshake-level hook can refuse a revoked client on its next request.**
  Go does not call `VerifyPeerCertificate` on a resumed connection at all, and
  `VerifyConnection` is still per connection — a keep-alive or an HTTP/2 stream
  outlives it, which is exactly what a streaming chat client holds. The lookup
  runs in `VerifyConnection` (with session tickets off) *and* on every request.
- **A fingerprint that arrives over mDNS or plain HTTP pins nothing.** `spki=`
  in the Bonjour record and the fingerprint in the pairing answer are
  display-only; what the client pins is the certificate presented at a handshake
  it made itself, and it shows that fingerprint so it can be compared with the
  one on the panel. `docs/pairing-explained.md` says what that leaves open.

## Tests watched to fail first

- `TestARevokedClientIsRefusedOnItsNextRequestOverAConnectionItAlreadyHad` —
  with only the handshake check: *"a revoked client was served on a connection
  it already had open — the check is per connection, and the request path is not
  checking again"*.
- `TestTheClientAttemptsTheEnclaveAndFallsBack` — with the Enclave attributes
  removed: *"the pairing source names no kSecAttrTokenIDSecureEnclave"*.
- `TestASaveThatChangesTheTLSPortAsksForARestart` — *"a save that moved the port
  paired clients use reported restart=false; the listener is still on the old
  port and nothing said so"*.
- `TestAPairingThatCouldNotBeSavedIsNotAnsweredWithACertificate`,
  `TestASettingsSaveCannotWriteThePairedSet`,
  `TestTheSettingsFormIsNotServedThePairedSet`, and the three architecture
  guards (`internal/pairing` on the enforcement path, the one-per-model-map
  rule, the `tls_port` panel control) each failed before the change that
  answers them.

## Issues

Resolved in this change: `iss-2609190100210971` (a pairing the server could not
record answered 200 with a usable certificate), `iss-2609190100212365` (a failed
pairing left the client unpinned), `iss-2609190100217781` (forgetting a pairing
left its certificate in the keychain), `iss-2609190103062546` (a save that moved
the TLS port reported no restart). Each was captured with `--source impl-review`
before it was fixed.

Captured and left open, outside this lane: `iss-2609190030050506` —
`TestOverridesAreNamedAndNeverValued` searches the whole marshalled statistics
record for the string `777`, so it fails whenever the epoch second contains
those digits. Observed failing on this tree before any change.

## Verified

`make test` (`go test -race ./...`), `gofmt -l .` empty, `go vet ./...` clean,
`abcd docs lint` clean; `client/build.sh` builds; `SIM=1 client/build-ipad.sh`
builds and launches on an iPad Pro 13-inch simulator. Driven end to end against
the built server on one Mac: paired over the plain port, pinned the fingerprint
the handshake presented (which matched the one the server logged), answered 200
over mutual TLS, survived a restart with the same key and the same pairing, and
revoked.

**Owed by hand, and not verified here:** a real iPad; the Local Network prompt
on a device whose grant has been reset; a second Mac on the LAN; and the Secure
Enclave, which needs a build signed with a provisioning profile.

Assisted-by: Claude:claude-opus-5
