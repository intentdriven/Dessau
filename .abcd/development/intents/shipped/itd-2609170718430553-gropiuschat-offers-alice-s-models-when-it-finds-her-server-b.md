---
id: itd-2609170718430553
slug: gropiuschat-offers-alice-s-models-when-it-finds-her-server-b
spec_id: spc-2609170842083562
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609151701196720, itd-2609170718438919]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# GropiusChat offers Alice's models when Bob looks for them

## Press Release

GropiusChat offers Alice's models when Bob looks for them. Bob is chatting
with the Mac's own model and opens the model picker: beneath "On this Mac" a
section reads "Servers on your network", and as the client finds Alice's
Gropius server it appears there. He picks it, the client asks it for its
models, and her chat models are listed under her server's name. One click and
the conversation carries on where it was, now answered by her model.

The client looks for servers only while the picker is open, so the Mac's
Local Network permission is asked for the first time Bob looks, not the first
time he opens the app. Finding a server is an offer, never a takeover: the
client never switches by itself, and "On this Mac" is always one click away.
Alice sees a models-list request when Bob picks her server, a chat request
once he has chosen a model, and nothing before.

## Why This Matters

The built-in model is the floor; Alice's server is the upgrade — bigger
models, longer windows, the same chat. If the client had to be configured to
find it, most people would never get there. Discovery already exists in the
client, hidden in a Settings pane; putting the offer where Bob is already
choosing a model is what makes the server discoverable in fact.

## Mechanism

We expect the offer to read as part of the app rather than an interruption
because the model picker is a toolbar button that presents a standard popover
holding a `List` with sections, and a popover's presented state is what
starts and stops the Bonjour browse the client already has. What would show
this wrong: a browse that needs longer than the popover stays open to find a
server (then the section needs a "looking…" row and a longer browse), or a
Local Network prompt that macOS raises at launch regardless.

## Scope Conditions

- Discovery runs only while the model picker is open: no browsing at launch, <!-- cond: cond-2609170842082521 -->
  no Local Network prompt before Bob looks. The maintainer's decision at the
  interview, 2026-09-17.
- The server side is untouched: the client uses the models list and the chat <!-- cond: cond-2609170842083752 -->
  endpoint exactly as today; a server's "offer" is what those endpoints
  already publish.
- The 27 client only, building on the built-in default <!-- cond: cond-2609170842089337 -->
  (itd-2609170718438919), whose bar that nothing is sent before Bob chooses
  this intent refines rather than restates.
- The choice of who answers is global to the client, as the model choice is <!-- cond: cond-2609170842085039 -->
  today; switching does not fork the conversation. A transcript that mixes
  both goes to a server whole and to the built-in model as the most recent
  turns that fit.
- The stored API key is sent only to the server it was saved for: a newly <!-- cond: cond-2609170842089482 -->
  found server is asked for its models without a key, a refusal says so and
  points at Settings, and `client/README.md`'s sentence on which machine
  receives the key is rewritten to say this.
- `client/README.md`'s Use section is rewritten: no address is tried at <!-- cond: cond-2609170842082080 -->
  launch, and the Local Network permission is asked the first time the model
  picker is opened. Checking that by hand needs a Mac whose local-network
  grant has been reset; the result is recorded in the shipping decision line.

## Acceptance Criteria

- Given Bob is chatting with the built-in model, when he opens the model
  picker, then the client starts browsing, the picker shows "On this Mac" and
  a "Servers on your network" section that fills as servers are found, and
  the browse stops when the picker closes.
- Given a server appears in the section, when Bob picks it, then the client
  fetches its models list (with the stored API key if there is one), lists
  its chat models under the server's name, and sends no chat request until he
  picks a model.
- Given Bob picks one of Alice's models, when he sends his next message, then
  the whole conversation so far goes to that model and the reply streams as
  it does today, and Alice's log shows that request and nothing earlier.
- Given Bob is on Alice's model, when he picks "On this Mac" again, then his
  next message goes to the built-in model with the most recent turns that fit
  its window, and nothing more is sent to Alice's server.
- Given no server is found, when the picker has been open for a few seconds,
  then the section says "No server found" and the built-in model stays
  selected.
- Given Alice's server leaves the network, when Bob sends, then the failure is
  shown on that message with the picker one click away, and the client does
  not switch by itself.
- Given a server whose models all fall outside the chat rule, when Bob picks
  it, then the section says under its name that none of its models can hold
  a conversation, and the built-in model stays selected.

## Open Questions

- The architecture test that holds "the client opens on the server's own
  default address" keeps its check on the typed address's default; its stated
  reason ("the composer is disabled until a server answers") is no longer
  true and the test's comment says what is.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-f71b7e2f71f1 -->
Fidelity review — receipt rcp-f71b7e2f71f1 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:62833cc0b65936f786e121171ca7d423ffc6ec8e4cd9334fc9d6ff5f48052d9d
Input attestations: diff:acacd07..210ce66 (PR 58, merged as 210ce66; read at origin/main 400ccd7), paths: client internal/archtest install.sh .github README.md docs CHANGELOG.md@-;

Acceptance rollup: MET 5 · MET_WITH_CONCERNS 2 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: The picker is a toolbar-button popover holding a List with an 'On this Mac' section and a 'Servers on your network' section fed by ServerBrowser's @Published servers; the browse starts in .onAppear and stops in .onDisappear, which is exactly the popover's lifetime, and an architecture test holds that shape and forbids any other file starting a browse.
  evidence: client/GropiusChat/GropiusChat.swift:1049 — ".popover(isPresented: $pickerShown) {"
  evidence: client/GropiusChat/Picker.swift:23 — "Section("On this Mac") {"
  evidence: client/GropiusChat/Picker.swift:42 — "Section("Servers on your network") {"
  evidence: client/GropiusChat/Picker.swift:54 — "browser.start()"
  evidence: client/GropiusChat/Picker.swift:61 — "browser.stop()"
  evidence: internal/archtest/chat_client_native_test.go:126 — "if name != "Picker.swift" && strings.Contains(src, "browser.start()") {"
- ac-2 — MET_WITH_CONCERNS: Picking a server resolves it, points the client at it and calls connect(), which is a GET of /models only; the row then lists model.chatModels under the server's own name and a model row merely sets the typed choice, so no chat request is sent until Bob sends. Concern: the criterion says the list is fetched 'with the stored API key if there is one', but request() attaches the bearer token only when the request's origin equals apiKeyHost, so a newly found server is asked WITHOUT a stored key — a divergence the intent's own cond-2609170842089482 signs off.
  evidence: client/GropiusChat/Picker.swift:163 — "model.use(server, resolvedAddress: address)"
  evidence: client/GropiusChat/Picker.swift:165 — "await model.connect()"
  evidence: client/GropiusChat/GropiusChat.swift:683 — "guard var req = request("/models") else {"
  evidence: client/GropiusChat/Picker.swift:136 — "ForEach(model.chatModels, id: \.self) { id in"
  evidence: client/GropiusChat/GropiusChat.swift:675 — "if !apiKey.isEmpty, origin == apiKeyHost {"
- ac-3 — MET_WITH_CONCERNS: stream() hands the backend the whole conversation minus the placeholder reply, ServerBackend maps every message into the request body and streams the SSE deltas back as today, and the only chat POST in the client is the one send() starts. Concern: 'Alice's log shows that request and nothing earlier' is weaker than delivered — AppModel.init re-connects to a stored server at launch and a residency poll re-reads /models every second while a reply is outstanding, so her log carries models-list traffic around the chat request; and the streaming itself could not be exercised here (Swift is not buildable in this checkout).
  evidence: client/GropiusChat/GropiusChat.swift:814 — "let history = messages(in: convoID).filter { $0.id != messageID }"
  evidence: client/GropiusChat/Backends.swift:231 — "let messages: [[String: String]] = history.map {"
  evidence: client/GropiusChat/Backends.swift:282 — "if let c = delta.content, !c.isEmpty { deliver(.text(c, replaces: false)) }"
  evidence: client/GropiusChat/GropiusChat.swift:492 — "if answererKind == "server" { Task { await connect() } }"
  evidence: client/GropiusChat/GropiusChat.swift:785 — "guard var req = request("/models") else { return nil }"
- ac-4 — MET: chooseBuiltIn() sets the single stored answerer back to 'builtin', stream() then selects BuiltInBackend, which builds its transcript from the most recent turns that fit the window it read (oldest dropped first) and imports no networking at all; the residency poll is only armed for a server answerer and is cancelled when the stream speaks, so nothing further goes to the server.
  evidence: client/GropiusChat/GropiusChat.swift:502 — "func chooseBuiltIn() {"
  evidence: client/GropiusChat/GropiusChat.swift:817 — "case .builtIn: backend = BuiltInBackend()"
  evidence: client/GropiusChat/Backends.swift:154 — "static func trimmed(_ history: [Message], model: SystemLanguageModel, budget: Int) async throws -> [Transcript.Entry] {"
  evidence: client/GropiusChat/GropiusChat.swift:743 — "if case .server(let model) = answerer { startResidencyPoll(for: model) }"
  evidence: internal/archtest/chat_client_native_test.go:82 — "for _, s := range []string{"URLSession", "URLRequest", "http"} {"
- ac-5 — MET: The picker arms a five-second timer in .onAppear; until it fires an empty browse shows 'Looking…', and after it the section reads 'No server found', with nothing in the picker touching the answerer, which stays at its 'builtin' default.
  evidence: client/GropiusChat/Picker.swift:56 — "try? await Task.sleep(for: .seconds(5))"
  evidence: client/GropiusChat/Picker.swift:87 — "Text("No server found. A server elsewhere can be typed into Settings.")"
  evidence: client/GropiusChat/GropiusChat.swift:422 — "@AppStorage("answerer") var answererKind: String = "builtin""
- ac-6 — MET: A throwing backend is caught in stream() and its words are appended to that assistant message rather than raised elsewhere; the picker button sits in the toolbar unconditionally and is one click away; no code path writes answererKind or selectedModel on a failure, so the client cannot switch by itself.
  evidence: client/GropiusChat/GropiusChat.swift:846 — "edit { $0.text += (($0.text.isEmpty ? "" : "\n") + "⚠️ " + error.localizedDescription) }"
  evidence: client/GropiusChat/GropiusChat.swift:1035 — "pickerShown = true"
  evidence: client/GropiusChat/GropiusChat.swift:507 — "func chooseServerModel(_ id: String) {"
- ac-7 — MET: connect() fills chatModels by the client's chat rule and deliberately never moves the chosen model; when that list is empty the expanded server row shows, under the server's own name, the sentence that none of its models can hold a conversation, and no answerer change is made.
  evidence: client/GropiusChat/GropiusChat.swift:709 — "chatModels = list.data.filter { rule.offers($0) }.map(\.id).sorted()"
  evidence: client/GropiusChat/Picker.swift:133 — "None of this server's models can hold a conversation; they stay callable over the API by name."
  evidence: client/GropiusChat/Picker.swift:113 — "} label: {"

Gap audit:
- honoured:
  - The offer lives inside the model picker Bob is already using: a toolbar popover with 'On this Mac' above 'Servers on your network'.
    evidence: client/GropiusChat/GropiusChat.swift:1049 — ".popover(isPresented: $pickerShown) {"
    evidence: client/GropiusChat/Picker.swift:42 — "Section("Servers on your network") {"
  - The client looks for servers only while the picker is open, and an architecture test arms that promise rather than leaving it to habit.
    evidence: client/GropiusChat/Picker.swift:61 — "browser.stop()"
    evidence: internal/archtest/chat_client_native_test.go:110 — "func TestChatClientBrowsesOnlyWhileThePickerIsShown(t *testing.T) {"
  - Finding a server is an offer, never a takeover: nothing switches the answerer by itself and 'On this Mac' is one click away.
    evidence: client/GropiusChat/Picker.swift:25 — "model.chooseBuiltIn()"
    evidence: CHANGELOG.md:28 — "model picker offers the Gropius servers it finds on the network, but only"
  - Alice sees a models-list request when Bob picks her server and a chat request once he has chosen a model — the pick path calls connect() only.
    evidence: client/GropiusChat/Picker.swift:165 — "await model.connect()"
    evidence: client/GropiusChat/Backends.swift:226 — "guard var req = request("/chat/completions") else {"
  - The stored API key is bound to one server and an architecture test holds the comparison.
    evidence: client/GropiusChat/GropiusChat.swift:675 — "if !apiKey.isEmpty, origin == apiKeyHost {"
    evidence: internal/archtest/chat_client_native_test.go:137 — "if !regexp.MustCompile(`if !apiKey\.isEmpty, origin == apiKeyHost \{`).MatchString(src) {"
- diverged:
  - Promised (ac-2): the picked server's models list is fetched 'with the stored API key if there is one'. Delivered: the token is attached only when the request's origin equals the origin the key was saved for, so a newly found server is asked without it — the tighter rule cond-2609170842089482 asks for, but not what ac-2 says.
    evidence: client/GropiusChat/GropiusChat.swift:675 — "if !apiKey.isEmpty, origin == apiKeyHost {"
    evidence: client/GropiusChat/GropiusChat.swift:658 — "apiKeyHost = value.isEmpty ? "" : serverOrigin"
  - Promised (cond-2609170842082080, via client/README.md): no address is tried at launch. Delivered: the README says nothing is reached at launch, but AppModel.init still fires connect() against the stored address whenever the last choice was a server, so the claim holds only for a client that has never been pointed at one.
    evidence: client/GropiusChat/GropiusChat.swift:492 — "if answererKind == "server" { Task { await connect() } }"
    evidence: client/README.md:44 — "macOS asks for permission to search the local network the first time the"
- missing:
  - The intent's Open Question: the default-address architecture test was to keep its check but have its comment say what is now true. The comment still gives the retired reason 'the composer is disabled until a server answers', and the file is untouched by the delivered range.
    evidence: internal/archtest/chat_client_discovery_test.go:40 — "address that resolves: the composer is disabled until a server answers, so a"
  - The hand check cond-2609170842082080 promised — the Local Network prompt at the first picker open on a Mac whose grant has been reset, its result recorded in the shipping decision line. The decision line records it as not verified and still owed, so no result exists.
    evidence: .abcd/work/DECISIONS.md:277 — "the Local Network prompt at the first picker open on a Mac whose grant has been reset"

Scope-condition dispositions:
- cond-2609170842082521 — narrowed: The browse limb is delivered and mechanically armed — start in .onAppear, stop in .onDisappear, and a test forbidding any other file from starting a browse — but the 'no Local Network prompt before Bob looks' limb rests on a manual check the shipping decision line records as still owed.
  narrowing: Holds for the browse's lifetime as the source and the architecture test can show it (no browse outside the picker, none at launch); it does not hold as an observed fact about the macOS Local Network prompt, which was never checked on a Mac with a reset grant.
  evidence: client/GropiusChat/Picker.swift:54 — "browser.start()"
  evidence: internal/archtest/chat_client_native_test.go:126 — "if name != "Picker.swift" && strings.Contains(src, "browser.start()") {"
  evidence: .abcd/work/DECISIONS.md:277 — "NOT verified, the maintainer's manual checks, still owed"
- cond-2609170842083752 — survived: No gateway handler changed in the delivered range — the client reads the same GET /models and POST /v1/chat/completions the server already publishes, and the only gateway file the range touches is a test.
  evidence: internal/gateway/gateway.go:135 — "mux.HandleFunc("POST /v1/chat/completions", g.handleCompletions)"
  evidence: client/GropiusChat/Backends.swift:226 — "guard var req = request("/chat/completions") else {"
  evidence: client/GropiusChat/GropiusChat.swift:683 — "guard var req = request("/models") else {"
- cond-2609170842089337 — survived: The change is the macOS 27 client and the architecture tests that read it, and it rests on the built-in default rather than restating it: 'builtin' remains the stored answerer's default and the picker's first section is the Mac's own model.
  evidence: client/GropiusChat/GropiusChat.swift:422 — "@AppStorage("answerer") var answererKind: String = "builtin""
  evidence: client/Info.plist:22 — "< string>27.0< /string>"
  evidence: client/GropiusChat/Picker.swift:29 — "Text(BuiltInBackend.displayName)"
- cond-2609170842085039 — survived: One typed choice for the whole client, stored once; switching rewrites no message, and the same transcript goes whole to a server and trimmed to the most recent fitting turns to the Mac's model.
  evidence: client/GropiusChat/Backends.swift:16 — "enum Answerer: Equatable {"
  evidence: client/GropiusChat/Backends.swift:231 — "let messages: [[String: String]] = history.map {"
  evidence: client/GropiusChat/Backends.swift:152 — "The instructions plus the most recent turns that fit the budget, oldest"
- cond-2609170842089482 — survived: The key is saved against the origin it was entered for and attached to no other, a newly found server is therefore asked without it, the 401 reply names Settings, and client/README.md's sentence on who receives the key now says exactly this.
  evidence: client/GropiusChat/GropiusChat.swift:658 — "apiKeyHost = value.isEmpty ? "" : serverOrigin"
  evidence: client/GropiusChat/GropiusChat.swift:675 — "if !apiKey.isEmpty, origin == apiKeyHost {"
  evidence: client/GropiusChat/Backends.swift:245 — "throw BackendMessage(text: "This server needs an API key; add it in Settings.")"
  evidence: client/README.md:49 — "one; the key is sent only to the server it was entered for, never to a"
- cond-2609170842082080 — narrowed: The Use section is rewritten and states both claims, but one of them is not true of the code for a returning user (init still connects to the stored address when the last choice was a server) and the hand check the condition made the other claim conditional on was never run — the decision line records it as owed rather than recording a result.
  narrowing: Holds as rewritten documentation: the README's 'no address at launch' claim holds only for a client that has not yet been pointed at a server, and the permission sentence stands unverified because the reset-grant check on a Mac was not performed and no result reached the shipping decision line.
  evidence: client/README.md:44 — "macOS asks for permission to search the local network the first time the"
  evidence: client/GropiusChat/GropiusChat.swift:492 — "if answererKind == "server" { Task { await connect() } }"
  evidence: .abcd/work/DECISIONS.md:277 — "the Local Network prompt at the first picker open on a Mac whose grant has been reset"
## Grounds

- pursued: we expect a server offered inside the picker Bob is already using to be found and chosen far more often than one hidden in Settings, because the offer appears at the moment of choosing and costs one click; wrong if the popover's browse is too short to find a server, or if the Local Network prompt at first pick puts people off
