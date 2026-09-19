---
schema_version: 1
id: "iss-2609181048176454"
slug: "the-server-s-model-search-does-not-find-a-model-that-exists"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "the maintainer's first look at the 0.7.0 server, 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/hub"
resolution: "The picker now also looks a query up exactly when it is a well-formed repository id, and offers the repo whatever account owns it, alongside the author-scoped search; the author: prefix and the typed-id route are both stated in the search box's help text and in the getting-started page."
impact: fix
---

The server's model search does not find a model that exists on Hugging Face and is MLX: the maintainer searched for the repo prism-ml/Ternary-Bonsai-2-27B-mlx-2bit in the panel's picker and it was not offered (2026-09-18). To establish: what the search sends to the Hub (a filter on the mlx library tag, an author, a pipeline tag, or a result cap) and whether this repo's card carries the tags the filter expects; and whether the picker should also accept a repo id typed exactly, so a model the search cannot surface can still be downloaded.

## Diagnosis (2026-09-18, the session that captured it)

The panel's search restricts every query to one Hub author, `mlx-community`,
unless the query begins with `author:` (`internal/gateway/control.go`,
`searchAuthor`). A repository under another account, such as
`prism-ml/Ternary-Bonsai-2-27B-mlx-2bit`, is found only by typing
`author:prism-ml Ternary`. Whether that prefix is discoverable from the
panel or the docs decides whether this is a documentation gap or a search
that should also try the typed text as a full repo id.

## Grounds

- pursued: we expect a model typed in full to be offered because the exact lookup asks the Hub's model-info endpoint for that id directly, whatever its author; wrong if the repo is gated, carries neither the mlx library tag nor an mlx marker in its name, or does not fit this Mac — in those cases it is still not offered.
