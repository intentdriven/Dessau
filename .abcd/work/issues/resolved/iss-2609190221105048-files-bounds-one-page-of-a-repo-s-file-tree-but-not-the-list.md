---
schema_version: 1
id: "iss-2609190221105048"
slug: "files-bounds-one-page-of-a-repo-s-file-tree-but-not-the-list"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "the adversarial read of the download-bounds change, which bounded the file body and raised the same question of the listing"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/hub/hub.go"
resolution: "Files now bounds the whole listing, not one page: a byte budget across every page (maxTreeBytes, one page's worth) and a cap on the entries held (maxTreeEntries, 100000), both refused with the package's ErrOversizedBody. An ordinary multi-page listing is still followed to its end."
impact: fix
---

Files bounds one page of a repo's file tree but not the listing. Each page is decoded under maxJSONBody (32 MiB) and up to maxTreePages (1000) pages are followed, and every page's entries are appended to one slice, so a Hub — or anything that has taken its place before the origin rule is reached — can make a single file listing allocate far more memory than any real repo needs, before a byte of the model is downloaded. The per-page cap reads as though it bounds the listing; it bounds a page. A cap on the total entry count, or on the bytes decoded across all pages, would bound what one listing can cost. internal/hub/hub.go, in files().

## Grounds

- pursued: we expect a whole-listing budget to hold every real MLX repo well inside it because the most heavily sharded model names hundreds of files and a few hundred KB of tree, while the old per-page-times-page-count product allowed 32 GiB; wrong if a legitimate repo ever lists more than a hundred thousand files or more than 32 MiB of tree, which would show as a refusal naming ErrOversizedBody on a real download.
