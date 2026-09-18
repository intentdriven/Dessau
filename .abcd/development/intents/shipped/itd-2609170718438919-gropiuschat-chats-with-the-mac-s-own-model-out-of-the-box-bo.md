---
id: itd-2609170718438919
slug: gropiuschat-chats-with-the-mac-s-own-model-out-of-the-box-bo
spec_id: spc-2609170842081563
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609151701196720]
severity: minor
impact: breaking
origin: researcher-authored
production_mode: hand-written
---

# GropiusChat chats with the Mac's own model out of the box

## Press Release

GropiusChat chats with the Mac's own model out of the box. Bob installs the
chat client on a Mac running macOS 27 and starts a conversation at once: no
server to find, nothing to configure. The model picker reads "On this Mac",
and every reply is written by the language model Apple ships with the system,
on the Mac, with nothing sent anywhere. It is the default: the first chat, and
every chat until Bob chooses otherwise, is answered by the Mac itself. The
choice of who answers is one choice for the whole client, as the model choice
is today, and it is a typed choice — the Mac, or a named model on a named
server — never a model name that a server could happen to share.

When the Mac cannot do it — Apple Intelligence is switched off, the model is
still downloading, the Mac is not eligible, or the language is not supported —
the client says which, in plain words, with what would fix it, and offers a
server instead; it never fails silently. Alice, running a Gropius server, sees
no request from the client until Bob chooses one of her models.

## Why This Matters

Today the client is only a window onto a Gropius server: without one it is a
disabled composer and a "not connected" message. A chat client that works the
moment it is opened is what a Mac user expects of a Mac app, and the system
now ships a model every app may use. Making it the default turns the client
from a companion of the server into an app in its own right, and turns Alice's
server into the upgrade — which is the offer the next intent makes.

## Mechanism

We expect the client to answer with no server because the Foundation Models
framework offers the system's on-device model to any app through a session
API that needs no entitlement, no Xcode-only build step and no App Store
distribution, and it streams; verified on the build Mac with a bare `swiftc`
build of a three-line probe. The framework streams cumulative snapshots, not
deltas, so the built-in backend replaces the reply's text with each snapshot
where the server backend appends a chunk. What would show this wrong: the
framework refusing an ad-hoc-signed app, or a chat of ordinary length
overrunning the model's context window so often that keeping only the turns
that fit makes the conversation incoherent.

## Scope Conditions

- macOS 27 on Apple silicon with Apple Intelligence switched on and a <!-- cond: cond-2609170842087565 -->
  supported language: that is where the on-device model exists. Anywhere
  else the client falls back to a server as today, and says why.
- On the device only: the client uses the system's on-device model and never <!-- cond: cond-2609170842084710 -->
  Apple's Private Cloud Compute model, so "nothing sent anywhere" stays true
  and the telemetry decision (adr-2609061503319212) is not crossed. Wrong if
  the framework ever escalates a prompt off the Mac on its own.
- The 27 client only: the kept macOS 26 client keeps its server-first <!-- cond: cond-2609170842088676 -->
  behaviour and never gains this.
- The three-surfaces exception, recorded: a decision line says that chat <!-- cond: cond-2609170842086229 -->
  against the system's own model is client-only functionality with no server
  equivalent and is not a gap, on the kept-release precedent, to be ratified
  as an ADR once `abcd decide` is available. The client calls a system
  service; it hosts no model, and nothing under `internal/` or `cmd/`
  changes.
- Every gate the client has — the composer's enabled state, the send button, <!-- cond: cond-2609170842088141 -->
  the picker's presence — reads the typed backend choice, never the server
  connection, which becomes a property of the server backend alone.
- `client/README.md`'s "until a server answers, the box is greyed out" and <!-- cond: cond-2609170842089840 -->
  first-run-address sentences and `README.md`'s client paragraph are
  rewritten to the built-in default.

## Acceptance Criteria

- Given a Mac where the on-device model is available, when Bob opens the
  client for the first time, then the composer is enabled at once, the model
  picker shows "On this Mac" selected, and his first message is answered as a
  streamed reply; the built-in backend's source holds no URL request and
  imports no networking, held by an architecture test, and the absence of
  traffic is checked by hand once and recorded in the shipping decision line.
- Given a reply is streaming, when Bob presses Stop, then generation stops and
  the partial reply stays in the transcript.
- Given the on-device model is unavailable, when Bob opens the client, then
  the empty state names the reason (Apple Intelligence off, model not ready,
  Mac not eligible, or language not supported) and what would fix it, the
  picker offers a server instead, and nothing is sent.
- Given a conversation longer than the model's context window, when Bob sends
  the next message, then the client sends the instructions and the most
  recent turns that fit, the reply arrives, and no overflow error is shown.
- Given the model declines a prompt, when it does, then the reply row shows
  the model's own explanation as an error state, never an empty bubble.
- Given the built-in model is selected, when the client applies the chat rule
  to the picker, then the built-in entry is exempt from it, and
  `docs/chat-models.md`'s "In the chat client" section says so.

## Open Questions

- The context window's size on a given Mac is read at runtime
  (`contextSize`); Apple's documentation says 4096 tokens and one WWDC26
  sample prints 8192. The client trims by the figure it reads, not a
  constant.
- The refusal path and the context-window trim are checked by hand on the
  maintainer's Mac and recorded in the shipping decision line; the client has
  no test target of its own.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-06e492691f86 -->
Fidelity review — receipt rcp-06e492691f86 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:b08f2872d9a0fac928920a33f502b87dacf5d2d7dcbbd7ad08538f5b041d791d
Input attestations: diff:acacd07..210ce66 -- client internal/archtest install.sh .github README.md docs CHANGELOG.md (origin/main at 400ccd7)@sha256:da0c18bfb59c2c67e4e35a363d5d29a445eb86969c738f5c943f26fd3167cbdb;

Acceptance rollup: MET 1 · MET_WITH_CONCERNS 5 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: The typed choice defaults to the Mac, the picker's first row is 'On this Mac' with a checkmark, the harness on the maintainer's Mac streamed snapshots, and the no-network architecture test exists; but the promised hand check of the absence of traffic appears in the shipping decision line neither as verified nor as owed.
  evidence: client/GropiusChat/GropiusChat.swift:422 — "@AppStorage("answerer") var answererKind: String = "builtin""
  evidence: client/GropiusChat/Picker.swift:23 — "Section("On this Mac") {"
  evidence: internal/archtest/chat_client_native_test.go:82 — "for _, s := range []string{"URLSession", "URLRequest", "http"} {"
  evidence: .abcd/work/DECISIONS.md:277 — "a harness on this Mac streamed nine snapshots from the on-device model"
- ac-2 — MET_WITH_CONCERNS: Stop cancels the reply task and the cancellation branch leaves whatever text was already written in the transcript (only an empty reply is replaced by a 'Stopped' marker); the concern is that nothing exercises it — the client has no test target and the shipping decision line records no Stop check.
  evidence: client/GropiusChat/GropiusChat.swift:746 — "func stop() { replyTask?.cancel() }"
  evidence: client/GropiusChat/GropiusChat.swift:843 — "} catch is CancellationError {"
  evidence: client/GropiusChat/Backends.swift:131 — "try Task.checkCancellation()"
- ac-3 — MET_WITH_CONCERNS: All four unavailability reasons map to a sentence plus a fix, the empty state renders that text with a 'Choose Who Answers…' button into the picker, and canSend is false so nothing is sent; the concern is that no unavailable branch is exercised by any test or by a recorded manual check — they are unreachable on the build Mac.
  evidence: client/GropiusChat/Backends.swift:99 — "case .appleIntelligenceNotEnabled:"
  evidence: client/GropiusChat/GropiusChat.swift:522 — "if let why = builtInUnavailable { return why.reason + " " + why.fix }"
  evidence: client/GropiusChat/GropiusChat.swift:1061 — "EmptyState(model: model, choose: { pickerShown = true })"
  evidence: client/GropiusChat/Picker.swift:31 — "Text(why.reason).font(.caption).foregroundStyle(.secondary)"
- ac-4 — MET_WITH_CONCERNS: The transcript is rebuilt each reply from the instructions plus the most recent turns that fit the runtime contextSize less a reserve, with one retry on contextSizeExceeded, and the decision line records a harness trimming a long transcript to its budget; the concern is that the budget counts only the prior turns — the new prompt's own tokens are not counted (Backends.swift:126), so an overflow is caught by the single retry rather than prevented, and the check was a harness rather than the shipped app.
  evidence: client/GropiusChat/Backends.swift:154 — "static func trimmed(_ history: [Message], model: SystemLanguageModel, budget: Int) async throws -> [Transcript.Entry] {"
  evidence: client/GropiusChat/Backends.swift:137 — "case .contextSizeExceeded where attempt == 0:"
  evidence: .abcd/work/DECISIONS.md:277 — "trimmed a long transcript to its budget"
- ac-5 — MET_WITH_CONCERNS: A .refusal is unwrapped to the model's own explanation and surfaced as an error row appended to the reply text, so the bubble is never empty; the concern is that the recorded manual check covered the guardrailViolation mapping, not the .refusal explanation path the criterion names, and an empty explanation silently falls back to the client's own sentence.
  evidence: client/GropiusChat/Backends.swift:185 — "case .refusal(let refusal):"
  evidence: client/GropiusChat/Backends.swift:187 — "return why.isEmpty ? "The Mac's own model declined to answer." : why"
  evidence: client/GropiusChat/GropiusChat.swift:845 — "+ "⚠️ " + error.localizedDescription"
  evidence: .abcd/work/DECISIONS.md:277 — "mapped a guardrail refusal to a sentence"
- ac-6 — MET: The built-in row is its own picker section outside the rule-filtered served models, which alone are passed through rule.offers, and the docs section says so in as many words.
  evidence: client/GropiusChat/Picker.swift:23 — "Section("On this Mac") {"
  evidence: client/GropiusChat/GropiusChat.swift:709 — "chatModels = list.data.filter { rule.offers($0) }"
  evidence: docs/chat-models.md:48 — "model and is exempt from the rule: it is offered whenever the Mac can run it."

Gap audit:
- honoured:
  - The client answers with the Mac's own model by default: one typed choice, defaulting to the built-in backend, read by the whole client.
    evidence: client/GropiusChat/Backends.swift:16 — "enum Answerer: Equatable {"
    evidence: client/GropiusChat/GropiusChat.swift:422 — "@AppStorage("answerer") var answererKind: String = "builtin""
  - The built-in backend imports no networking and builds no request, held by an architecture test in the server's suite.
    evidence: internal/archtest/chat_client_native_test.go:67 — "func TestChatClientBuiltInBackendTouchesNoNetwork(t *testing.T) {"
  - The picker reads 'On this Mac' and the built-in entry is exempt from the chat rule, with the docs sentence to match.
    evidence: client/GropiusChat/Backends.swift:67 — "static let displayName = "On this Mac""
    evidence: docs/chat-models.md:48 — "is not a served"
  - The prose was rewritten to a client that works without a server, and the changelog carries impact: breaking.
    evidence: client/README.md:3 — "A native macOS chat app. It chats with the Mac's own model out of the box —"
    evidence: README.md:170 — "client (`GropiusChat.app`) **Requires macOS 27** and runs on Apple Silicon,"
    evidence: CHANGELOG.md:20 — "(`impact: breaking` for the client:"
  - The three-surfaces exception is recorded as a dated decision line, to be ratified as an ADR.
    evidence: .abcd/work/DECISIONS.md:276 — "Chat against the system's own model is client-only functionality with no server equivalent, and it is not a gap."
- diverged:
  - The refusal path was to be checked by hand and recorded; what the shipping decision line records instead is the guardrailViolation mapping, a different branch of the error switch from the .refusal explanation the criterion names.
    evidence: .abcd/work/DECISIONS.md:277 — "mapped a guardrail refusal to a sentence"
    evidence: client/GropiusChat/Backends.swift:183 — "case .guardrailViolation:"
  - The context-window trim was verified by a harness exercising the logic on the maintainer's Mac rather than by the shipped client, and the trim budget counts the prior turns only, leaving the single retry to absorb a prompt that overflows on its own.
    evidence: client/GropiusChat/Backends.swift:126 — "let entries = try await Self.trimmed(prior, model: model, budget: model.contextSize - reserve)"
    evidence: .abcd/work/DECISIONS.md:277 — "trimmed a long transcript to its budget"
- missing:
  - The hand check of the absence of traffic, recorded in the shipping decision line: the line's verified list does not include it and its owed list does not name it either, so the 'nothing sent anywhere' headline rests on the source-reading architecture test alone.
    evidence: .abcd/work/DECISIONS.md:277 — "NOT verified, the maintainer's manual checks, still owed:"
    evidence: internal/archtest/chat_client_native_test.go:73 — "if strings.Contains(src, "import Network") {"

Scope-condition dispositions:
- cond-2609170842087565 — survived: The client's floor is macOS 27 on Apple Silicon and everywhere the model does not exist the availability reading falls back to a server with a reason, as the condition assumed.
  evidence: client/Info.plist:22 — "< string>27.0< /string>"
  evidence: client/GropiusChat/Backends.swift:87 — "static func unavailability() -> Unavailable? {"
- cond-2609170842084710 — narrowed: The built-in backend uses SystemLanguageModel.default and reaches for no Private Cloud Compute API, and the architecture test forbids networking in that type — but no run of the app was watched for traffic, so nothing delivered speaks to the framework escalating a prompt on its own.
  narrowing: Holds as a source-level property of the client — BuiltInBackend requests only the default on-device model and mentions no URL, request or Network import — not as an observed absence of traffic; the framework-side escalation the condition names as its falsifier is unverified because the hand check was never recorded.
  evidence: client/GropiusChat/Backends.swift:88 — "let model = SystemLanguageModel.default"
  evidence: internal/archtest/chat_client_native_test.go:82 — "for _, s := range []string{"URLSession", "URLRequest", "http"} {"
  evidence: .abcd/work/DECISIONS.md:277 — "NOT verified, the maintainer's manual checks, still owed:"
- cond-2609170842088676 — survived: The macOS 26 client is a frozen v0.6.0 build the installer fetches by the Mac's version, so it gains nothing from this change, and a test holds the tag and the two floors together.
  evidence: install.sh:101 — "KEPT_CLIENT_TAG=v0.6.0"
  evidence: internal/archtest/minimum_macos_test.go:41 — "installerKeptClientTag = "KEPT_CLIENT_TAG""
- cond-2609170842086229 — survived: The dated decision line records the exception on the kept-release precedent, and within the audited surface the only addition under internal/ is an architecture test — no server package or cmd/ code changed for this intent.
  evidence: .abcd/work/DECISIONS.md:276 — "to be ratified as an ADR once `abcd decide` is available"
  evidence: internal/archtest/chat_client_native_test.go:67 — "func TestChatClientBuiltInBackendTouchesNoNetwork(t *testing.T) {"
- cond-2609170842088141 — survived: cannotSend switches on the typed answerer and only its .server branch consults the connection, the composer and send button read canSend, and the picker's toolbar button is present unconditionally.
  evidence: client/GropiusChat/GropiusChat.swift:518 — "var cannotSend: String? {"
  evidence: client/GropiusChat/GropiusChat.swift:531 — "var canSend: Bool { !sending && cannotSend == nil }"
  evidence: client/GropiusChat/GropiusChat.swift:1094 — ".disabled(!model.canSend)"
- cond-2609170842089840 — survived: The greyed-out and first-run-address sentences are gone from client/README.md and both READMEs now open on a client that chats with the Mac's own model out of the box.
  evidence: client/README.md:3 — "It chats with the Mac's own model out of the box —"
  evidence: client/README.md:32 — "Launch it and type. The picker in the toolbar reads **On this Mac**"
  evidence: README.md:171 — "It chats with the Mac's own model out"
## Grounds

- pursued: we expect a client that answers with no server to be the thing that makes the client an app rather than a window, and the system's on-device model to be good enough for the first conversation, because it needs no entitlement and streams from a bare swiftc build (verified on this Mac); wrong if people turn it off for a server at once because its context or its refusals get in the way of ordinary chat
