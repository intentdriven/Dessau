---
id: itd-2609182357325215
slug: bob-s-chat-client-pairs-with-alice-s-gropius-server-once-and
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609151701196720, itd-2609170718430553]
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# Bob's chat client pairs with Alice's server once

## Press Release

> _Seeded from a quoted-text intent capture. Expand into the full press-release narrative before planning._

## Why This Matters

Bob's chat client pairs with Alice's Gropius server once, and from then on the server knows it by name: the client makes a keypair of its own, kept in the Mac's Keychain, and pairs it with the server over the local network; Alice sees every paired client on the panel — its name, when it paired, when it was last seen — and can revoke one with a click. Once paired, everything between that client and the server travels encrypted, and the client shows a lock beside the server's name. An unpaired OpenAI client — curl, a Python script — keeps working on the LAN as it does today.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

- The transport and the keys are decided in an ADR minted beside this
  draft (TLS on the server with a certificate the client pins at pairing;
  a keypair per client, made in its Keychain; the shared API key stays for
  unpaired OpenAI clients). Ratify it before planning.
- Resolved 2026-09-19 (the maintainer): a pairing is first-come, then
  revoke — no code, no click; the docs say what that leaves open.
- Resolved in the ADR: a paired client's identity replaces the API key for
  that client, which is never typed on it; the certificate's SPKI fingerprint
  travels in the Bonjour TXT record and is shown on the panel.
- Open, for the SOTA note: which client-side TLS stack presents a Keychain
  identity cleanly on both platforms, and whether the server issues the
  client's certificate at pairing (a client cannot mint its own without a
  private API).

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
