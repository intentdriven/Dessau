---
schema_version: 1
id: "iss-2609181213199613"
slug: "the-thoughts-row-s-own-textselection-was-removed-and-selecti"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "fidelity audits 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/GropiusChat.swift"
resolution: "The reasoningDisclosure doc comment in client/GropiusChat/GropiusChat.swift no longer promises that dragging selects the text: it now says selection is per block — each block the thinking is drawn as carries its own textSelection and the row carries none, so a drag selects within one block and does not run across two. Comment only; no behaviour changed."
impact: internal
---

The Thoughts row's own textSelection was removed and selection is now per block, so a drag across two thought blocks is no longer one selection, while the row's doc comment still promises that dragging selects the text.

## Grounds

- pursued: we expect the comment to match the code because blockViews attaches textSelection(.enabled) per block and reasoningDisclosure attaches none; wrong if a row-level textSelection is put back, which would make a cross-block drag one selection again
