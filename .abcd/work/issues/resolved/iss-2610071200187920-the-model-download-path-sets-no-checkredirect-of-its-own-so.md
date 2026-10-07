---
schema_version: 1
id: "iss-2610071200187920"
slug: "the-model-download-path-sets-no-checkredirect-of-its-own-so"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "security review of the LFS update-check fix (iss-2610042101439623), 2026-10-07"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/hub/download.go"
remedy: "Give the download client the same redirect policy as doContent, one shared primitive rather than a second copy: strip Authorization off-origin and keep the 10-hop limit, with a download test using a same-host, other-port CDN."
resolution: "downloads now fetch through doContent, sharing its one redirect rule: token stays on the Hub's origin, https-to-http refused, 10 hops"
impact: fix
---

The model download path sets no CheckRedirect of its own, so it relies on net/http's default header stripping, which keeps Authorization on a redirect to a subdomain of the Hub's host. The update check's newer content fetch (doContent) drops the token on any hop that leaves the Hub's origin, so the path that moves gigabytes is the less strict of the two. Today's CDN hosts are on another domain, so net/http drops the token anyway; the gap opens only if the Hub redirects to one of its own subdomains.
