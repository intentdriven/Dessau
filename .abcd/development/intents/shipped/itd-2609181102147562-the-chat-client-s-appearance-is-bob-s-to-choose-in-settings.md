---
id: itd-2609181102147562
slug: the-chat-client-s-appearance-is-bob-s-to-choose-in-settings
spec_id: spc-2609181102146375
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609151701196720]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# The chat client's appearance is Bob's to choose

## Press Release

The chat client's appearance is Bob's to choose. In Settings he picks Light,
Dark or System, and the whole window follows at once — the conversation, the
sidebar, the toolbar and Settings alike. System means the client follows the
Mac's own appearance, as it does today. The choice persists across a
relaunch. Alice, running the server, sees nothing of it.

## Why This Matters

A chat window is often the one window someone wants dark at night or light
beside a document, without changing the whole Mac. Every colour the client
uses is the system's, so an appearance chosen here still looks like a Mac
app.

## Mechanism

We expect one setting to switch the whole window because SwiftUI's preferred
colour scheme, set at a scene's root, is what every standard control and the
system's colours resolve against, and no preference at all is exactly the
system's behaviour. What would show this wrong: a view that resolves a colour
outside the environment on macOS 27, or a window that ignores the preference
until it is reopened.

## Scope Conditions

- Three choices, and no more: Light, Dark, System; System is the absence of a <!-- cond: cond-2609181102149647 -->
  preference, not a third scheme.
- Applied at the root of every window and of Settings, on the Mac and the <!-- cond: cond-2609181102147799 -->
  iPad clients alike; the bubble colours and every other colour stay the
  system's and follow.

## Acceptance Criteria

- Given Settings open, when Bob picks Dark, then every open window and
  Settings turn dark at once without being reopened; when he picks Light,
  they turn light.
- Given System chosen, when the Mac's appearance changes, then the client
  follows it, as it did before this change.
- Given a choice made, when Bob quits and relaunches, then the window opens
  in that appearance.
- Given the client's source, when the architecture tests run, then the
  preference is applied at the window and Settings roots and the picker
  offers exactly Light, Dark and System.

## Open Questions

_None recorded yet._

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-df22ad80d10d -->
Fidelity review — receipt rcp-df22ad80d10d (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:024ed2e9d3465749b6d218777a1ae7f1cd2666e2b245b71eb9356fccd59178f1
Input attestations: diff:400ccd7..origin/main -- client internal/archtest CHANGELOG.md (PR 61, merged as d388193)@sha256:9ba8f5a61d6eea70265f920e09b7f66e7e3ca4d990ae69275c77089565543c5b;

Acceptance rollup: MET 1 · MET_WITH_CONCERNS 3 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: One App-level @AppStorage drives .preferredColorScheme at both the WindowGroup content root and the Settings content root, so a change invalidates both scenes' bodies; concern: the 'at once without being reopened' clause is the intent's own named falsifier and nothing exercises it — the Swift client has no test target and the shipping decision line records only the architecture test, no hand check.
  evidence: client/GropiusChat/GropiusChat.swift:942 — ".preferredColorScheme(colorScheme)"
  evidence: client/GropiusChat/GropiusChat.swift:972 — ".preferredColorScheme(colorScheme)"
  evidence: client/GropiusChat/GropiusChat.swift:927 — "@AppStorage("appearance") private var appearance: String = Appearance.system.rawValue"
  evidence: .abcd/work/DECISIONS.md:281 — "Reviews scaled to the blast radius: one architecture test."
- ac-2 — MET_WITH_CONCERNS: System maps to a nil preference, which is by construction the no-preference state the client had before this change; concern: that the running app on macOS 27 still follows a live system-appearance change is unverified — no runtime or manual check is recorded for it.
  evidence: client/GropiusChat/GropiusChat.swift:915 — "case .system: return nil"
  evidence: client/GropiusChat/GropiusChat.swift:934 — "Appearance(rawValue: appearance)?.colorScheme"
  evidence: .abcd/work/DECISIONS.md:281 — "Light, Dark or System as the preferred colour scheme at the window's and Settings' roots, System being no preference."
- ac-3 — MET_WITH_CONCERNS: The choice is held in @AppStorage("appearance"), i.e. UserDefaults, and read back at the App root as the scenes' preferred scheme, which is a genuine persistence mechanism; concern: the quit-and-relaunch outcome itself is never exercised — the client cannot be built or run in this repo's test lane and the decision record lists no hand check of it.
  evidence: client/GropiusChat/GropiusChat.swift:927 — "@AppStorage("appearance") private var appearance: String = Appearance.system.rawValue"
  evidence: client/GropiusChat/GropiusChat.swift:1425 — "@AppStorage("appearance") private var appearance: String = Appearance.system.rawValue"
  evidence: .abcd/work/DECISIONS.md:281 — "Reviews scaled to the blast radius: one architecture test."
- ac-4 — MET: The architecture test exists and was run here green (go test -count=1 -run TestChatClientAppearanceFollowsOneSetting ./internal/archtest/ => PASS), and the source it guards does apply the preference at the window root and the Settings root and offers exactly the three cases through Appearance.allCases.
  evidence: internal/archtest/chat_client_composer_test.go:126 — "func TestChatClientAppearanceFollowsOneSetting(t *testing.T) {"
  evidence: internal/archtest/chat_client_composer_test.go:129 — "enum Appearance: String, CaseIterable \{\s*case system, light, dark"
  evidence: client/GropiusChat/GropiusChat.swift:903 — "case system, light, dark"
  evidence: client/GropiusChat/GropiusChat.swift:1458 — "ForEach(Appearance.allCases, id: \.rawValue) { a in"

Gap audit:
- honoured:
  - In Settings he picks Light, Dark or System
    evidence: client/GropiusChat/GropiusChat.swift:1457 — "Picker("Appearance", selection: $appearance) {"
    evidence: client/GropiusChat/GropiusChat.swift:1456 — "Section("Appearance") {"
  - one setting switches the whole window and Settings alike, applied at each scene's root
    evidence: client/GropiusChat/GropiusChat.swift:942 — ".preferredColorScheme(colorScheme)"
    evidence: client/GropiusChat/GropiusChat.swift:972 — ".preferredColorScheme(colorScheme)"
  - System means the client follows the Mac's own appearance, as it does today — no preference, not a third scheme
    evidence: client/GropiusChat/GropiusChat.swift:915 — "case .system: return nil"
  - the choice persists across a relaunch
    evidence: client/GropiusChat/GropiusChat.swift:927 — "@AppStorage("appearance") private var appearance: String = Appearance.system.rawValue"
  - Alice, running the server, sees nothing of it — the setting lives wholly in the client's source
    evidence: client/GropiusChat/GropiusChat.swift:902 — "enum Appearance: String, CaseIterable {"
    evidence: client/README.md:62 — "Settings also offers the appearance (Light, Dark or System), the text size"
- diverged:
  - the architecture test holds the preference 'at the window and Settings roots' — delivered as a whole-file count of '.preferredColorScheme(' occurrences (n < 2), which would pass with both modifiers inside one scene, and the enum regex pins only the first case line, so a fourth case on a following line would slip past
    evidence: internal/archtest/chat_client_composer_test.go:132 — "if n := strings.Count(src, ".preferredColorScheme("); n < 2 {"
    evidence: internal/archtest/chat_client_composer_test.go:129 — "regexp.MustCompile(`enum Appearance: String, CaseIterable \{\s*case system, light, dark`)"
  - 'every colour the client uses is the system's, so an appearance chosen here still looks like a Mac app' — the user bubble's text is a fixed Color.white and a picked bubble colour is a stored literal hex; neither is a system colour that follows the chosen appearance
    evidence: client/GropiusChat/Bubbles.swift:61 — ".foregroundStyle(isUser ? Color.white : Color.primary)"
    evidence: client/GropiusChat/Bubbles.swift:20 — "var userColor: Color { Color(hex: user) ?? Self.defaultUser }"
  - the feature commit 9e25b3a ('feat: the chat client's appearance is Bob's to choose') carried the test, the README and the changelog but no Swift change; the setting itself arrived only in the follow-up 58c0471, so the promise was true of the range and not of the commit that announced it
    evidence: client/GropiusChat/GropiusChat.swift:902 — "enum Appearance: String, CaseIterable {"
    evidence: CHANGELOG.md:16 — "- **The chat client's appearance, text size, bubble colours and composer are"
- missing:
  - 'the Mac and the iPad clients alike' — the tree carries no iPad or iPadOS client; build.sh compiles one macOS-only target, and the sole UIKit branch is a hex-conversion fallback, not a client
    evidence: client/build.sh:16 — "TARGET="arm64-apple-macos27.0""
    evidence: client/GropiusChat/Bubbles.swift:43 — "#else"
  - the spec's Verification promised 'the three appearances checked by hand and recorded in the shipping decision line'; the shipping decision line records only the architecture test and no hand check, owed or done
    evidence: .abcd/work/DECISIONS.md:281 — "Reviews scaled to the blast radius: one architecture test."
    evidence: .abcd/development/specs/closed/spc-2609181102146375-the-chat-client-s-appearance-is-bob-s-to-choose-in-settings.md:48 — "three appearances checked by hand and recorded in the shipping decision"

Scope-condition dispositions:
- cond-2609181102149647 — survived: The enum carries exactly the three cases and System resolves to nil, so it is the absence of a preference rather than a third scheme, and the picker iterates allCases so it can offer no more.
  evidence: client/GropiusChat/GropiusChat.swift:903 — "case system, light, dark"
  evidence: client/GropiusChat/GropiusChat.swift:915 — "case .system: return nil"
  evidence: client/GropiusChat/GropiusChat.swift:1458 — "ForEach(Appearance.allCases, id: \.rawValue) { a in"
- cond-2609181102147799 — narrowed: The modifier is at both Mac scene roots as assumed, but the tree holds no iPad client for it to reach and two of the transcript's colours are not system colours that follow the preference.
  narrowing: Holds for the Mac client's two scene roots only: build.sh compiles a macOS-only target and no iPad client exists in the tree, and 'every other colour stays the system's and follows' does not hold for the user bubble's fixed Color.white text nor for a bubble colour the operator picks, which is stored as a literal hex.
  evidence: client/GropiusChat/GropiusChat.swift:942 — ".preferredColorScheme(colorScheme)"
  evidence: client/build.sh:16 — "TARGET="arm64-apple-macos27.0""
  evidence: client/GropiusChat/Bubbles.swift:61 — ".foregroundStyle(isUser ? Color.white : Color.primary)"
  evidence: client/GropiusChat/Bubbles.swift:20 — "var userColor: Color { Color(hex: user) ?? Self.defaultUser }"
## Grounds

- pursued: we expect the preferred colour scheme at the scene roots to switch every window at once with no colour of the client's own; wrong if a window ignores the preference until reopened
