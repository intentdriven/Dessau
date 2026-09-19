# Prefer SOTA, adversary-filtered

## The rule

> Prefer SOTA, adversary-filtered: where a choice has a credible
> state-of-the-art answer that is the presumptive pick, but challenge it for
> fit against this repo's preferences before adopting — see
> .abcd/development/principles/prefer-sota.md.

## Why

The repository's record shows both halves of the rule doing work, and the
second half overturning the first as often as it confirms it.

`adr-2609061503319212` (no public telemetry) rests on two research passes,
`.abcd/development/research/notes/2026-09-06-local-telemetry-sota.md` and
`2026-09-06-anonymous-telemetry-sota.md`. The state of the art there is
anonymous opt-in telemetry in the style of the Go toolchain, and the ADR's
Alternatives Considered rejects it on fit: the population is too small for the
numbers to mean anything, and for a local-LLM tool the mere existence of an
upload path is brand-negative even when technically clean. The presumptive pick
lost to a stated preference, and the record says which preference.

`adr-2609182357322050` (paired clients over pinned TLS) runs the same shape in
the other direction, on
`.abcd/development/research/notes/2026-09-19-pairing-pinned-tls-sota.md`. The
request was for a bespoke application-layer scheme keyed on a unique hash; the
ADR takes the sanctioned shape instead — TLS for transit, a keypair per client
— and then bends it to fit: no certificate authority, the client pins the
server's own certificate, and the keypair is made through the Security
framework rather than CryptoKit because that is what reaches a Secure Enclave.

The decision line of 2026-09-17 in `.abcd/work/DECISIONS.md` records the client
lane running the whole pattern as a staffed pass: three SOTA researchers, two
adversarial reviewers, and builds on the build Mac, with the facts that changed
the record listed one by one.

Records drawn on: `adr-2609061503319212`, `adr-2609182357322050`, the
`.abcd/work/DECISIONS.md` lines of 2026-09-10 and 2026-09-17.

## How to apply here

- A choice at a trust boundary (`internal/gateway`, `internal/hub`,
  `internal/runtime`, `internal/config`, `internal/capability`) takes the
  sanctioned platform shape as its starting point, and an ADR states where it
  is bent and why.
- The filter this repository applies is its own stated preferences: nothing
  leaves the Mac to the project or a third party, no new outbound host, and a
  new dependency needs the maintainer's sign-off. The 2026-09-10 line refuses a
  CDN-fetched asset for the control panel on exactly those grounds.
- A SOTA pass lands as a dated note under
  `.abcd/development/research/notes/` with its evidence tiers marked, and the
  decision it feeds cites it.
