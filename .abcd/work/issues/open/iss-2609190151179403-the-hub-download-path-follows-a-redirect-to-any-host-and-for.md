---
schema_version: 1
id: "iss-2609190151179403"
slug: "the-hub-download-path-follows-a-redirect-to-any-host-and-for"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "the hostile read of the diff that gave internal/hub one same-origin rule for every API request path (iss-2609190023171129)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/hub/download.go"
---

The hub download path follows a redirect to any host, and for a file the tree lists without an LFS object there is no hash to check what arrives. downloadFile verifies sha256 only when f.LFS.OID is set (internal/hub/download.go), which covers the weights; config.json, tokenizer.json and any .py in the repo are small non-LFS entries whose bytes are accepted on size alone. The API paths now refuse an off-origin answer (Client.do), and the download path is the documented exception because the Hub redirects an LFS object to its content CDN by design — but the exception is currently wider than its justification: it also lets a hostile or compromised Hub response redirect an unhashed small file to another host. The narrow rule would be to allow an off-origin file response only when the tree gave a content hash for that file; it is not done here because it rests on an unverified claim about when the real Hub redirects, and getting it wrong breaks downloads outright.

## Deferral 2026-09-19

Waits on the maintainer: narrowing the download exception to LFS objects rests on an unverified claim about when the real Hub redirects a non-LFS file, and being wrong breaks every download outright; PR 114 bounds the body and PR 107 pins the origin of every listing, so what remains is a policy call on what a git-stored file must be verified against.
