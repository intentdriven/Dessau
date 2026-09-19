---
schema_version: 1
id: "iss-2609190023171129"
slug: "hub-client-repoinfo-follows-a-redirect-to-any-host-it-uses-t"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "the adversarial read of the exact-repo-id search lookup, which is a new caller of RepoInfo"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/hub/hub.go"
resolution: "The hub client now has one seam, Client.do, that refuses a response whose final URL after redirects is not same-origin with the Hub's base URL; Search, RepoInfo and each page of Files go through it, and both that refusal and the pre-existing cross-origin rel=next refusal in Files wrap a single ErrCrossOrigin sentinel, so every request path answers the same way and with the same error class. The file-download path is the one documented exception: the Hub answers a /resolve/ GET with a redirect to its content CDN by design, and those bytes are anchored by the sha256 the Hub's own API stated, which downloadFile verifies."
impact: fix
---

hub.Client.RepoInfo follows a redirect to any host: it uses the shared http.Client, which follows up to ten redirects, and unlike Files — which refuses a cross-origin rel=next page explicitly, so the access token is never sent elsewhere — RepoInfo has no same-origin check. Go's net/http does strip the Authorization header on a redirect to another domain, so the token does not leak, but the decoded answer about what a repo is can then come from a host that is not the Hub. The two request paths in this package should answer the same way about where a response may come from.

## Grounds

- pursued: we expect one same-origin rule shared by every request path in internal/hub, because two request paths in one package answering differently about where a response may come from is how a gap survives review; wrong if the Hub ever redirects an API endpoint off huggingface.co legitimately, which would turn the refusal into a false negative on search and repo lookups.
