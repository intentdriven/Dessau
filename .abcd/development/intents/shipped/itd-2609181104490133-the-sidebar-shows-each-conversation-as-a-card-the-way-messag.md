---
id: itd-2609181104490133
slug: the-sidebar-shows-each-conversation-as-a-card-the-way-messag
spec_id: spc-2609181104494855
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609151701196720]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# The sidebar shows each conversation as a card

## Press Release

The sidebar shows each conversation as a card, the way Messages does: a
larger row with an icon for who answered, the title, the date the
conversation started, and a summary that says how many exchanges and how
many words it holds. The selected conversation is drawn in the system's
highlight colour, as a selected row in any Mac sidebar is.

## Why This Matters

A list of titles tells Bob what a chat was about but not when, how long, or
with which model; the card is the glance that saves opening it.

## Mechanism

We expect the card to be the standard sidebar row given more content
because a List with the sidebar style already draws its selection in the
system's highlight, and the row's content is a Label with an icon and two
lines of text — no styling of the client's own. What would show this
wrong: a row height the sidebar list refuses to give, or a selection colour
that is not the system's.

## Scope Conditions

- The icon is the system symbol for who answered last: the Mac's own model <!-- cond: cond-2609181104499203 -->
  or a server; the date is the conversation's creation date in the
  system's short format; the summary counts exchanges (a person's message
  and its reply) and words.
- The selection colour is the sidebar's own; nothing is drawn behind the <!-- cond: cond-2609181104494069 -->
  row by the client.

## Acceptance Criteria

- Given three conversations, when Bob looks at the sidebar, then each row
  shows an icon, the title, the date and "N exchanges · M words", and is
  visibly taller than a single line.
- Given a conversation selected, when Bob looks, then its row is in the
  system's highlight colour and its text reads on it.
- Given the client's source, when the architecture tests run, then the row
  carries the four parts and the list uses the sidebar style.

## Open Questions

_None recorded yet._

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-576a7829485a -->
Fidelity review — receipt rcp-576a7829485a (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:0661b7c1b1d2d27a31876cc9a5dbfc78e8025f2fba7a67d971258624e08d22a8
Input attestations: diff:c2bd0ed..6534be1 -- client internal/archtest (PR 66)@-;

Acceptance rollup: MET 0 · MET_WITH_CONCERNS 3 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: ConversationCard puts all four parts on the row in source — Image(systemName: icon), the title, createdAt and the exchanges/words summary — but 'visibly taller than a single line' is a rendered property: the Swift client has no test target, was not built or run here, and the ledger's ship entry records no completed by-hand look check; the summary also singularises to '1 exchange · 1 word' rather than the criterion's literal plural form
  evidence: client/GropiusChat/GropiusChat.swift:939 — "return "\(exchanges) exchange\(exchanges == 1 ? "" : "s") · \(words) word\(words == 1 ? "" : "s")""
  evidence: client/GropiusChat/GropiusChat.swift:948 — "Text(conversation.createdAt, format: .dateTime.day().month().year())"
  evidence: client/GropiusChat/GropiusChat.swift:956 — ".padding(.vertical, 6)"
  evidence: .abcd/work/DECISIONS.md:284 — "Three more client intents SHIPPED with the composer branch on the maintainer's manual-test asks"
- ac-2 — MET_WITH_CONCERNS: the list takes .listStyle(.sidebar) and the card draws nothing behind itself, so the selection is delegated to the system's highlight; the second half — that the row's text reads on that highlight — is a rendered legibility observation over .foregroundStyle(.secondary) date and summary text that nothing here builds, runs or records as checked
  evidence: client/GropiusChat/GropiusChat.swift:904 — ".listStyle(.sidebar)"
  evidence: client/GropiusChat/GropiusChat.swift:949 — "Text(summary).font(.caption).foregroundStyle(.secondary).lineLimit(1)"
  evidence: .abcd/development/specs/closed/spc-2609181104494855-the-sidebar-shows-each-conversation-as-a-card-the-way-messag.md:38 — "Suite, build, docs lint; the look checked by hand."
- ac-3 — MET_WITH_CONCERNS: TestChatClientSidebarShowsCards exists and passes under `go test ./internal/archtest/` and genuinely pins .listStyle(.sidebar) and ConversationCard as the sidebar's row, but it is a file-wide strings.Contains scan over GropiusChat.swift rather than a read of the row: it is not scoped to the ConversationCard block the way the sibling Thoughts test is, it asserts nothing about the icon or the title, and `createdAt` matches the Conversation model's own field declaration at line 107 whether or not the card shows a date
  evidence: internal/archtest/chat_client_composer_test.go:169 — "for _, want := range []string{`struct ConversationCard: View`, `.listStyle(.sidebar)`, `ConversationCard(conversation:`, `createdAt`, `exchange`, `word`} {"
  evidence: client/GropiusChat/GropiusChat.swift:107 — "var createdAt = Date()"
  evidence: internal/archtest/chat_client_composer_test.go:145 — "block := src[start[0]:]"

Gap audit:
- honoured:
  - the sidebar row is a card: an icon for who answered, the title, the date it started, and a summary of exchanges and words
    evidence: client/GropiusChat/GropiusChat.swift:920 — "struct ConversationCard: View"
    evidence: client/GropiusChat/GropiusChat.swift:952 — "Image(systemName: icon).font(.title2)"
  - the selected conversation is drawn in the system's highlight because the List takes the sidebar style
    evidence: client/GropiusChat/GropiusChat.swift:904 — ".listStyle(.sidebar)"
  - no styling of the client's own — the card needed no named exception to the no-styling architecture test, unlike Bubbles.swift
    evidence: internal/archtest/chat_client_native_test.go:48 — "exempt := map[string]string{"
    evidence: client/GropiusChat/GropiusChat.swift:953 — "} icon: {"
  - the user-facing client README describes the card
    evidence: client/README.md:79 — "The sidebar shows each conversation as a card — who answered, the title,"
- diverged:
  - the summary reads "N exchanges · M words"; delivered, it singularises at a count of one — "1 exchange · 1 word"
    evidence: client/GropiusChat/GropiusChat.swift:939 — "exchange\(exchanges == 1 ? "" : "s")"
  - the architecture test reads the card's parts; delivered, it scans the whole file for six substrings, scoped to no block, with the icon and title unasserted
    evidence: internal/archtest/chat_client_composer_test.go:169 — "`createdAt`, `exchange`, `word`"
  - the date is the creation date in the system's short format; delivered, it is an explicit day/month/year field list, not the system's short date style
    evidence: client/GropiusChat/GropiusChat.swift:948 — "format: .dateTime.day().month().year()"
  - an exchange is a person's message and its reply; delivered, the count is the assistant replies alone, a proxy for the pair
    evidence: client/GropiusChat/GropiusChat.swift:937 — "let exchanges = conversation.messages.filter { $0.role == .assistant }.count"
- missing:
  - the by-hand look check the spec's Verification promised — the rendered row height and the selected row's legibility are never observed; the ledger's ship entry records the manual-test asks but no completed look check, and the Swift client has no test target here
    evidence: .abcd/development/specs/closed/spc-2609181104494855-the-sidebar-shows-each-conversation-as-a-card-the-way-messag.md:38 — "the look checked by hand."
    evidence: .abcd/work/DECISIONS.md:284 — "Reviews scaled to the blast radius: one architecture test each."

Scope-condition dispositions:
- cond-2609181104499203 — narrowed: the icon resolves to a system symbol from the last reply's recorded answerer and the summary counts replies and whitespace-split words, but the date is rendered through an explicit day/month/year field list rather than the system's short date style, and an exchange is counted as an assistant reply rather than the person-message-and-its-reply pair the condition defines
  narrowing: holds with the date as an explicit .dateTime.day().month().year() field list — locale-ordered but not the system's short date style — and with an exchange counted as one assistant reply rather than the message pair
  evidence: client/GropiusChat/GropiusChat.swift:948 — "Text(conversation.createdAt, format: .dateTime.day().month().year())"
  evidence: client/GropiusChat/GropiusChat.swift:933 — "return last == BuiltInBackend.recordedName ? "apple.intelligence" : "network""
  evidence: client/GropiusChat/GropiusChat.swift:937 — "let exchanges = conversation.messages.filter { $0.role == .assistant }.count"
- cond-2609181104494069 — survived: the List takes the sidebar style so the selection colour is the system's, and the card's body draws nothing behind the row — no background, overlay, shape or colour — which is why GropiusChat.swift still passes the no-styling architecture test without joining Effects.swift and Bubbles.swift in its named exemptions
  evidence: client/GropiusChat/GropiusChat.swift:904 — ".listStyle(.sidebar)"
  evidence: client/GropiusChat/GropiusChat.swift:956 — ".padding(.vertical, 6)"
  evidence: internal/archtest/chat_client_native_test.go:45 — "".buttonStyle(.glass", ".background(", ".overlay(", ".shadow(","
## Grounds

- pursued: we expect the sidebar's standard row with more content and the sidebar list style to read as Messages' cards with no drawing of our own; wrong if the row height or the highlight is not the system's
