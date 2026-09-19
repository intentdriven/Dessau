---
schema_version: 1
id: "iss-2609190200162002"
slug: "the-hub-client-escapes-a-repo-id-into-the-url-on-one-path-an"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "the adversarial security review of the same-origin rule added for iss-2609190023171129"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/hub/hub.go"
---

The hub client escapes a repo id into the URL on one path and interpolates it raw on another. RepoInfo builds its URL with escapePathSegments; files() builds the tree URL with fmt.Sprintf and the bare repoID, and ResolveURL escapes the file path and revision but not the repo id either. No origin bypass follows, because the host prefix is fixed, but a repo id carrying '?', '#' or '..' reaches a different Hub endpoint than the caller named, and the two paths in one package should escape the same way. Escape the repo id everywhere it enters a URL, or validate it once at the edge and say so.
