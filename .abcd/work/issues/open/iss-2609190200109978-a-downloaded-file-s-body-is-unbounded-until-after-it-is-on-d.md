---
schema_version: 1
id: "iss-2609190200109978"
slug: "a-downloaded-file-s-body-is-unbounded-until-after-it-is-on-d"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "the adversarial security review of the same-origin rule added for iss-2609190023171129"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/hub/download.go"
---

A downloaded file's body is unbounded until after it is on disk. downloadFile io.Copy's the response into the .part and only then compares the length to the size the tree declared, so a tree that says 10 bytes against a body that streams 32 MiB writes all 32 MiB before rejecting it, at Concurrency files at once; and a connection reset mid-stream deliberately keeps the oversized .part for resume. Peak disk is whatever answers the request rather than what the repo said it would be. An io.LimitReader at the declared size (or a generous multiple where the size is unknown) would cost nothing and turn this into a read error at the boundary.
