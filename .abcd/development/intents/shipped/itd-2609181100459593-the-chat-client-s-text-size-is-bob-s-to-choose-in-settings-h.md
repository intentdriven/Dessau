---
id: itd-2609181100459593
slug: the-chat-client-s-text-size-is-bob-s-to-choose-in-settings-h
spec_id: spc-2609181100463204
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609151701196720]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# The chat client's text size is Bob's to choose

## Press Release

The chat client's text size is Bob's to choose. In Settings he picks one of
five sizes — smaller, the system's default, larger, extra large, huge — and
the whole window follows: the conversation, the sidebar, the toolbar and
Settings alike. Every font stays the system's, at a different scale, so the
window still looks like a Mac app at every size. The choice persists across a
relaunch. Alice, running the server, sees nothing of it.

## Why This Matters

A chat window is read for long stretches; the system's default is right for
most eyes and wrong for some, and a person who needs larger text should not
have to change their whole Mac for one app.

## Mechanism

We expect one setting to scale the whole window without touching a single
font because SwiftUI's Dynamic Type sizes are an environment value every
standard control and every `Text` already reads, and a size set at the
window's root reaches all of them. What would show this wrong: a control that
ignores the environment's size on macOS 27, or a layout that breaks at the
largest step.

## Scope Conditions

- The five steps are the system's own Dynamic Type sizes (small, large as <!-- cond: cond-2609181100466055 -->
  the default, extra large, double extra large, triple extra large); the
  client stores the choice, never a point size.
- The size is set at the root of every window and of Settings, and applies <!-- cond: cond-2609181100463398 -->
  to the Mac and the iPad clients alike.
- The maintainer's decision at the interview, 2026-09-18: the whole window, <!-- cond: cond-2609181100469817 -->
  not the conversation alone.

## Acceptance Criteria

- Given Settings open, when Bob picks "Larger", then the conversation, the
  sidebar, the toolbar labels and Settings itself grow together, and the
  layout holds.
- Given a size chosen, when Bob quits and relaunches, then the window opens
  at that size.
- Given "Default" chosen, when the window is shown, then it is
  indistinguishable from the client before this change.
- Given the client's source, when the architecture tests run, then the size
  is applied through the Dynamic Type environment at the window and Settings
  roots and the Settings picker offers exactly the five steps.

## Open Questions

_None recorded yet._

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-dbe9ae70feb0 -->
Fidelity review — receipt rcp-dbe9ae70feb0 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:7e7d6fcec03fd529ba2816a12b7041e28cf536ba1eb0e5648d4c3c7884aebd8a
Input attestations: diff:400ccd7..origin/main (PR 61, merged as d388193)@-;

Acceptance rollup: MET 1 · MET_WITH_CONCERNS 3 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: one Dynamic Type value is applied at the window content's root, above the NavigationSplitView that carries the conversation, the sidebar and both toolbars, and again at Settings' root, so all four surfaces read one setting; the concern is that the second half of the promise — the layout holds at the larger steps — is a rendering claim nothing here exercises, and the ledger records the hand check as still owed
  evidence: client/GropiusChat/GropiusChat.swift:941 — ".dynamicTypeSize(dynamicType)"
  evidence: client/GropiusChat/GropiusChat.swift:1010 — "NavigationSplitView {"
  evidence: client/GropiusChat/GropiusChat.swift:1078 — ".toolbar {"
  evidence: client/GropiusChat/GropiusChat.swift:971 — ".dynamicTypeSize(dynamicType)"
  evidence: .abcd/work/DECISIONS.md:280 — "the size steps are checked by hand at the maintainer's next look"
- ac-2 — MET: the choice is held in UserDefaults through @AppStorage at the App itself and read back into the modifier applied at the window's root, so the stored name is what the window opens at
  evidence: client/GropiusChat/GropiusChat.swift:926 — "@AppStorage("textSize") private var textSize: String = TextSize.standard.rawValue"
  evidence: client/GropiusChat/GropiusChat.swift:929 — "private var dynamicType: DynamicTypeSize {"
  evidence: client/GropiusChat/GropiusChat.swift:1424 — "@AppStorage("textSize") private var textSize: String = TextSize.standard.rawValue"
- ac-3 — MET_WITH_CONCERNS: the standard case maps to .large, which is the system's default step, so on a Mac the rendered result should match the prior client; two named caveats: unlike the Appearance enum shipped beside it, TextSize has no 'no preference' case, so .dynamicTypeSize(.large) is applied unconditionally rather than left absent — on a platform where the person set their own Dynamic Type size this pins .large instead of inheriting it — and no hand check of the default's appearance is recorded
  evidence: client/GropiusChat/GropiusChat.swift:891 — "case .standard: return .large"
  evidence: client/GropiusChat/GropiusChat.swift:915 — "case .system: return nil"
  evidence: client/GropiusChat/GropiusChat.swift:941 — ".dynamicTypeSize(dynamicType)"
  evidence: .abcd/work/DECISIONS.md:280 — "the size steps are checked by hand at the maintainer's next look"
- ac-4 — MET_WITH_CONCERNS: the architecture test exists and passes here (go test ./internal/archtest/ reported ok), pinning the five-case enum, the stored setting and the picker; the concern is what it actually asserts — a count of at least two .dynamicTypeSize( occurrences anywhere in the file rather than the two roots the criterion names, and the picker's mere presence rather than that it offers exactly the five steps, which only the enum regex carries
  evidence: internal/archtest/chat_client_composer_test.go:105 — "func TestChatClientTextSizeScalesTheWholeWindow(t *testing.T) {"
  evidence: internal/archtest/chat_client_composer_test.go:108 — "enum TextSize: String, CaseIterable \{\s*case smaller, standard, larger, extraLarge, huge"
  evidence: internal/archtest/chat_client_composer_test.go:111 — "if n := strings.Count(src, ".dynamicTypeSize("); n < 2 {"
  evidence: internal/archtest/chat_client_composer_test.go:117 — "if !strings.Contains(src, `Picker("Text size"`) {"

Gap audit:
- honoured:
  - one setting scales the whole window: the Dynamic Type size is applied at the window's root and at Settings' root
    evidence: client/GropiusChat/GropiusChat.swift:941 — ".dynamicTypeSize(dynamicType)"
    evidence: client/GropiusChat/GropiusChat.swift:971 — ".dynamicTypeSize(dynamicType)"
  - five sizes — smaller, the system's default, larger, extra large, huge — every font the system's at a different scale
    evidence: client/GropiusChat/GropiusChat.swift:876 — "case smaller, standard, larger, extraLarge, huge"
    evidence: client/GropiusChat/GropiusChat.swift:888 — "var dynamicType: DynamicTypeSize {"
  - in Settings he picks one of five sizes
    evidence: client/GropiusChat/GropiusChat.swift:1463 — "Picker("Text size", selection: $textSize) {"
    evidence: client/README.md:62 — "the text size"
  - the choice persists across a relaunch
    evidence: client/GropiusChat/GropiusChat.swift:926 — "@AppStorage("textSize")"
  - Alice, running the server, sees nothing of it — the delivery touches the client, its README, the architecture tests and the changelog only
    evidence: CHANGELOG.md:16 — "The chat client's appearance, text size, bubble colours and composer are"
    evidence: internal/archtest/chat_client_composer_test.go:105 — "func TestChatClientTextSizeScalesTheWholeWindow(t *testing.T) {"
- diverged:
  - the system's default is one of the five steps — delivered as an explicit .large pin rather than as the absence of a preference, which is the shape the Appearance enum shipped in the same diff chose for its System case
    evidence: client/GropiusChat/GropiusChat.swift:891 — "case .standard: return .large"
    evidence: client/GropiusChat/GropiusChat.swift:915 — "case .system: return nil"
  - the spec put the Picker in Settings under a 'Text' section; it is delivered inside a combined 'Appearance' section shared with the light/dark setting
    evidence: client/GropiusChat/GropiusChat.swift:1456 — "Section("Appearance") {"
  - the architecture test was to read 'the two modifier sites'; it reads a count of at least two occurrences anywhere in the file, not their position at the two scene roots
    evidence: internal/archtest/chat_client_composer_test.go:111 — "if n := strings.Count(src, ".dynamicTypeSize("); n < 2 {"
- missing:
  - the spec's verification — the Mac client builds and launches and the sizes are checked by hand and recorded in the shipping decision line; the ledger records that check as still owed, and no rendering of any step is demonstrated anywhere in the delivery
    evidence: .abcd/work/DECISIONS.md:280 — "the size steps are checked by hand at the maintainer's next look"
  - the iPad client applying the same modifier at its window root — the client is built for a macOS target only and no iPad client exists in the repository
    evidence: client/build.sh:16 — "TARGET="arm64-apple-macos27.0""

Scope-condition dispositions:
- cond-2609181100466055 — survived: the five cases map one-to-one onto the system's own Dynamic Type sizes with .large as the default, and what is stored is the case's name, never a point size
  evidence: client/GropiusChat/GropiusChat.swift:891 — "case .standard: return .large"
  evidence: client/GropiusChat/GropiusChat.swift:894 — "case .huge: return .xxxLarge"
  evidence: client/GropiusChat/GropiusChat.swift:926 — "@AppStorage("textSize") private var textSize: String = TextSize.standard.rawValue"
- cond-2609181100463398 — narrowed: the size is set at the root of the window scene and of the Settings scene as assumed, but the second half of the assumption has nothing to hold over: the client is built for a macOS target alone and the repository carries no iPad client
  narrowing: holds for the macOS client's two scene roots only; the 'Mac and iPad clients alike' half is unsupported, as client/build.sh builds a single arm64-apple-macos27.0 target and no iPad client exists
  evidence: client/GropiusChat/GropiusChat.swift:941 — ".dynamicTypeSize(dynamicType)"
  evidence: client/GropiusChat/GropiusChat.swift:971 — ".dynamicTypeSize(dynamicType)"
  evidence: client/build.sh:16 — "TARGET="arm64-apple-macos27.0""
- cond-2609181100469817 — survived: the modifier sits on the whole window's content and on Settings, not on the conversation view, so the maintainer's decision for the whole window is what the code applies
  evidence: client/GropiusChat/GropiusChat.swift:939 — "RootView(model: model)"
  evidence: client/GropiusChat/GropiusChat.swift:941 — ".dynamicTypeSize(dynamicType)"
  evidence: client/GropiusChat/GropiusChat.swift:971 — ".dynamicTypeSize(dynamicType)"
## Grounds

- pursued: we expect one Dynamic Type size at the window's root to scale every control without touching a font; wrong if a control on macOS 27 ignores the environment or the layout breaks at the largest step
