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
resolution: "Every Hub URL is now built by one helper, hubURL, which escapes each part exactly once: the repo id and a file path keep their '/' separators, a revision occupies one element. Search, RepoInfo, the tree listing and ResolveURL all go through it, and a repo id carrying a '.' or '..' segment — which escaping cannot make safe — is refused once at the edge by checkRepoID."
impact: fix
---

The hub client escapes a repo id into the URL on one path and interpolates it raw on another. RepoInfo builds its URL with escapePathSegments; files() builds the tree URL with fmt.Sprintf and the bare repoID, and ResolveURL escapes the file path and revision but not the repo id either. No origin bypass follows, because the host prefix is fixed, but a repo id carrying '?', '#' or '..' reaches a different Hub endpoint than the caller named, and the two paths in one package should escape the same way. Escape the repo id everywhere it enters a URL, or validate it once at the edge and say so.

## Grounds

- pursued: we expect one escaper to remove the class of bug where two paths in one package disagree, because the raw-interpolation path was the one that got it wrong; wrong if a Hub repo id legitimately contains a '.' or '..' path segment, which would now be refused.
