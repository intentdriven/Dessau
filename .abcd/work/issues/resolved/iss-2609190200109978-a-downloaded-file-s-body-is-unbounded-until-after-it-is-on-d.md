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
resolution: "A download body is now held to the smaller of the response's Content-Length and what the repo's file tree says is outstanding for the file, by a boundedReader that refuses the first byte past the boundary rather than after the body is on disk; a body that ends short of a declared length is refused too, and neither leaves a .part behind. A file neither the tree nor the response sized gets a 1 GiB ceiling instead of no bound at all."
impact: fix
---

A downloaded file's body is unbounded until after it is on disk. downloadFile io.Copy's the response into the .part and only then compares the length to the size the tree declared, so a tree that says 10 bytes against a body that streams 32 MiB writes all 32 MiB before rejecting it, at Concurrency files at once; and a connection reset mid-stream deliberately keeps the oversized .part for resume. Peak disk is whatever answers the request rather than what the repo said it would be. An io.LimitReader at the declared size (or a generous multiple where the size is unknown) would cost nothing and turn this into a read error at the boundary.

## Grounds

- pursued: we expect peak disk during a download to be what the repo said it would be, because the length is stated before the body arrives and the boundary is where a disagreement is cheapest to refuse; wrong if the Hub legitimately serves a file larger than the size its own tree states, which would now fail instead of overshooting.
