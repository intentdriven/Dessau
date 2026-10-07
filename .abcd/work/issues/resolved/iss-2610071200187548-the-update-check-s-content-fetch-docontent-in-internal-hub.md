---
schema_version: 1
id: "iss-2610071200187548"
slug: "the-update-check-s-content-fetch-docontent-in-internal-hub"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "security review of the LFS update-check fix (iss-2610042101439623), 2026-10-07"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/hub/hub.go"
remedy: "In doContent's CheckRedirect, drop Authorization for the rest of the chain once any hop in via is off-origin, and refuse a non-https hop when the base URL is https; add hub tests for leave-then-return and for an https-to-http redirect."
resolution: "doContent's redirect rule drops the token for the rest of the chain once any hop is off-origin and refuses https-to-http"
impact: fix
---

The update check's content fetch (doContent in internal/hub/hub.go) strips the access token only on the hop that leaves the Hub's origin, so a redirect chain that leaves and then returns sends the token to the Hub again. A CDN on a host net/http treats as the same domain, or the same host on another port, can redirect back to the Hub origin, and the next request is rebuilt from the original headers. The reviewer's probe showed the CDN hop receiving no token and the hop back receiving the bearer token. The token only reaches its own audience and the bytes are still checked against the listed sha256, but a hostile CDN can choose one authenticated Hub GET, which refuseOffOrigin exists to prevent. The same fetch also follows an https-to-http hop: integrity holds through the hash and the token is dropped, but a gated repo's config.json crosses the network unencrypted.
