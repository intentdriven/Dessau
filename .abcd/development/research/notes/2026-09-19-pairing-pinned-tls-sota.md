# SOTA — pairing, pinned TLS and mutual TLS for a LAN server (2026-09-19)

Research for itd-2609182357325215 and adr-2609182357322050, run by an Opus 5
research agent from primary sources (Apple DTS forum answers by Quinn, Apple
security and Network documentation, TN2232, RFC 7469, RFC 6763, the Go crypto/tls
docs, Syncthing, HomeKit and Tailscale design notes, Ollama and llama.cpp server
docs). Ranked; what could not be verified is at the end.

1. **The server mints the client's certificate.** No public Apple API mints a
   certificate; a `SecIdentity` comes only from `SecPKCS12Import` or from the
   Keychain matching a certificate to a private key already there. So the
   client makes its key, sends the raw public key over the pairing call, the
   server signs a leaf with `x509.CreateCertificate`, and the client stores the
   DER, which forms the identity. No CSR: proof of possession is the mTLS
   handshake itself.
2. **Make the Secure Enclave key with the Security framework**
   (`SecKeyCreateRandomKey`, `kSecAttrTokenIDSecureEnclave`, permanent), not
   CryptoKit, whose Enclave key stores as a blob and can never become an
   identity. Present it with `URLCredential(identity:…)` on the client
   certificate challenge. Apple says an Enclave-backed identity *should* work
   with URLSession — spike this first; a non-Enclave Keychain key is the
   fallback.
3. **URLSession, not the Network framework,** for the client transport.
4. **Pin the SPKI** (`base64(SHA-256(DER SubjectPublicKeyInfo))`, RFC 7469), in
   Go `sha256.Sum256(cert.RawSubjectPublicKeyInfo)`. Persist the server KEY
   across restarts; the leaf may be reissued freely (new SANs, validity) and
   no client re-pairs. Regenerating the key is what breaks pairing.
5. **`SecKeyCopyExternalRepresentation` is not SPKI** — it returns the X9.63
   point; prepend the fixed 26-byte P-256 SPKI header before hashing.
6. **ATS pinning cannot loosen trust**, so `NSPinnedDomains` cannot accept a
   self-signed leaf; the delegate route (TN2232) is the sanctioned one, and the
   SPKI comparison — not the trust evaluation, whose result the system
   ignores — is the gate.
7. **Go side is stdlib, about forty lines:** `ClientAuth: RequireAnyClientCert`
   plus `VerifyPeerCertificate` parsing `rawCerts[0]`, hashing its SPKI and
   looking it up in the paired set; a second `http.Server` with its own
   `tls.Config` over the same handler.
8. **TXT size is fine** (a `spki=` key is 49 bytes; RFC 6763 targets 200), but a
   TXT change re-registers the advertisement — one more reason not to rotate
   the key.
9. **Pairing UX — the research sides with approve-on-panel over a typed code,
   and against PAKE** (an unaudited Go dependency for a threat panel approval
   closes). The maintainer chose first-come-then-revoke (2026-09-19); the
   research's option stays recorded in the ADR as not chosen, not closed.
10. **Plain HTTP beside HTTPS is what every peer does.** Ollama is HTTP-only;
    stock llama-server is HTTP; the official OpenAI clients need a custom CA
    file (`httpx.Client(verify=…)`, `NODE_EXTRA_CA_CERTS`) for a self-signed
    endpoint. That friction is the argument for keeping the plain port.
11. **Local Network privacy:** registering with Bonjour prompts; listening does
    not. Both apps already carry the usage description.

## Not adopted
SPAKE2/SRP pairing; Network framework for transport; ATS declarative pinning; a
private CA with chain verification; scheduled server-certificate rotation.

## Could not be verified
- Any macOS 27 / iPadOS 27 change to URLSession, Network framework TLS,
  CryptoKit or Local Network privacy — only WWDC25 material was found.
- A first-hand report of an Enclave-backed identity completing an mTLS
  handshake through URLSession. Spike before the design hardens.
