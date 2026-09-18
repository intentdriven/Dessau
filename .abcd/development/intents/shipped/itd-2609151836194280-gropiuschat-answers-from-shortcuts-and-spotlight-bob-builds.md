---
id: itd-2609151836194280
slug: gropiuschat-answers-from-shortcuts-and-spotlight-bob-builds
spec_id: spc-2609170842092383
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609151701196720]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# GropiusChat answers from Shortcuts and Spotlight

## Press Release

GropiusChat answers from Shortcuts and Spotlight. Bob builds a shortcut
that sends a prompt to the model the client is set to — the Mac's own by
default, one of Alice's if he has chosen it — and gets the reply back as
text; the exchange is kept as a new chat in the client's sidebar. From
Spotlight he starts a new chat or opens an existing one by name. The client
declares its actions as App Intents, so the system lists them by itself and
Bob never configures anything to make them appear.

## Why This Matters

Shortcuts and Spotlight are how a Mac user makes an app part of their day
without opening it. A chat that can be asked from a shortcut is a tool; one
that can only be typed into is a window.

## Mechanism

We expect Shortcuts and Spotlight to list the actions because App Intents
metadata is extracted at build time by the toolchain's metadata processor,
which the build script runs itself the way other non-Xcode build systems do,
and the system reads it from the bundle. What would show this wrong: the
processor refusing a single-file build, or the system not indexing an
ad-hoc-signed, zip-installed bundle until it has been opened once — in which
case the acceptance bar moves to "after the first launch".

## Scope Conditions

- Built with the installed Xcode 27 toolchain: the metadata processor ships <!-- cond: cond-2609170842095773 -->
  only in Xcode, not in the Command Line Tools. The trunk's condition on the
  toolchain was narrowed to exactly this on 2026-09-17 before this intent is
  planned, and the trunk's spec names the processor step as this intent's.
- The actions run inside the app's own process — the system launches it in <!-- cond: cond-2609170842096291 -->
  the background when it is not running — through the same model object the
  windows use, so there is no second writer of the conversations file.
- Chats are looked up on demand through an entity query when Bob picks one; <!-- cond: cond-2609170842094378 -->
  nothing is donated to the system's index, so no title, prompt or reply
  leaves the app's own store.
- The 27 client only; the prompt action answers with whichever model the <!-- cond: cond-2609170842090438 -->
  client is set to, so a Gropius model needs the app's stored server.

## Acceptance Criteria

- Given the client is installed and has been opened once, when Bob opens
  Shortcuts, then "Ask Gropius" (a prompt, returning text), "New Chat" and
  "Open Chat" (a chat, chosen by name) are listed under the app.
- Given a shortcut runs "Ask Gropius", when the model replies, then the
  shortcut receives the reply text, and the exchange appears in the client's
  sidebar as a chat titled from the prompt.
- Given Spotlight, when Bob types "New Chat", then the client's action is
  offered and running it opens the client on a new chat.
- Given the prompt action is declared long-running, when a model takes over
  a minute to answer (checked by hand with a cold Gropius model and recorded
  in the shipping decision line), then the shortcut completes with the reply
  rather than timing out.
- Given the build, when the metadata processor does not run or writes
  nothing, then the build fails and says so; a test holds the bundle to
  carrying the metadata.

## Open Questions

_None recorded yet._

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-4049a6fa90f8 -->
Fidelity review — receipt rcp-4049a6fa90f8 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:dd736e3d51ed95c499902144222e4e3c175a8749a386d9d12a645ada311e2f0a
Input attestations: diff:acacd07..210ce66 (PR 58, origin/main at 400ccd7) -- client internal/archtest install.sh .github README.md docs CHANGELOG.md@sha256:da0c18bfb59c2c67e4e35a363d5d29a445eb86969c738f5c943f26fd3167cbdb;

Acceptance rollup: MET 0 · MET_WITH_CONCERNS 2 · NOT_MET 1 · INCONCLUSIVE 2

Per-criterion verdicts:
- ac-1 — INCONCLUSIVE: the three intents and the shortcuts provider are declared in source and the build writes the metadata, but whether Shortcuts lists them under the app after a first launch is a system behaviour the intent itself flagged as risky, and the shipping decision line records that hand check as still owed
  evidence: client/GropiusChat/Intents.swift:92 — "struct GropiusShortcuts: AppShortcutsProvider {"
  evidence: client/GropiusChat/Intents.swift:14 — "static let title: LocalizedStringResource = "Ask Gropius""
  evidence: .abcd/work/DECISIONS.md:277 — "NOT verified, the maintainer's manual checks, still owed: ... Shortcuts listing the three actions after the first launch and the long-running prompt"
- ac-2 — MET_WITH_CONCERNS: AskIntent returns the reply as its value and AppModel.ask creates a chat whose title is set from the first words of the prompt, so the whole path is present and readable in source; the concern is that no test in the delivered suite exercises any Swift path and no run has been observed, the client being unbuildable outside Xcode 27
  evidence: client/GropiusChat/Intents.swift:28 — "return .result(value: reply)"
  evidence: client/GropiusChat/GropiusChat.swift:751 — "func ask(prompt: String) async throws -> String {"
  evidence: client/GropiusChat/GropiusChat.swift:733 — "conversations[idx].title = String(prompt.prefix(48))"
  evidence: .abcd/work/DECISIONS.md:277 — "NOT verified, the maintainer's manual checks, still owed"
- ac-3 — INCONCLUSIVE: NewChatIntent is declared with openAppWhenRun and carries a shortcut phrase, so the offer is prepared, but whether Spotlight offers it and opens the client on a new chat is unobserved system behaviour and the manual check is recorded as owed
  evidence: client/GropiusChat/Intents.swift:36 — "static let openAppWhenRun = true"
  evidence: client/GropiusChat/Intents.swift:98 — "AppShortcut(intent: NewChatIntent(),"
  evidence: .abcd/work/DECISIONS.md:277 — "NOT verified, the maintainer's manual checks, still owed"
- ac-4 — NOT_MET: the criterion makes a hand check with a cold Gropius model, recorded in the shipping decision line, part of its own bar; AskIntent does conform to LongRunningIntent, but the shipping decision line records that very check as NOT verified and still owed, so the promised record is not merely absent, it affirmatively says the opposite
  evidence: client/GropiusChat/Intents.swift:13 — "struct AskIntent: AppIntent, LongRunningIntent {"
  evidence: .abcd/work/DECISIONS.md:277 — "NOT verified, the maintainer's manual checks, still owed: ... Shortcuts listing the three actions after the first launch and the long-running prompt"
- ac-5 — MET_WITH_CONCERNS: build.sh runs the processor and exits 1 with a named message when extract.actionsdata is absent, and a test that was run green pins that step; the concern is that the test reads the build script's TEXT rather than holding a built bundle to carrying the metadata, which is what the spec named and what the criterion's second clause says
  evidence: client/build.sh:57 — "[ -f "$RES/Metadata.appintents/extract.actionsdata" ] ||"
  evidence: client/build.sh:58 — "{ echo "error: the App Intents metadata was not written; Shortcuts would list nothing" >&2; exit 1; }"
  evidence: internal/archtest/chat_client_native_test.go:153 — "raw := readRepoFile(t, root, filepath.Join("client", "build.sh"))"

Gap audit:
- honoured:
  - three App Intents — a prompt returning text, a new chat, and a chat chosen by name — plus an AppShortcutsProvider with one phrase each
    evidence: client/GropiusChat/Intents.swift:92 — "struct GropiusShortcuts: AppShortcutsProvider {"
    evidence: client/GropiusChat/Intents.swift:45 — "struct OpenChatIntent: AppIntent {"
  - the build script writes the App Intents metadata with the toolchain's own processor and fails loudly when nothing was written
    evidence: client/build.sh:45 — "xcrun appintentsmetadataprocessor \"
    evidence: client/build.sh:58 — "echo "error: the App Intents metadata was not written; Shortcuts would list nothing" >&2; exit 1;"
  - the actions reach the same model object the windows use, so the conversations file has one writer
    evidence: client/GropiusChat/GropiusChat.swift:410 — "static let shared = AppModel()"
    evidence: client/GropiusChat/GropiusChat.swift:869 — "@StateObject private var model = AppModel.shared"
  - chats are looked up on demand through an entity query; nothing is donated to the system's index
    evidence: client/GropiusChat/Intents.swift:74 — "struct ChatQuery: EntityStringQuery {"
    evidence: client/README.md:88 — "looked up when you pick one; nothing is added to the system's index."
  - the client README gains its Shortcuts and Spotlight section
    evidence: client/README.md:83 — "### Shortcuts and Spotlight"
- diverged:
  - a test holds the BUNDLE to carrying the metadata (spec: TestChatClientBundleCarriesAppIntentsMetadata); delivered is TestChatClientBuildWritesTheAppIntentsMetadata, which greps client/build.sh for the processor step and its failure line — the suite never builds or inspects a bundle, so a processor that silently stopped writing would still pass
    evidence: internal/archtest/chat_client_native_test.go:151 — "func TestChatClientBuildWritesTheAppIntentsMetadata(t *testing.T) {"
    evidence: internal/archtest/chat_client_native_test.go:153 — "raw := readRepoFile(t, root, filepath.Join("client", "build.sh"))"
  - the shipped changelog states as fact that Shortcuts and Spotlight list the client's actions, while the shipping decision line records that exact check as not yet performed
    evidence: CHANGELOG.md:32 — "and Shortcuts and Spotlight list the"
    evidence: .abcd/work/DECISIONS.md:277 — "NOT verified, the maintainer's manual checks, still owed"
- missing:
  - the long-running prompt checked by hand with a cold Gropius model and recorded in the shipping decision line — the decision line records it as owed instead
    evidence: .abcd/work/DECISIONS.md:277 — "NOT verified, the maintainer's manual checks, still owed: ... the long-running prompt"
  - the hand check that Shortcuts lists the three actions after the first launch, which the intent's Mechanism named as the thing that would move the acceptance bar
    evidence: .abcd/work/DECISIONS.md:277 — "Shortcuts listing the three actions after the first launch"

Scope-condition dispositions:
- cond-2609170842095773 — survived: the build script drives the processor out of the installed Xcode's toolchain directory and the release job selects Xcode 27 or later and fails loudly without it, exactly as the condition assumed
  evidence: client/build.sh:47 — "--toolchain-dir "$(xcode-select -p)/Toolchains/XcodeDefault.xctoolchain""
  evidence: .github/workflows/release.yml:184 — "no Xcode 27 or later on this runner; the client cannot be built"
- cond-2609170842096291 — narrowed: the single-writer half is demonstrable in source — one AppModel.shared that both the intents and the root view hold, and no intents extension target in the delivery — but no intent has ever been run, so the in-process background launch the condition also assumed was never exercised
  narrowing: holds as a source-level structure (one shared AppModel, no extension target, so one writer of the conversations file); the system launching the app in the background to run an intent is unobserved, the Shortcuts hand check being recorded as owed
  evidence: client/GropiusChat/GropiusChat.swift:410 — "static let shared = AppModel()"
  evidence: client/GropiusChat/Intents.swift:25 — "try await AppModel.shared.ask(prompt: prompt)"
  evidence: .abcd/work/DECISIONS.md:277 — "NOT verified, the maintainer's manual checks, still owed"
- cond-2609170842094378 — survived: ChatEntity is a plain AppEntity backed by an EntityStringQuery that filters the live conversations on demand; the delivery carries no IndexedEntity, no donation and no CoreSpotlight use anywhere under client/
  evidence: client/GropiusChat/Intents.swift:81 — "func entities(matching string: String) async throws -> [ChatEntity] {"
  evidence: client/GropiusChat/Intents.swift:62 — "struct ChatEntity: AppEntity {"
- cond-2609170842090438 — survived: the bundle is built for macOS 27 alone and ask() routes through the client's current answerer, connecting to the app's stored server before it will answer with a server model
  evidence: client/build.sh:16 — "TARGET="arm64-apple-macos27.0""
  evidence: client/GropiusChat/GropiusChat.swift:753 — "if case .server = answerer, !connected { await connect() }"
  evidence: client/GropiusChat/GropiusChat.swift:418 — "@AppStorage("serverURL") var serverURL: String = "http://localhost:11535""
## Grounds

- pursued: we expect Shortcuts and Spotlight to list the client's actions from metadata the build script writes with the toolchain's own processor, because that step ran and produced the metadata on this Mac outside Xcode; wrong if the system does not index an ad-hoc-signed zip-installed bundle, or if a long-running prompt intent still times out
