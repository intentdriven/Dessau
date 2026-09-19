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

<!-- abcd-review: INGESTED receipt=rcp-fc75998d7eb2 -->
Fidelity review — receipt rcp-fc75998d7eb2 (verifier intent-auditor claude-opus-5[1m]).

Provenance: intent-auditor@claude-opus-5[1m] · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:fdc2b837e95ac8998c75444442528c63dde48e3aa723c45dbf17582183ad90be
Input attestations: tree:73d4703^1..73d4703 (PR 93, feat/client-pairing, merged to main); file:line pointers are against the tree as shipped, where `go test ./internal/archtest ./internal/gateway ./internal/config ./internal/pairing ./internal/ui ./internal/discovery ./internal/lifecycle ./cmd/gropius` all pass@-; request:.abcd/.work.local/reviews/rcp-fc75998d7eb2.request.md@sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e; intent:.abcd/development/intents/shipped/itd-2609182357325215-bob-s-chat-client-pairs-with-alice-s-gropius-server-once-and.md@sha256:fdc2b837e95ac8998c75444442528c63dde48e3aa723c45dbf17582183ad90be;

Acceptance rollup: MET 18 · MET_WITH_CONCERNS 7 · NOT_MET 1 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: The automated half is fully delivered — the client's pairing flow makes a Keychain key and pairs, an architecture test holds both TLS challenges in the client's source, and a Go test admits a paired key under a certificate the client signed itself — but the hand check the criterion demands is recorded only as an end-to-end drive whose client was a THROWAWAY Swift command-line program, not the shipping Mac chat app; the shipping decision line does not say which client it was.
  evidence: client/GropiusChat/GropiusChat.swift:486 — "func pair(as name: String) async throws {"
  evidence: internal/archtest/chat_client_pairing_test.go:21 — "func TestTheClientAnswersBothTLSChallenges(t *testing.T) {"
  evidence: internal/gateway/pairing_race_test.go:250 — "func TestAPairedKeyIsWhatAdmits_NotTheMintedCertificate(t *testing.T) {"
  evidence: .abcd/development/specs/closed/spc-2609190031320045-bob-s-chat-client-pairs-with-alice-s-gropius-server-once-and.md:329 — "came up on both ports, logging its fingerprint. A throwaway Swift client made a"
  evidence: .abcd/work/DECISIONS.md:298 — "Driven end to end against the built server on one Mac: paired, pinned the fingerprint the handshake presented, answered over mutual TLS, survived a restart, revoked."
- ac-2 — NOT_MET: The criterion promises a hand check on a real iPad, recorded as such; what the committed record actually states is that a real iPad was NOT verified, and the spec repeats it — so the promised outcome is absent, and the record says so itself. A simulator build is explicitly disclaimed by the criterion.
  evidence: .abcd/work/DECISIONS.md:298 — "**OWED by hand, and not verified here:** a real iPad; the Local Network prompt on a device whose grant has been reset; a second Mac on the LAN; the Secure Enclave"
  evidence: .abcd/development/specs/closed/spc-2609190031320045-bob-s-chat-client-pairs-with-alice-s-gropius-server-once-and.md:343 — "What stays owed by hand, and cannot be shown here: a real iPad; the Local"
  evidence: .abcd/development/specs/closed/spc-2609190031320045-bob-s-chat-client-pairs-with-alice-s-gropius-server-once-and.md:322 — "lint`; `client/build.sh` builds, and `SIM=1 client/build-ipad.sh\` builds and"
- ac-3 — MET: The spike's outcome is recorded in the spec's Verification with the exact OSStatus (-34018, errSecMissingEntitlement) and in one DECISIONS.md line, and the fallback the result chose — a permanent non-Enclave Keychain key — is named in both.
  evidence: .abcd/development/specs/closed/spc-2609190031320045-bob-s-chat-client-pairs-with-alice-s-gropius-server-once-and.md:355 — "- **The Secure Enclave identity does not form under that signature.**"
  evidence: .abcd/work/DECISIONS.md:292 — "So the client attempts the Enclave and falls back on `errSecMissingEntitlement`, one path and two attempt dictionaries"
- ac-4 — MET_WITH_CONCERNS: The pairing lock is drawn with a glyph distinct from the API-key lock and an architecture test holds both facts, so the automated half is delivered; but the owed hand check on how it renders is recorded nowhere in the committed record (the shipping line's owed list names the iPad, the Local Network prompt, a second Mac and the Enclave, and not the lock), and the lock is drawn only on the stored-server row, which the picker shows only while the client is connected.
  evidence: client/GropiusChat/Picker.swift:94 — "Image(systemName: model.pairedHere ? "lock.shield.fill" : "network")"
  evidence: internal/archtest/chat_client_pairing_test.go:187 — "func TestTheLockForAPairingIsNotTheLockForAnAPIKey(t *testing.T) {"
  evidence: client/GropiusChat/Picker.swift:51 — "if !model.serverURL.isEmpty, expanded == nil, model.connected {"
  evidence: .abcd/work/DECISIONS.md:298 — "**OWED by hand, and not verified here:** a real iPad; the Local Network prompt on a device whose grant has been reset; a second Mac on the LAN; the Secure Enclave"
- ac-5 — MET: The request builder takes the paired branch only when a pairing exists for the server in front of the user; everything else falls through to the unchanged plain path that attaches the bearer token, and an architecture test holds that the bearer line still exists and is reached after the paired branch returns.
  evidence: client/GropiusChat/GropiusChat.swift:581 — "if let p = paired, let httpsBase = p.httpsBase {"
  evidence: client/GropiusChat/GropiusChat.swift:596 — "r.setValue("Bearer \(apiKey)", forHTTPHeaderField: "Authorization")"
  evidence: internal/archtest/chat_client_pairing_test.go:129 — "func TestAPairedServerIsNeverSpokenToInTheClear(t *testing.T) {"
- ac-6 — MET: A paired server is addressed over its HTTPS base with no plain-HTTP alternative anywhere in the request builder, the architecture test pins that the paired branch returns before the bearer path is reached, and a transport failure surfaces as a report that the server could not be reached rather than as a retry in the clear.
  evidence: client/GropiusChat/GropiusChat.swift:581 — "if let p = paired, let httpsBase = p.httpsBase {"
  evidence: client/GropiusChat/GropiusChat.swift:644 — "status = "Could not reach \(serverHost). Is Gropius running and on the same network?""
  evidence: internal/archtest/chat_client_pairing_test.go:129 — "func TestAPairedServerIsNeverSpokenToInTheClear(t *testing.T) {"
- ac-7 — MET_WITH_CONCERNS: A replay test drives the same requests at a gateway with nobody paired and at one with two clients paired and compares status, headers and body, and it passes; the concerns are that the fixtures are an eight-row table written for this test rather than the gateway's existing fixture set, that `Date` and `Content-Length` are dropped from the header comparison, and that no successful chat-completion path is among the rows (only the models list, health, and refused completions).
  evidence: internal/gateway/plainport_test.go:23 — "func TestThePlainPortDoesNotMoveWhenAClientPairs(t *testing.T) {"
  evidence: internal/gateway/plainport_test.go:105 — "if k == "Date" || k == "Content-Length" {"
  evidence: internal/gateway/gateway.go:134 — "return g.withAuth(g.routes())"
- ac-8 — MET: `withAuth` is unchanged by the delivery — the only edit to it is that `Handler()` now wraps a `routes()` helper — and the replay covers a request carrying the shared key (answered) and requests carrying none or a wrong one (refused), identically with and without a paired client.
  evidence: internal/gateway/gateway.go:134 — "return g.withAuth(g.routes())"
  evidence: internal/gateway/plainport_test.go:32 — "{"the models list with the key", "GET", "/v1/models", "secret", ""},"
  evidence: internal/gateway/plainport_test.go:23 — "func TestThePlainPortDoesNotMoveWhenAClientPairs(t *testing.T) {"
- ac-9 — MET_WITH_CONCERNS: A table test drives an empty name, a name with a newline, an overlong name, a missing key, a key that is not base64 and a key of the wrong kind, asserting each is refused and that the paired set stays empty; the name grammar refuses every unprintable rune and the key must parse as a P-256 SPKI. Concerns: no case exercises an oversized BODY, and the body bound is an `io.LimitReader` around a streaming decoder, so a request whose first 4 KiB is a complete JSON object followed by more data decodes and is accepted rather than refused; and the 'exactly the expected length' P-256 check is delivered as a curve-and-DER parse rather than a length check.
  evidence: internal/gateway/pairing_test.go:198 — "func TestThePairingEndpointRefusesWhatItCannotStore(t *testing.T) {"
  evidence: internal/config/clients.go:140 — "if r == unicode.ReplacementChar || !unicode.IsPrint(r) {"
  evidence: internal/pairing/pairing.go:156 — "func PublicKeyFromSPKI(der []byte) (crypto.PublicKey, error) {"
  evidence: internal/gateway/pairing.go:145 — "if err := json.NewDecoder(io.LimitReader(r.Body, maxPairBodyBytes)).Decode(&req); err != nil {"
- ac-10 — MET: A client whose fingerprint is not already in the set is refused with 409 and a reason once the set has reached MaxClients (64), before anything is minted or written, and a test drives the endpoint to the ceiling.
  evidence: internal/gateway/pairing.go:177 — "if _, already := cfg.Clients[fingerprint]; !already && len(cfg.Clients) >= config.MaxClients {"
  evidence: internal/config/clients.go:44 — "const MaxClients = 64"
  evidence: internal/gateway/pairing_test.go:236 — "func TestThePairingEndpointStopsAtTheCeiling(t *testing.T) {"
- ac-11 — MET: The paired set is keyed by the key's fingerprint, so a second pairing under the same key overwrites the one row, keeping the original pairing time and taking the new name; a test pairs twice and asserts one client under the later name.
  evidence: internal/gateway/pairing.go:198 — "if was, already := cfg.Clients[fingerprint]; already {"
  evidence: internal/gateway/pairing_test.go:262 — "func TestPairingTwiceWithOneKeyUpdatesTheSameClient(t *testing.T) {"
  evidence: internal/gateway/pairing_test.go:282 — "if got := cfg.Clients[k.fingerprint()].Name; got != "Bob's iPad Pro" {"
- ac-12 — MET: The admission wrapper looks the presented certificate's key up in the live paired set on EVERY request, not only at the handshake, and the test drives two requests down one reused keep-alive connection with the revocation in between and asserts the second is not served.
  evidence: internal/gateway/pairing.go:321 — "client, paired := reg.Sight(spki)"
  evidence: internal/gateway/pairing_test.go:114 — "func TestARevokedClientIsRefusedOnItsNextRequestOverAConnectionItAlreadyHad(t *testing.T) {"
  evidence: internal/pairing/registry.go:111 — "VerifyConnection: func(cs tls.ConnectionState) error {"
- ac-13 — MET: Revoking deletes the row from the stored settings and the pane is drawn from exactly that map, so a revoked client has no row to be listed under; the sighting is dropped with it so nothing stale is left behind.
  evidence: internal/gateway/pairing.go:239 — "delete(cfg.Clients, req.Fingerprint)"
  evidence: internal/gateway/pairing.go:274 — "for _, id := range cfg.ClientIDs() {"
  evidence: internal/gateway/pairing.go:248 — "c.Clients.Forget(req.Fingerprint)"
- ac-14 — MET: The Clients pane draws one row per client carrying the name, the whole fingerprint, the pairing time and the last sighting, each appended with a Revoke button, and a panel test holds the pane's markup and the four fields.
  evidence: internal/ui/static/app.js:603 — "strong.textContent = c.name;"
  evidence: internal/ui/static/app.js:609 — "detail.textContent = `${c.spki} · paired ${clientPaired(c.paired_at)}`"
  evidence: internal/ui/static/app.js:614 — "row.append(confirmBtn('Revoke', 'Revoke?', 'ghost', () => {"
  evidence: internal/ui/clients_test.go:35 — "func TestNoClientFieldIsEverInterpolatedIntoMarkup(t *testing.T) {"
- ac-15 — MET: The whole 44-character fingerprint is drawn beside every name and a panel test refuses a shortened one, so two clients that chose the same name are still told apart by the value the server actually files them under.
  evidence: internal/ui/static/app.js:609 — "detail.textContent = `${c.spki} · paired ${clientPaired(c.paired_at)}`"
  evidence: internal/ui/clients_test.go:60 — "func TestTheWholeFingerprintIsDrawn(t *testing.T) {"
  evidence: internal/config/clients.go:30 — "SPKI string `json:"spki"`"
- ac-16 — MET_WITH_CONCERNS: Every field of a row is set with `textContent` and the pane's only `innerHTML` is the empty-the-box call, which a panel test holds by scanning the render function — so a name that is HTML is drawn as text. The concern is the evidence's shape: the spec promised 'a test that plants a name containing markup and asserts it is drawn as text' and what shipped is a source-shape scan, so nothing exercises the rendering with a hostile name.
  evidence: internal/ui/static/app.js:603 — "strong.textContent = c.name;"
  evidence: internal/ui/clients_test.go:35 — "func TestNoClientFieldIsEverInterpolatedIntoMarkup(t *testing.T) {"
  evidence: internal/ui/clients_test.go:50 — "t.Errorf("the Clients pane builds markup: %q — every field is set with textContent, "+"
  evidence: .abcd/development/specs/closed/spc-2609190031320045-bob-s-chat-client-pairs-with-alice-s-gropius-server-once-and.md:307 — "14-17. The Clients pane — the pane and its rows; a test that plants a name"
- ac-17 — MET: The three-surfaces enumeration walks the clients map into `clients.*.name`, `clients.*.spki` and `clients.*.paired_at`, the exemption table is unchanged at its two long-standing entries, and the whole archtest package passes — so the block is carried by a control rather than waved through.
  evidence: internal/archtest/settings_surface_test.go:78 — "var settingsPaneExemptions = map[string]settingExemption{"
  evidence: internal/archtest/settings_surface_test.go:99 — "func TestEverySettingHasAPanelControlOrAnExemption(t *testing.T) {"
  evidence: internal/ui/static/index.html:249 — "< section id="tab-clients" class="panel">"
  evidence: internal/ui/static/app.js:583 — "function renderClients() {"
- ac-18 — MET: `applySettings` overwrites whatever clients map the posted body carried with a clone of what is stored, before validation and before saving, and a test posts an intruder row and asserts it is neither added nor able to remove the stored one.
  evidence: internal/gateway/control.go:1533 — "incoming.Clients = current.Clone().Clients"
  evidence: internal/gateway/plainport_test.go:118 — "func TestASettingsSaveCannotWriteThePairedSet(t *testing.T) {"
  evidence: internal/gateway/control.go:814 — "c.Clients = nil"
- ac-19 — MET: The TXT record gains `spki=` from a live callback, wired in `cmd/gropius` to the same `Identity.Fingerprint()` the state snapshot serves the panel's Clients pane, and a discovery test asserts the record carries it and stays inside RFC 6763's size target.
  evidence: internal/discovery/discovery.go:150 — "txt["spki"] = fp"
  evidence: cmd/gropius/main.go:459 — "Fingerprint: func() string {"
  evidence: internal/gateway/control.go:478 — "st.ServerFingerprint = c.Identity.Fingerprint()"
  evidence: internal/discovery/pairing_test.go:17 — "func TestTheAdvertisementCarriesTheFingerprintAndTheTLSPort(t *testing.T) {"
- ac-20 — MET: After pairing the client clears any pin, forgets the last handshake, opens a TLS connection of its own and records the fingerprint THAT handshake presented; an architecture test refuses a pin taken from the pairing answer and refuses any mention of the pin in the Bonjour browser.
  evidence: client/GropiusChat/Pairing.swift:306 — "lock.withLock { _lastPresentedSPKI = presented }"
  evidence: client/GropiusChat/GropiusChat.swift:545 — "record.pinnedSPKI = presented"
  evidence: internal/archtest/chat_client_pairing_test.go:97 — "func TestTheClientPinsWhatTheHandshakePresented(t *testing.T) {"
  evidence: internal/archtest/chat_client_pairing_test.go:116 — "if strings.Contains(app, "pinnedSPKI = answer.fingerprint") {"
- ac-21 — MET: The client's Pairing settings show the pinned fingerprint under 'That server's key', which is the value learned from the handshake, and the docs page walks the reader through comparing it against the panel's.
  evidence: client/GropiusChat/GropiusChat.swift:1608 — "LabeledContent("That server's key", value: paired.pinnedSPKI)"
  evidence: docs/pairing.md:35 — "2. Compare **This server's key** with the server fingerprint the client shows"
- ac-22 — MET: Neither redaction path is touched — the control plane still replaces the API key and the HuggingFace token with the placeholder and `gropius config show` still redacts through its unchanged two-entry secret list — and a save that says nothing about clients leaves the stored set exactly as it was, which the settings test asserts directly.
  evidence: internal/gateway/control.go:802 — "c.APIKey = "********""
  evidence: internal/lifecycle/config.go:48 — "{"api_key", func(c config.Config) string { return c.APIKey }, func(c *config.Config, v string) { c.APIKey = v }},"
  evidence: internal/gateway/plainport_test.go:136 — "after = applyOver(t, stored, `{"log_level":"detailed"}`)"
- ac-23 — MET_WITH_CONCERNS: Pairing and revocation are logged at Info with the client's chosen name and the first eight characters of its key FINGERPRINT — never the key — and the per-request line does the same. Concerns: the per-request line is `log.Debug`, and only the `detailed` log level maps to debug, so under the shipped default a request from a paired client is not logged at all; and no test holds any of these three lines.
  evidence: internal/gateway/pairing.go:327 — "log.Debug("paired request", "client", client.Name, "fingerprint", shortFingerprint(spki),"
  evidence: internal/gateway/pairing.go:120 — "c.Log.Info("client paired", "client", answer.Name, "fingerprint", shortFingerprint(answer.clientSPKI))"
  evidence: internal/config/config.go:1311 — "if c.EffectiveLogLevel() == LogLevelDetailed {"
- ac-24 — MET: `LoadIdentity` reads the key file before it would write one, and a test loads twice from one directory and asserts the fingerprint — which is what every client pinned and what the paired-set lookup keys on — is identical across the two loads.
  evidence: internal/pairing/pairing_test.go:20 — "func TestTheServerKeyPersistsAcrossRestartsAndTheLeafIsReissued(t *testing.T) {"
  evidence: internal/pairing/pairing.go:94 — "key, err := loadOrCreateKey(path)"
  evidence: internal/pairing/pairing_test.go:32 — "if first.Fingerprint() != second.Fingerprint() {"
- ac-25 — MET: The leaf is derived at every start and never stored, so the same test that loads twice with a changed address set asserts a different leaf and the same fingerprint — a reissue that costs no client its pairing.
  evidence: internal/pairing/pairing.go:98 — "leafDER, err := selfSign(key, sans)"
  evidence: internal/pairing/pairing_test.go:36 — "if string(first.Leaf().Raw) == string(second.Leaf().Raw) {"
  evidence: internal/pairing/pairing_test.go:20 — "func TestTheServerKeyPersistsAcrossRestartsAndTheLeafIsReissued(t *testing.T) {"
- ac-26 — MET_WITH_CONCERNS: All four 'does not protect against' items are present in the prose — the first-connection impostor pinned permanently with the lock showing and the fingerprint comparison as the only thing that catches it; the pairing window before the list is read; the shared API key still opening the ordinary port; and a compromised device — alongside what pairing does buy. The concern is that the compromise item is written about the DEVICE holding the client key, so the criterion's 'neither end' — a compromised Mac running the server — is not stated.
  evidence: docs/pairing-explained.md:30 — "**The first connection is the one you have to check.** A client learns your"
  evidence: docs/pairing-explained.md:23 — "**Anything on your network can pair.** Your server does not ask you first — that"
  evidence: docs/pairing-explained.md:38 — "**The ordinary port is still there.** Pairing adds a way in; it takes none away."
  evidence: docs/pairing-explained.md:43 — "**A compromised device is compromised.** A key in a device's keychain is"

Gap audit:
- honoured:
  - The client makes a keypair of its own, kept in the Keychain and never sent anywhere, and the server signs it a certificate under the name Bob chose.
    evidence: client/GropiusChat/Pairing.swift:185 — "if let key = SecKeyCreateRandomKey(enclave as CFDictionary, nil) {"
    evidence: internal/pairing/pairing.go:132 — "func (i *Identity) MintLeaf(pub crypto.PublicKey, name string) ([]byte, error) {"
  - From then on the client speaks over TLS on a second port, proving itself with that key on every request, and is never asked for the API key again.
    evidence: internal/gateway/pairing.go:321 — "client, paired := reg.Sight(spki)"
    evidence: client/GropiusChat/GropiusChat.swift:581 — "if let p = paired, let httpsBase = p.httpsBase {"
  - Alice finds a Clients pane listing every client that has paired, and Revoke refuses that client's next request.
    evidence: internal/ui/static/app.js:614 — "row.append(confirmBtn('Revoke', 'Revoke?', 'ghost', () => {"
    evidence: internal/gateway/pairing_test.go:114 — "func TestARevokedClientIsRefusedOnItsNextRequestOverAConnectionItAlreadyHad(t *testing.T) {"
  - The plain port answers exactly as it did — status, headers and body identical with and without a paired client.
    evidence: internal/gateway/plainport_test.go:23 — "func TestThePlainPortDoesNotMoveWhenAClientPairs(t *testing.T) {"
    evidence: internal/gateway/gateway.go:134 — "return g.withAuth(g.routes())"
  - The server keeps its key across restarts and reissues its own certificate after a rename or a new address without a client pairing again.
    evidence: internal/pairing/pairing_test.go:20 — "func TestTheServerKeyPersistsAcrossRestartsAndTheLeafIsReissued(t *testing.T) {"
  - A settings save is not a way to write the list of who may connect, and the clients block never reaches the settings form.
    evidence: internal/gateway/control.go:1533 — "incoming.Clients = current.Clone().Clients"
    evidence: internal/gateway/control.go:814 — "c.Clients = nil"
  - The docs say what pairing is worth, including everything it does not protect against.
    evidence: docs/pairing-explained.md:21 — "## What it does not solve"
    evidence: docs/pairing.md:10 — "What pairing protects against, and what it does not, is on"
- diverged:
  - The end-to-end hand check the intent asks for on the maintainer's own Mac: delivered with a throwaway Swift command-line client rather than the shipping chat app, and the shipping decision line does not say which client it was.
    evidence: .abcd/development/specs/closed/spc-2609190031320045-bob-s-chat-client-pairs-with-alice-s-gropius-server-once-and.md:329 — "came up on both ports, logging its fingerprint. A throwaway Swift client made a"
    evidence: .abcd/work/DECISIONS.md:298 — "Driven end to end against the built server on one Mac: paired, pinned the fingerprint the handshake presented, answered over mutual TLS, survived a restart, revoked."
  - First-come pairing — 'anything on the network that reaches the pairing endpoint pairs itself' — is delivered narrower: a request carrying an Origin header, or any content type but application/json, is refused, so a web page on the network cannot pair.
    evidence: internal/gateway/pairing.go:141 — "if r.Header.Get("Origin") != "" {"
    evidence: internal/gateway/pairing.go:138 — "if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {"
  - 'Nothing else changes' on the plain port: its existing behaviour is unchanged, but the plain listener now carries one new route that asks for no credential, POST /pair.
    evidence: cmd/gropius/main.go:415 — "mux.Handle("/pair", ctrl.PairHandler())"
  - The decision line says the client falls back to a permanent Keychain key 'on errSecMissingEntitlement'; the code discards the CFError and falls back on ANY failure of the Enclave attempt, so a different Enclave failure is also silently downgraded.
    evidence: client/GropiusChat/Pairing.swift:185 — "if let key = SecKeyCreateRandomKey(enclave as CFDictionary, nil) {"
    evidence: .abcd/work/DECISIONS.md:292 — "So the client attempts the Enclave and falls back on `errSecMissingEntitlement`, one path and two attempt dictionaries"
  - 'A lock sits beside the server's name in the picker': the pairing lock is drawn only on the stored-server row, and that row is shown only while the client is connected; a discovered row for the same server carries the API-key glyph and no pairing lock.
    evidence: client/GropiusChat/Picker.swift:94 — "Image(systemName: model.pairedHere ? "lock.shield.fill" : "network")"
    evidence: client/GropiusChat/Picker.swift:51 — "if !model.serverURL.isEmpty, expanded == nil, model.connected {"
  - The spec promised a test that plants a name containing markup and asserts it is drawn as text; what shipped is a source scan of the render function for textContent and innerHTML.
    evidence: internal/ui/clients_test.go:35 — "func TestNoClientFieldIsEverInterpolatedIntoMarkup(t *testing.T) {"
    evidence: .abcd/development/specs/closed/spc-2609190031320045-bob-s-chat-client-pairs-with-alice-s-gropius-server-once-and.md:307 — "14-17. The Clients pane — the pane and its rows; a test that plants a name"
- missing:
  - The iPad hand check the intent promises — 'checked by hand on the maintainer's device and recorded as such' — was not done; the record states it as owed.
    evidence: .abcd/work/DECISIONS.md:298 — "**OWED by hand, and not verified here:** a real iPad; the Local Network prompt on a device whose grant has been reset; a second Mac on the LAN; the Secure Enclave"
  - The owed hand check on how the lock renders is recorded nowhere: the shipping line's owed list names the iPad, the Local Network prompt, a second Mac and the Enclave, and not the picker's lock.
    evidence: .abcd/work/DECISIONS.md:298 — "**OWED by hand, and not verified here:** a real iPad; the Local Network prompt on a device whose grant has been reset; a second Mac on the LAN; the Secure Enclave"
    evidence: .abcd/development/specs/closed/spc-2609190031320045-bob-s-chat-client-pairs-with-alice-s-gropius-server-once-and.md:343 — "What stays owed by hand, and cannot be shown here: a real iPad; the Local"
  - No test holds the pairing, revocation or per-request log lines to naming the client by its pairing name, so the criterion rests on the code alone.
    evidence: internal/gateway/pairing.go:327 — "log.Debug("paired request", "client", client.Name, "fingerprint", shortFingerprint(spki),"
  - No pairing-endpoint case exercises an oversized request body, which is one of the three bounds the criterion names.
    evidence: internal/gateway/pairing_test.go:198 — "func TestThePairingEndpointRefusesWhatItCannotStore(t *testing.T) {"
    evidence: internal/gateway/pairing.go:145 — "if err := json.NewDecoder(io.LimitReader(r.Body, maxPairBodyBytes)).Decode(&req); err != nil {"
  - Nothing exercises `gropius config show` over a populated clients block, so the terminal surface's report of the paired set is derived and untested.
    evidence: internal/lifecycle/config.go:113 — "if fv.Type().Elem().Kind() == reflect.Struct {"
    evidence: internal/lifecycle/testdata/config_show.json:45 — ""tls_port": 0,"

Scope-condition dispositions:
- cond-2609190031321992 — survived: Pairing.swift carries no platform conditional at all and both build scripts glob the same GropiusChat/*.swift, so the Mac and iPad clients are one source; Swift gained a pairing file and nothing on the server side.
  evidence: client/build-ipad.sh:76 — "GropiusChat/*.swift"
  evidence: client/build.sh:34 — "GropiusChat/*.swift"
  evidence: client/GropiusChat/Pairing.swift:89 — "enum PairingStore {"
- cond-2609190031325339 — survived: makeKey holds one code path with two attempt dictionaries and chooses between them at runtime on the Keychain's own answer, and an architecture test refuses a single SecKeyCreateRandomKey call so the Enclave cannot quietly become an assumption.
  evidence: client/GropiusChat/Pairing.swift:185 — "if let key = SecKeyCreateRandomKey(enclave as CFDictionary, nil) {"
  evidence: client/GropiusChat/Pairing.swift:197 — "return SecKeyCreateRandomKey(ordinary as CFDictionary, nil)"
  evidence: internal/archtest/chat_client_pairing_test.go:68 — "func TestTheClientAttemptsTheEnclaveAndFallsBack(t *testing.T) {"
- cond-2609190031322606 — narrowed: Pairing is still first-come with no code and no click, but the endpoint now refuses two classes of caller the condition's 'anything on the network' covered, as a deliberate answer to iss-2609190110118690.
  narrowing: Holds for a native caller that posts application/json and sends no Origin header; a request carrying an Origin, or any other content type, is refused — so a web page anyone on the network visits can no longer enrol a key, which the condition as written did not carve out.
  evidence: internal/gateway/pairing.go:141 — "if r.Header.Get("Origin") != "" {"
  evidence: internal/gateway/pairing.go:138 — "if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {"
  evidence: internal/gateway/pairing_race_test.go:114 — "func TestThePairingEndpointIsNotAFormAPageCanPost(t *testing.T) {"
- cond-2609190031320071 — survived: Nothing was added to close the window: there is no approval step on the panel and no displayed code anywhere, and the docs page states the exposure and names reading the Clients list as the only thing that reveals it.
  evidence: docs/pairing-explained.md:23 — "**Anything on your network can pair.** Your server does not ask you first — that"
  evidence: internal/ui/static/index.html:250 — "< h2>Chat clients paired with this server< /h2>"
- cond-2609190031327891 — survived: The paired-set lookup runs at the start of every request, so a revocation takes effect on the next request down a connection already open, and nothing anywhere cancels a response already being written.
  evidence: internal/gateway/pairing.go:321 — "client, paired := reg.Sight(spki)"
  evidence: internal/gateway/pairing_test.go:114 — "func TestARevokedClientIsRefusedOnItsNextRequestOverAConnectionItAlreadyHad(t *testing.T) {"
- cond-2609190031321181 — narrowed: The bearer check, the loopback exemption and the refusal text are untouched and the replay proves the plain port's answers do not move, but the plain listener is no longer only what it was: it now also serves the pairing route.
  narrowing: Holds for the gateway's own routes and the API-key rule, which are byte for byte unchanged; it does not hold for the plain listener's route set, which gains POST /pair — an endpoint that asks for no credential and writes the settings file.
  evidence: internal/gateway/gateway.go:134 — "return g.withAuth(g.routes())"
  evidence: internal/gateway/plainport_test.go:23 — "func TestThePlainPortDoesNotMoveWhenAClientPairs(t *testing.T) {"
  evidence: cmd/gropius/main.go:415 — "mux.Handle("/pair", ctrl.PairHandler())"
- cond-2609190031322420 — survived: The TLS listeners are taken from the same plan object on its own port, and they are acquired inside runServer well after secureExposedBind has had its say, so a bind narrowed to this Mac narrows the TLS set with it rather than leaving a second socket behind the lockdown's back.
  evidence: cmd/gropius/tlsbind.go:42 — "for _, addr := range plan.Addrs(port) {"
  evidence: cmd/gropius/main.go:195 — "lns, plan, saved := secureExposedBind(paths, &cfg, lns, plan, log)"
  evidence: cmd/gropius/tlsbind_test.go:18 — "func TestTheTLSBindIsThePlansAddressesOnItsOwnPort(t *testing.T) {"
- cond-2609190031322346 — survived: The name, fingerprint and pairing time are a settings field in config.json reached from the panel's Clients pane and covered by the three-surfaces enumeration, while the last sighting lives only in the registry's in-memory map and reaches the panel through the state snapshot.
  evidence: internal/config/config.go:714 — "Clients map[string]Client `json:"clients,omitempty"`"
  evidence: internal/pairing/registry.go:26 — "seen map[string]time.Time"
  evidence: internal/gateway/control.go:475 — "Clients: pairedClients(cfg, c.Clients),"
- cond-2609190031325817 — survived: No writing verb was added, and the settings walk recurses a map whose element is a struct, so the clients block is reported per field the way the per-model settings are, with the fingerprint unredacted because the secret list is unchanged at two entries.
  evidence: internal/lifecycle/config.go:113 — "if fv.Type().Elem().Kind() == reflect.Struct {"
  evidence: internal/lifecycle/config.go:48 — "{"api_key", func(c config.Config) string { return c.APIKey }, func(c *config.Config, v string) { c.APIKey = v }},"
  evidence: internal/gateway/pairing.go:220 — "func (c *Control) handleRevoke(w http.ResponseWriter, r *http.Request) {"
- cond-2609190031325530 — survived: The new block is an omitempty field with no migration anywhere; a row an older or hand-edited file carries that this build will not use is dropped with a notice rather than converted, which is the pre-1.0 treatment and not a compatibility shim.
  evidence: internal/config/config.go:714 — "Clients map[string]Client `json:"clients,omitempty"`"
  evidence: internal/config/clients.go:163 — "func (c *Config) sanitizeClients() []string {"
  evidence: internal/config/clients_test.go:20 — "func TestAMalformedClientIsDroppedWithANoticeAndNeverRefuses(t *testing.T) {"
- cond-2609190031322324 — survived: What shipped for the client is ten build-time source assertions over the Swift files — the delegate methods, the Enclave-or-fallback attempt pair, the pin's provenance and the absence of a plain-HTTP fallback — and the file's own header states that it asserts shape and never behaviour.
  evidence: internal/archtest/chat_client_pairing_test.go:10 — "// The client has no test target, so what is assertable here is the SHAPE — that"
  evidence: internal/archtest/chat_client_pairing_test.go:21 — "func TestTheClientAnswersBothTLSChallenges(t *testing.T) {"
  evidence: internal/archtest/chat_client_pairing_test.go:169 — "func TestThePinIsNotInThePreferences(t *testing.T) {"
## Grounds

- pursued: we expect a per-client keypair proved by mutual TLS on a second port to give Alice a revocable identity per device, because every piece is system-provided — Go's stdlib mTLS on one side and URLSession's client-certificate challenge on the other — so the risk is whether two Apple pieces meet rather than the volume of code; wrong if a Keychain identity cannot be presented through URLSession at all, or if the pinning delegate cannot accept a self-signed leaf without loosening App Transport Security
