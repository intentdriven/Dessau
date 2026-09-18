---
id: itd-2609181104498312
slug: bob-finds-a-conversation-from-a-search-bar-at-the-top-of-the
spec_id: spc-2609181104491592
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609151701196720]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# Bob finds a conversation from a search bar in the sidebar

## Press Release

Bob finds a conversation from a search bar at the top of the sidebar, like
Messages': typing filters the list live to the conversations whose title or
messages contain the words, and clearing the search brings every
conversation back.

## Why This Matters

A sidebar of many chats is only useful if the one from last week can be
found by what was said in it.

## Mechanism

We expect the search to be the system's because SwiftUI's searchable
modifier on the sidebar places the field where the platform puts it and
hands the client the text; filtering is one case-insensitive contains over
titles and messages. What would show this wrong: a search field the sidebar
column cannot host at its minimum width.

## Scope Conditions

- The search is over titles and message text, case-insensitive, live on <!-- cond: cond-2609181104490024 -->
  every keystroke; it is the system's searchable field, not a field of the
  client's own.
- An empty search shows every conversation; the selection is kept if the <!-- cond: cond-2609181104498637 -->
  selected conversation still matches.

## Acceptance Criteria

- Given several conversations, when Bob types a word that appears in one
  conversation's messages, then only that conversation is listed; when he
  clears the field, then all are listed again.
- Given the client's source, when the architecture tests run, then the
  sidebar carries the searchable modifier and the filter reads titles and
  messages.

## Open Questions

_None recorded yet._

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-513e379573b4 -->
Fidelity review — receipt rcp-513e379573b4 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:174538bc6f69cc23419391fe849d285500956eefefebcd76ad51051c9812ba15
Input attestations: diff:c2bd0ed..6534be1 -- client internal/archtest (PR 66)@sha256:8360f15991fd63f4f790a03f2425c4a91bb7454074a488ee8fa21a79201551b4;

Acceptance rollup: MET 1 · MET_WITH_CONCERNS 1 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: The filter is literal in source — the List renders `shown`, which returns every conversation on an empty query and otherwise those whose title or any message contains the query — but nothing ran it: the Swift client has no test target and .abcd/work/DECISIONS.md:284 records only an architecture test for this intent, no hand-check of typing and clearing.
  evidence: client/GropiusChat/GropiusChat.swift:877 — "guard !q.isEmpty else { return model.conversations }"
  evidence: client/GropiusChat/GropiusChat.swift:879 — "c.title.localizedCaseInsensitiveContains(q)"
  evidence: client/GropiusChat/GropiusChat.swift:886 — "ForEach(shown) { c in"
  evidence: .abcd/work/DECISIONS.md:284 — "Reviews scaled to the blast radius: one architecture test each."
- ac-2 — MET: `go test ./internal/archtest/` was run here and TestChatClientSidebarIsSearchable PASSes; it asserts the searchable modifier and both filter fields, and the source carries the modifier on the sidebar List itself.
  evidence: internal/archtest/chat_client_composer_test.go:182 — "if !strings.Contains(src, `.searchable(text: $query`) {"
  evidence: internal/archtest/chat_client_composer_test.go:185 — "title\.localizedCaseInsensitiveContains\(q\)"
  evidence: client/GropiusChat/GropiusChat.swift:905 — ".searchable(text: $query, placement: .sidebar, prompt: "Search")"

Gap audit:
- honoured:
  - The search is the system's searchable field placed on the sidebar, not a field of the client's own
    evidence: client/GropiusChat/GropiusChat.swift:905 — ".searchable(text: $query, placement: .sidebar, prompt: "Search")"
  - Filtering is one case-insensitive contains over titles and message text
    evidence: client/GropiusChat/GropiusChat.swift:878 — "return model.conversations.filter { c in"
    evidence: client/GropiusChat/GropiusChat.swift:880 — "|| c.messages.contains { $0.text.localizedCaseInsensitiveContains(q) }"
  - Clearing the search brings every conversation back
    evidence: client/GropiusChat/GropiusChat.swift:876 — "let q = query.trimmingCharacters(in: .whitespaces)"
    evidence: client/GropiusChat/GropiusChat.swift:877 — "guard !q.isEmpty else { return model.conversations }"
  - An architecture test holds the sidebar to the searchable modifier and the two filter fields, and it passes
    evidence: internal/archtest/chat_client_composer_test.go:179 — "func TestChatClientSidebarIsSearchable(t *testing.T) {"
  - The user-facing client README states the search bar and what it filters on
    evidence: client/README.md:80 — "the date and how many exchanges and words it holds — with a search bar above"
- diverged:
  - The press release promises the list filters to conversations whose title or messages 'contain the words'; the delivery matches the whole trimmed query as one literal substring, so a multi-word query matches only contiguous text and never words found separately
    evidence: client/GropiusChat/GropiusChat.swift:876 — "let q = query.trimmingCharacters(in: .whitespaces)"
    evidence: client/GropiusChat/GropiusChat.swift:879 — "c.title.localizedCaseInsensitiveContains(q)"
- missing:
  - The Mechanism's own falsifier — a search field the sidebar column cannot host at its minimum width — was never exercised; the client is not built or run by any gate here and the ledger records no hand-check of the shipped field
    evidence: .abcd/work/DECISIONS.md:284 — "Reviews scaled to the blast radius: one architecture test each."
  - No coverage of any kind exists for the selection's behaviour across a live filter — no Swift test target, and the architecture test asserts only source text
    evidence: internal/archtest/chat_client_composer_test.go:181 — "src := clientSources(t, root)["GropiusChat.swift"]"

Scope-condition dispositions:
- cond-2609181104490024 — survived: All four halves of the assumption are visible in the delivered source: the system's searchable field with sidebar placement, a case-insensitive contains over titles and message text, driven live by the @State query the field is bound to.
  evidence: client/GropiusChat/GropiusChat.swift:871 — "@State private var query = """
  evidence: client/GropiusChat/GropiusChat.swift:880 — "|| c.messages.contains { $0.text.localizedCaseInsensitiveContains(q) }"
  evidence: client/GropiusChat/GropiusChat.swift:905 — ".searchable(text: $query, placement: .sidebar, prompt: "Search")"
- cond-2609181104498637 — narrowed: The empty-search half is literal in the delivered source; the selection half rests only on the absence of any mutation in the filter path — the selection binding is touched solely by the delete paths — and nothing exercised it.
  narrowing: Holds as written only for the empty-search half (the guard returning model.conversations). The selection half now holds only in the weaker sense that no filter code path mutates the selection; whether the selection is actually kept when the selected conversation still matches was exercised by no test and by no manual check the ledger records.
  evidence: client/GropiusChat/GropiusChat.swift:877 — "guard !q.isEmpty else { return model.conversations }"
  evidence: client/GropiusChat/GropiusChat.swift:885 — "List(selection: $selection) {"
  evidence: client/GropiusChat/GropiusChat.swift:899 — "if let s = selection, !model.conversations.contains(where: { $0.id == s }) {"
## Grounds

- pursued: we expect the system's searchable field on the sidebar to be Messages' search bar for free; wrong if the sidebar column cannot host it at its minimum width
