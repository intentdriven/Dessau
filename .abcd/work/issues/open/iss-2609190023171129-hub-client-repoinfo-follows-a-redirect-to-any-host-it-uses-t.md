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
---

hub.Client.RepoInfo follows a redirect to any host: it uses the shared http.Client, which follows up to ten redirects, and unlike Files — which refuses a cross-origin rel=next page explicitly, so the access token is never sent elsewhere — RepoInfo has no same-origin check. Go's net/http does strip the Authorization header on a redirect to another domain, so the token does not leak, but the decoded answer about what a repo is can then come from a host that is not the Hub. The two request paths in this package should answer the same way about where a response may come from.
