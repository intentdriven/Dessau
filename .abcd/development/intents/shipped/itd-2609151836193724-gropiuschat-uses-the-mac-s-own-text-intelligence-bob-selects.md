---
id: itd-2609151836193724
slug: gropiuschat-uses-the-mac-s-own-text-intelligence-bob-selects
spec_id: spc-2609170842097413
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609151701196720]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# GropiusChat uses the Mac's own text intelligence

## Press Release

GropiusChat uses the Mac's own text intelligence. Bob selects a reply, or
his own draft in the composer, and the system's Writing Tools appear the way
they do in every Apple app — proofread, rewrite, summarise — working on the
text where it is. In the composer the rewrite lands in place of what he
selected; on a reply it opens in the system's panel, ready to copy. The
client sends nothing to Gropius or anywhere else for this: it is the Mac's
on-device feature, present because the client is a native macOS 27 app.
Alice, running the server, sees no request from it.

## Why This Matters

Writing Tools are what a Mac user reaches for when a draft is not right, and
they are absent from the client only because its composer was a hand-built
text view. A native app gets them by being native; keeping them out would be
the styling the trunk removes, in another form.

## Mechanism

We expect Writing Tools to appear without code of the client's own because
SwiftUI's `TextField` and a selectable `Text` carry them on macOS when Apple
Intelligence is on, and the trunk makes the composer a standard `TextField`.
What would show this wrong: a selectable `Text` whose context menu on macOS
27 shows no Writing Tools item, in which case the reply views need
`writingToolsBehavior(.complete)` or a `TextEditor` to carry them.

## Scope Conditions

- Apple Intelligence switched on, on a Mac and in a language the system <!-- cond: cond-2609170842094850 -->
  supports; the client neither checks nor shows an error when it is off — the
  system decides whether the menu item appears.
- The 27 client only, on the trunk's standard composer: a multi-line <!-- cond: cond-2609170842097488 -->
  `TextField` sends on Return and takes a line break on Option-Return, and a
  rewrite that carries line breaks lands in the field as text, not as a send.
- The reply view is shared with the formatted-replies and text-effects <!-- cond: cond-2609170842096052 -->
  intents: a reply is one or more selectable `Text` blocks, and an effect
  never leaves a block unselectable once it has played.
- The first three criteria are checked by hand on the maintainer's Mac and <!-- cond: cond-2609170842092363 -->
  recorded in the shipping decision line; the fourth is the architecture
  test.

## Acceptance Criteria

- Given Apple Intelligence is on, when Bob selects words in a reply and opens
  the context menu, then Writing Tools is there and a rewrite opens in the
  system's panel, read-only, with the result ready to copy; the reply itself
  never becomes editable.
- Given the composer holds a draft, when Bob selects it and chooses a rewrite
  from Writing Tools, then the rewritten text replaces the selection in the
  composer and nothing is sent to any server.
- Given Apple Intelligence is off, when Bob selects text, then no Writing
  Tools item appears and the client shows no error of its own.
- Given the client's source, when the architecture tests run, then the
  composer is a standard `TextField` and no `writingToolsBehavior(.disabled)`
  appears anywhere in the client.

## Open Questions

_None recorded yet._

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-d6fd7dcb1049 -->
Fidelity review — receipt rcp-d6fd7dcb1049 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:b3b802d15f75773703737a517147ea93eef76b1729bcf176161154bb5c9b1439
Input attestations: diff:acacd07..210ce66 (PR 58, on origin/main at 400ccd7)@-;

Acceptance rollup: MET 1 · MET_WITH_CONCERNS 0 · NOT_MET 0 · INCONCLUSIVE 3

Per-criterion verdicts:
- ac-1 — INCONCLUSIVE: the reply blocks are selectable Text and never become editable, which is citable, but whether the context menu carries Writing Tools and opens a read-only panel is a manual check the shipping decision line records as NOT verified and still owed, so the criterion's central observable cannot be resolved from the delivered artefacts
  evidence: client/GropiusChat/GropiusChat.swift:1300 — "Text(s).textSelection(.enabled)"
  evidence: .abcd/work/DECISIONS.md:277 — "NOT verified, the maintainer's manual checks, still owed: Writing Tools in a reply's context menu and in the composer"
- ac-2 — INCONCLUSIVE: the composer is a standard multi-line TextField bound to a local draft with no code that sends on change, so the 'nothing sent to any server' half is supported, but that a Writing Tools rewrite replaces the selection in the field is the unrun manual check the decision line still owes
  evidence: client/GropiusChat/GropiusChat.swift:1091 — "TextField("Message…", text: $draft, axis: .vertical)"
  evidence: client/GropiusChat/GropiusChat.swift:1093 — ".onSubmit(send)"
  evidence: .abcd/work/DECISIONS.md:277 — "NOT verified, the maintainer's manual checks, still owed: Writing Tools in a reply's context menu and in the composer"
- ac-3 — INCONCLUSIVE: no client code touches Writing Tools or gates on Apple Intelligence for text selection, but the Apple-Intelligence-off behaviour was never exercised on a Mac, and the client does carry an Apple-Intelligence-off message of its own for the built-in model backend, so the 'no error of its own' claim holds only outside that path
  evidence: client/GropiusChat/Backends.swift:101 — "reason: "Apple Intelligence is switched off, so the Mac cannot answer.""
  evidence: .abcd/work/DECISIONS.md:277 — "NOT verified, the maintainer's manual checks, still owed"
- ac-4 — MET: the architecture test exists, was run here and passes: it reads every client Swift source, fails on writingToolsBehavior(.disabled)/NSViewRepresentable/NSTextView, and requires the multi-line TextField composer, which the source carries
  evidence: internal/archtest/chat_client_native_test.go:95 — "for _, s := range []string{"writingToolsBehavior(.disabled)", "NSViewRepresentable", "NSTextView"} {"
  evidence: internal/archtest/chat_client_native_test.go:101 — "regexp.MustCompile(`TextField\("Message…", text: \$draft, axis: \.vertical\)`)"
  evidence: client/GropiusChat/GropiusChat.swift:1091 — "TextField("Message…", text: $draft, axis: .vertical)"

Gap audit:
- honoured:
  - the composer is a standard multi-line TextField, so the system's Writing Tools reach it with no code of the client's own
    evidence: client/GropiusChat/GropiusChat.swift:1091 — "TextField("Message…", text: $draft, axis: .vertical)"
  - nothing in the client suppresses Writing Tools, and an architecture test pins that against a later edit
    evidence: internal/archtest/chat_client_native_test.go:91 — "func TestChatClientKeepsWritingTools(t *testing.T) {"
  - a reply is selectable Text blocks and an effect leaves its block selectable once it has played
    evidence: client/GropiusChat/Effects.swift:121 — "if finished || matches.isEmpty {"
    evidence: client/GropiusChat/Effects.swift:122 — "Text(text).textSelection(.enabled)"
  - the client README says Writing Tools work in the composer and on replies when Apple Intelligence is on
    evidence: client/README.md:54 — "a reply and the system's **Writing Tools** — proofread, rewrite, summarise —"
- diverged:
  - the press release says the client neither checks Apple Intelligence's state nor shows an error of its own when it is off; the delivered client reads SystemLanguageModel.availability and shows an Apple-Intelligence-off message for the built-in model backend, outside the Writing Tools path
    evidence: client/GropiusChat/Backends.swift:101 — "reason: "Apple Intelligence is switched off, so the Mac cannot answer.""
  - the spec names the composer as TextField("Message…", text: $model.input, axis: .vertical); the delivered composer binds a local $draft instead, and the architecture test pins that spelling
    evidence: client/GropiusChat/GropiusChat.swift:1091 — "TextField("Message…", text: $draft, axis: .vertical)"
- missing:
  - the three manual checks on the maintainer's Mac (Writing Tools in a reply's menu, a rewrite in the composer, and the Apple-Intelligence-off case) recorded in the shipping decision line; the line records them as owed, not as run
    evidence: .abcd/work/DECISIONS.md:277 — "NOT verified, the maintainer's manual checks, still owed: Writing Tools in a reply's context menu and in the composer"
  - the changelog states as shipped fact that Writing Tools work in the composer and on replies, which no run check in the delivery backs
    evidence: CHANGELOG.md:31 — "Writing Tools work in the composer and on replies, a few words animate"

Scope-condition dispositions:
- cond-2609170842094850 — narrowed: no client code checks Apple Intelligence for text selection, but the same client does read SystemLanguageModel.availability and shows its own Apple-Intelligence-off message for the built-in model
  narrowing: holds for the Writing Tools / text-selection path only; the client does check Apple Intelligence's state and shows an error of its own in the built-in model backend
  evidence: client/GropiusChat/Backends.swift:89 — "switch model.availability {"
  evidence: client/GropiusChat/Backends.swift:101 — "reason: "Apple Intelligence is switched off, so the Mac cannot answer.""
- cond-2609170842097488 — narrowed: the 27-only scope and the standard multi-line TextField with onSubmit are both in the delivered source, but nothing in the delivery exercises Option-Return or a rewrite carrying line breaks landing as text rather than a send
  narrowing: holds as a source-level claim about the standard control on the 27 client; the Return / Option-Return and line-break-rewrite behaviour is assumed from the control, never exercised by a test or a recorded manual check
  evidence: client/GropiusChat/GropiusChat.swift:1092 — ".lineLimit(1...8)"
  evidence: client/Info.plist:22 — "< string>27.0< /string>"
- cond-2609170842096052 — survived: every reply block in the shared view is Text(...).textSelection(.enabled), and the effect renderer applies only while the animation plays, falling back to a selectable Text once finished
  evidence: client/GropiusChat/Effects.swift:121 — "if finished || matches.isEmpty {"
  evidence: client/GropiusChat/GropiusChat.swift:1257 — ".textSelection(.enabled)"
- cond-2609170842092363 — falsified: the condition assumed the first three criteria would be checked by hand and recorded in the shipping decision line; the shipping decision line instead records those checks as NOT verified and still owed, while only the fourth leg — the architecture test — held
  evidence: .abcd/work/DECISIONS.md:277 — "NOT verified, the maintainer's manual checks, still owed: Writing Tools in a reply's context menu and in the composer"
  evidence: internal/archtest/chat_client_native_test.go:91 — "func TestChatClientKeepsWritingTools(t *testing.T) {"
## Grounds

- pursued: we expect Writing Tools to arrive with no code of the client's own once the composer is a standard TextField and replies are selectable Text, because the system attaches them to those controls; wrong if a selectable Text on macOS 27 shows no Writing Tools in its context menu
