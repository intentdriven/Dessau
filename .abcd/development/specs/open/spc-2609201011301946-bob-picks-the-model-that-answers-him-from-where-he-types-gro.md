---
id: spc-2609201011301946
slug: bob-picks-the-model-that-answers-him-from-where-he-types-gro
intent: itd-2609200829199959
origin: researcher-authored
production_mode: hand-written
---

# The model picker at the composer, in the chat client

## Summary

This spec delivers itd-2609200829199959 in the chat client only. The composer
becomes a pill that floats over the transcript, and the control that decides
who answers sits at its start, carrying the model's name and where it runs.
Its popover lists **On this Mac** first and then one section per Gropius
server found on the network, each listing that server's chat models, with a
lock where a key is wanted, a greyed **No response** where a server has gone
quiet, and a paired server shown as paired. A server appears once, matched by
the certificate's spki fingerprint where TLS is on and by the instance name
otherwise. A welcome sheet greets Bob once per install and selects nothing.
The first time servers appear, one dismissible line above the composer offers
one of them. Settings holds a default and a fallback and nothing list-shaped.
A fallback happens only between turns: server to this Mac announces itself in
the transcript with a retry, this Mac to a server always asks. The menu bar
gains a **Model** menu that mirrors the picker. Nothing changes on the server,
the control panel or `config.json`. Impact: additive.

## Scope

In: `client/GropiusChat/` (the composer, the picker, discovery, the backends,
the conversation record, the app's commands and Settings), three new
SwiftUI-free files that hold the logic the unit tier tests, `client/tests/`
(the Swift unit tier), a new XCUITest bundle under `client/`,
`internal/archtest/`, and `client/README.md`. Neither build script needs
editing: both compile `GropiusChat/*.swift` as a glob, so a new source file is
picked up by itself.

Out: the server, the control panel, `config.json`, the Discord bridge (its
`/model` stays a per-channel choice), the three answer styles
(itd-2609200850330402), the chat backgrounds (itd-2609200854205251), the
mark's three forms (itd-2609200827202340), and the server's stable per-install
identity (iss-2609200830185068), which this intent neither waits for nor
blocks on.

### What the mock's boards show

The Model Picker Mock of 2026-09-20 is the maintainer's evidence and is not in
the repository; the six boards are described here from the intent and from the
2026-09-20 decision lines that record what was chosen on them. Board 1 is the
first launch with the offer line. Board 2 is the pill composer with the picker
at its start, chosen over board 6B's toolbar picker. Board 3 is the announced
fallback as a row in the transcript. Board 4 is the four states the on-device
row can be in — available, not eligible, not switched on, not ready yet.
Board 5 is Settings with the default and the fallback and nothing else. Board
6 is the narrow-window collapse, board 6A being the server glyph that was
chosen. Where this spec and a board would disagree, this spec is what is
built and the disagreement is a question for the maintainer, not a licence.

## Approach

### The welcome sheet

A new file `client/GropiusChat/Welcome.swift` declares `struct WelcomeSheet:
View`. `RootView` presents it with `.sheet(isPresented:)` bound to
`@AppStorage("hasSeenWelcome")` inverted; the one button sets the flag and the
sheet closes. There is no second path that sets or clears it, and Settings
carries no control that names it. The sheet holds a title, two lines and one
button, and no model control of any kind:

- Title: **Welcome to Gropius Chat**
- "Type a question and a language model answers. That is the whole of it."
- "This \(deviceNoun) can already answer, on the device, with nothing sent
  anywhere. When a Gropius server turns up on your network, its models are one
  click away in the picker beside the message box."
- Button: **Start Chatting**

`deviceNoun` is the client's existing noun, so the second line reads correctly
on the Mac and on the iPad. The sheet gates nothing: it is presented over a
chat that is already open on this Mac's model, and dismissing it leaves Bob in
that chat. On a \(deviceNoun) whose own model is unavailable the second line
is replaced by the on-device row's own reason and fix, so the sheet never
promises an answerer that cannot answer.

### The pill composer and the picker control

The composer moves out of a bar beneath the transcript and becomes a pill
floating over it: the transcript scrolls under it with a bottom safe-area
inset the pill's height, and the pill is drawn in the system's own material in
`Composer.swift`. `Composer.swift` is already the no-styling rule's third
named exception, and the pill's container is drawn there so the exception
stays one file wide and no fourth exception is added by this intent. (The
answer-styles spec adds `Styles.swift` as its own named exception; the two
specs do not collide.)

At the start of the pill sits `ModelPickerControl`, a standard `Button` that
presents the picker's popover. Expanded it carries two lines: the model's
name, from `Answerer.displayName`, and where it runs — **On this Mac**, or the
server's name. Collapsed it carries one glyph and a chevron.

The collapse rule is a stated width, not `ViewThatFits`, because both the
architecture test and the XCUITest need a number to hold: a `GeometryReader`
over the pill reports its width, and the label is dropped below
`ModelPickerMetrics.labelFloor`, **480 points**. The Mac's window minimum is
720 points wide, so Carol's narrow window beside her editor, with the sidebar
shown, gives the pill about 460 points and the control is collapsed; with the
sidebar hidden it is about 700 and the label shows. On the iPad a Split View
share collapses it the same way.

Collapsed, the glyph is `network` for a server and `apple.intelligence` for
this \(deviceNoun) — the intent's "Apple mark" is read as the symbol the
client already uses for its own model, in the toolbar and on a conversation
card, rather than as the Apple logo, which is a trademark and would read as a
second meaning. Each carries `chevron.up.chevron.down`; no circle glyph appears
anywhere in the control, because the circle is the Circle mark's
(itd-2609200827202340). The model's name is in `.help()` and in
`.accessibilityLabel()`, and nothing is drawn on hover — there is no
`.onHover` in the control — so the same control works under a finger on the
iPad. The tooltip is the system's own help and is not a change of layout.

The toolbar loses its picker. The window's title is the conversation's title,
never the app's name. The top right of the window belongs to
itd-2609200850330402's three answer styles; the two controls stay
distinguishable at every width because they are different shapes in different
places — a glyph with a chevron at the start of the pill, three marks in a
segmented control at the top right — and neither ever collapses into the
other's form.

### The picker's popover

`ModelPickerView` is rewritten around a list of sections built by a pure
function rather than around the one server the client happens to be pointed
at. The first section is **On this Mac**. Then, where Bob has starred models,
a **Favourites** section. Then one section per deduplicated server, in the
browse's own order.

A server's section header carries the server's name, its advertised summary,
and one glyph: `lock` when the advertisement says `auth=bearer`,
`lock.shield.fill` when this client is paired with it, `network` otherwise.
The lock and the pairing glyph stay the two distinct claims they already are
in this file.

The rows under a section are that server's chat models. Today the client can
only list the models of the single server it is connected to; that is what
lists one server's models under another's name and what makes the stored-server
row a duplicate. A new `ServerCatalogue` actor replaces it: keyed by server
identity, it resolves a server and fetches `GET <base>/v1/models` for it, with
that origin's key or with none, and caches the answer for the life of the
popover. Sections appear immediately from the advertisement; a section's model
rows are fetched when the section is first shown. The whole of resolve and
fetch is given **3 seconds** — `ServiceResolver.resolve(timeout: 1.5)`, which
is shorter than its 5-second default because a popover cannot wait that long,
and a 1.5-second `timeoutInterval` on the models request. Past it the section
is greyed, reads **No response**, and stays listed — it is never dropped, on the Home app's
precedent. Reopening the picker retries. A server that answers but serves no
chat model says so in its own section, in the words the file already uses.

Stars pin favourites: a row's context menu offers **Add to Favourites** /
**Remove from Favourites**, and a starred model is drawn with
`star.fill`. Favourites are stored in `@AppStorage("favouriteModels")` as
newline-separated `origin<TAB>model` keys — the origin is part of the key
because a model id means nothing without the server that serves it.
Unstarring returns the model to its server's section and to nowhere else.

Choosing a row sets the chat's answerer and closes the popover. A row's
context menu also offers **Set as Default**, which writes the same choice to
`@AppStorage("defaultAnswerer")`.

The per-chat choice is stored on the conversation. `Conversation` gains
`var answerer: StoredAnswerer? = nil`, where `struct StoredAnswerer: Codable,
Equatable { var origin: String?; var model: String? }` and a nil origin means
this \(deviceNoun). It is optional so a `conversations.json` written before
this change decodes unchanged. A new chat starts on the default; changing the
model inside a chat changes that chat and nothing else. The single
`@AppStorage("selectedModel")` that stood for "the model, for the whole
client" is retired in favour of the per-chat field and `defaultAnswerer`;
pre-1.0, nothing migrates it.

### Server identity, deduplication and the paired row

`DiscoveredServer` starts reading two TXT keys the server already publishes
and this client has been ignoring: `spki`, the certificate's fingerprint, and
`tlsport`. Identity is a new SwiftUI-free type:

    enum ServerIdentity: Hashable { case fingerprint(String), instance(String) }

`.fingerprint` where the advertisement carries a non-empty `spki`,
`.instance(name)` otherwise — the interim rule of cond-2609201011309289, held
until iss-2609200830185068 gives the server an identity of its own. The
instance name is taken as Bonjour gives it, suffix and all: stripping a
`(2)` would risk folding two genuinely different servers into one row, which
the scope condition rules out more firmly than it rules out a duplicate.

`deduplicated(_:)` keeps one advertisement per identity, preferring the one
whose TXT says most (a fingerprint, then a model count), tie-broken by the
shorter name. This is what collapses a Bonjour rename and a stale twin left by
a restart.

The stored-server row goes away as a row of its own, which is
iss-2609200822241060 fixed as a consequence rather than claimed as a criterion
here. The client's stored address, its connected state and its pairing are
carried **on** the matching discovered section: matched by the pinned spki
where the client is paired and the advertisement carries one, by the resolved
origin where the client has resolved it, and by instance name otherwise. The
bare address row appears only when no discovered section matches it — the case
it was written for, a server the browse cannot see.

### The three on-device states

`BuiltInBackend.unavailability()` already returns a reason and a fix for each
of the framework's three cases. This spec holds that shape and sharpens what a
fix is: **not switched on** and **not ready yet** have a fix for this
\(deviceNoun) and offer it; **not eligible** has none, and its row offers only
the alternative and says plainly that this \(deviceNoun) cannot run the
system's own model. The three texts stay distinct, stay the client's own
words, and never show the framework's error text — the side already settled on
iss-2609181116081228 at the 2026-09-20 interview, which this spec refines and
does not reopen. The on-device row is rendered in every state, disabled rather
than absent, so the reason is always readable.

### The offer line

`ServerOfferBar` sits directly above the pill. It is shown when servers have
appeared for the first time — `@AppStorage("hasSeenAServer")` is false and the
browse has returned at least one advertisement that speaks this client's API —
and `@AppStorage("serverOfferDismissed")` is false. It names one server, the
first in the deduplicated browse order that speaks this client's API, with its
model count from the advertisement, and offers three things:

- **Use it** — picks that server's default model for this chat and closes the
  line. If the server wants a key, the same key sheet the picker already shows
  asks for one; no stored key is reused.
- **See all** — opens the picker's popover and closes the line.
- A dismissal (`xmark`) — closes the line for good.

All three set `serverOfferDismissed`, so the line is shown once and never
returns. Nothing changes model until Bob clicks. This is the whole of the
tour: there is no onboarding sequence, and the welcome sheet says nothing
about servers being present because at first launch they are not yet news.

**This widens the browse, and that is a change to a shipped promise.** Today
the browse runs only while the picker's popover is shown, which is held by
`TestChatClientBrowsesOnlyWhileThePickerIsShown` and stated in
`client/README.md` as the reason the system asks for local-network permission
at the first picker open rather than at launch. A line that appears "the
afternoon Alice switches her server on" cannot wait for Bob to open a picker
he has never opened. So the rule becomes: the browse runs while the picker is
shown, **or** while the offer is still undismissed. Once
`serverOfferDismissed` is set — which every path out of the line sets, on the
first day of use — the old rule is back for the rest of the install's life.
The consequences, each of which is part of this work rather than a surprise
during it: the existing architecture test is amended to the two-condition rule
and watched red first; the permission prompt moves to first launch for a new
install; and `client/README.md`'s sentence about when the Mac asks is
rewritten to say so.

### Settings

A new `Section("Model")` holds exactly two controls:

- **Default for a new chat** — a pop-up over this \(deviceNoun) and the models
  the catalogue knows, writing `@AppStorage("defaultAnswerer")`.
- **If the server cannot be reached, answer on this \(deviceNoun)** — a
  toggle, `@AppStorage("fallbackToBuiltIn")`, **on** by default.

Nothing ranked or reorderable exists anywhere in the client: no `.onMove(`, no
list of servers to build, and no list to maintain when a server is retired.
The existing `Section("Models to offer")` is a different setting — which of a
server's models count as chat models — and is untouched; it is not a list of
servers and holds no order.

### Fallback

The decision is a pure function in a SwiftUI-free file, called once per turn,
before the first token, and nowhere else:

    func decideAnswerer(chosen:reachable:fallbackEnabled:isStreaming:) -> AnswerDecision

with `AnswerDecision` one of `.use(chosen)`, `.fallBack(to: .builtIn,
announcing: server)`, `.ask(server)`, `.failTurn(retryable: true)`.

- **Server to this \(deviceNoun)**: with the fallback on and the server
  unreachable at the moment Bob sends, the turn is answered here, and a row is
  written into the transcript naming the server that failed and offering
  **Retry on <server>**. It announces and continues; it does not ask.
- **This \(deviceNoun) to a server**: never by itself. The decision is
  `.ask`, and nothing moves until Bob answers. A conversation held on this
  \(deviceNoun) is never moved to somebody else's machine without his say.
- **Mid-stream**: `isStreaming` forces `.failTurn`. The turn ends as a failed
  turn Bob can retry, and no reply ever carries text from two answerers. The
  decision function is not called from inside the streaming loop at all, which
  is what the architecture test holds.

While a chat is answering somewhere other than its chosen model, the picker
control is tinted — `.tint(.orange)` — and its accessibility label says so.
`.tint(` is not one of the modifiers the no-styling rule names, so this adds
no exception.

### The menu bar's Model menu

`enum PickerAction: String, CaseIterable` names what the picker can do:
`onThisDevice`, `chooseModel`, `setAsDefault`, `toggleFavourite`. A new
`CommandMenu("Model")` carries one `Button` per case; the existing **Choose
Model…** moves out of the Chat menu into it and keeps Cmd-Shift-M. The menu
reaches the front window through the client's existing seam — `ChatActions`,
published as a focused scene value — which gains one closure per action, so
the menu decides nothing of its own and each button calls the same action the
popover calls. A capability in one and not the other is a defect the
architecture test names.

### The client README

`client/README.md` is rewritten where it describes the picker: the control at
the start of the message pill rather than in the toolbar, the welcome sheet,
the offer line, the two Settings controls, the announced fallback with its
retry, and the collapse on a narrow window. Every sentence describing a
toolbar picker goes, on the Mac and in the iPad section. Present tense,
British English, no history.

### Where the logic lives

Three new SwiftUI-free files, so the Swift unit tier can compile and test them
alone as `client/tests/` already does for the card summary, the sidebar search
and the context budget:

- `PickerModel.swift` — `ServerIdentity`, `deduplicated(_:)`,
  `ModelOrdering.sections(...)`, `PickerAction`.
- `Answering.swift` — `AnswerDecision` and `decideAnswerer(...)`.
- `Requests.swift` — `authorisedRequest(base:path:origin:key:keyOrigin:)`,
  the one place an `Authorization` header is ever attached.

Each gets a test file and a shell wrapper in `client/tests/`, on the pattern of
`card-summary.sh`, and a runner in `internal/archtest/` that skips loudly where
no toolchain is present. `TestChatClientCardSummaryCompilesAlone`'s rule
applies to all three: the file mentions no `import`, no `SwiftUI`, and neither
`Conversation` nor `Message`, so it compiles standalone. Neither build script
changes — both glob `GropiusChat/*.swift`.

## Acceptance-criteria map

The intent's criteria carry no minted ids — `abcd` stamps scope conditions,
not criteria — so they are numbered here in the order the intent lists them,
which is the order `abcd intent ready` counts. Every test named below is
written and watched to fail before the change that makes it pass. The XCUITest
tier is the one decided for iss-2609181116225273 on 2026-09-20: a bundle under
`client/`, run only when a change touches `client/**`, on a CI path filter and
on the same rule as a script argument locally.

| # | Criterion | Held by |
|---|---|---|
| 1 | Clean install: welcome sheet with no model control; dismissing leaves an open chat reading **On this Mac**, nothing blocking the first message | XCUITest `WelcomeSheetTests.testFirstLaunchGreetsAndGatesNothing` on a reset simulator; archtest `TestChatClientWelcomeSheetSelectsNothing` — the `WelcomeSheet` block names no `Picker(`, no `ModelPickerView`, and no chooser call |
| 2 | Dismissed once, never again; no setting brings it back | XCUITest `WelcomeSheetTests.testSecondLaunchIsSilent` across two launches; archtest `TestChatClientWelcomeSheetIsShownOnOneStoredFlag` — exactly one `@AppStorage("hasSeenWelcome")`, the sheet presented on it, the button setting it, and `SettingsView` naming it nowhere |
| 3 | Servers first appear: exactly one line above the composer naming one, with **Use it**, **See all** and a permanent dismissal; no model changes until Bob clicks | archtest `TestChatClientOffersTheFirstServerItSees` over `ServerOfferBar` — the three actions, the dismissal flag, and no chooser call outside the three buttons; the amended `TestChatClientBrowsesOnlyWhileThePickerIsShown` for the widened browse; hand check HC-2 |
| 4 | The picker sits at the start of the message pill; no model picker in the toolbar | archtest `TestChatClientPickerSitsAtTheStartOfTheComposer`, on `TestChatClientComposerHasMessagesForm`'s pattern — `ModelPickerControl(` precedes the `TextField` in the composer block, and the toolbar block names no picker |
| 5 | Narrowed: glyph or Apple mark with a chevron and no dot, name in tooltip and accessibility label, nothing on hover | XCUITest `PickerTests.testCollapsesOnANarrowWindow` for the drawn collapse; archtest `TestChatClientCollapsedPickerNamesNoCircle` — the named `labelFloor` constant, `.help(`, `.accessibilityLabel(`, no `circle` symbol and no `.onHover` in the control |
| 6 | **On this Mac** first, one section per server listing only its chat models, lock where a key is wanted, greyed **No response** kept listed, paired shown as paired | archtest `TestChatClientPickerListsOneSectionPerServer` over `Picker.swift` — the section order, the three glyphs distinct, the **No response** text and the 3-second constant; hand check HC-3 |
| 7 | Two starred models sit at the top in a favourites group; unstarring returns one to its server's section | Swift unit `ModelOrderingTests` over `ModelOrdering.sections(...)`; XCUITest `PickerTests.testFavouritesGroup` for the drawn group |
| 8 | Settings' model section holds exactly two controls; no ranked or reorderable list anywhere | archtest `TestChatClientSettingsHoldsOneDefaultAndOneFallback` — the `Section("Model")` block holds exactly two controls, the toggle defaults to true, and no `.onMove(` or server `ForEach` exists in `SettingsView` |
| 9 | Server unreachable at send, fallback on: answered here, a transcript row naming the server with a retry, the picker tinted | Swift unit `AnsweringTests.testServerToBuiltInAnnouncesAndContinues`; XCUITest `FallbackTests.testTranscriptRowOffersRetry` |
| 10 | Mid-stream failure: no switch, a retryable failed turn, never two answerers in one reply | Swift unit `AnsweringTests.testMidStreamNeverSwitches`; archtest `TestChatClientNeverFallsBackInsideTheStreamingLoop` — the streaming block contains no call to `decideAnswerer` |
| 11 | This Mac to a server asks first; server to this Mac announces and continues | Swift unit `AnsweringTests` over both directions |
| 12 | On-device unavailable: the right one of three reasons in the client's own words, a fix for the two that have one, the row never silently absent | archtest `TestChatClientNamesThreeOnDeviceReasons` over `Backends.swift` — three distinct reasons, a fix for `appleIntelligenceNotEnabled` and `modelNotReady`, the alternative only for `deviceNotEligible`, and the row rendered-and-disabled rather than conditional; hand check HC-5 |
| 13 | A key stored for one server never reaches a newly found one; the client asks for its own | Swift unit `RequestsTests.testForeignOriginCarriesNoAuthorization` over `authorisedRequest(...)`; the existing archtest on the origin check |
| 14 | Two advertisements for one server — a rename, a stale twin — list as one row | Swift unit `ServerIdentityTests` over `deduplicated(_:)` with both advertisement shapes, fingerprint and instance name |
| 15 | The **Model** menu offers every model and every action the picker does | archtest `TestChatClientModelMenuMirrorsThePicker` — every `PickerAction` case has a `Button` in the `CommandMenu("Model")` block and no action exists in only one place; hand check HC-4 |
| 16 | `client/README.md` describes the picker at the composer, the sheet, the offer line, the two settings and the announced fallback, and no toolbar picker | archtest `TestTheChatClientReadmeDescribesThePicker`, on the docs tests' pattern — the new helper reads `client/README.md` as `readDoc` reads a page, asserts each promise and asserts the toolbar-picker sentences are gone |

### Named hand checks

- **HC-1 First launch.** A reset container: the sheet, the dismissal, the
  first message answered on this Mac.
- **HC-2 The offer line.** The maintainer's own network with a second Mac
  running Gropius: the line appears once, **Use it** switches the chat, the
  dismissal is permanent across a relaunch.
- **HC-3 A server switched off while the picker is open.** Its section greys
  to **No response** and stays listed.
- **HC-4 The Model menu against an open picker.** Every action in one is in
  the other, and picking in either does the same thing.
- **HC-5 Apple Intelligence switched off.** The on-device row's reason and its
  fix.
- **HC-6 The stale twin.** A server restarted without deregistering, and a
  second server given the same name so Bonjour renames it: one row each.

Each hand check is recorded in the shipping decision line, as the client's
hand checks already are — the client has no test target that can assert a
drawn appearance outside the XCUITest tier.

## Security

- **A key is bound to the origin it was saved for.** `authorisedRequest(...)`
  is the only place an `Authorization` header is attached, and it attaches one
  only when the request's origin equals the origin the key was stored for —
  the behaviour iss-2609181116082211 settled, now enforced at one seam instead
  of at a call site. The per-server sections, the catalogue's model fetches,
  the offer line's **Use it** and the fallback path all go through it, so none
  of them can leak a key to a newly found server. A server that wants a key is
  asked for its own, in the sheet the picker already shows.
- **The pairing stays bound to its origin.** The pinned spki is used only for
  the origin the pairing was made with; matching a discovered advertisement to
  the stored pairing by fingerprint is a read of the advertisement and grants
  no trust by itself — the TLS challenge still pins.
- **Nothing new leaves the device.** The welcome sheet, the offer line, the
  favourites and the two settings are local state. The browse is mDNS on the
  local link, as it already is, and still runs only while the picker or the
  offer's check is live. The catalogue adds one `GET /v1/models` per server
  section the person opens, to servers that advertised themselves on the local
  network, and sends no prompt text. `BuiltInBackend` still imports no
  networking and still builds no request, which its existing architecture test
  holds.
- **No new secret is stored.** The favourites and the default are plain
  preferences and carry no key material; the origin in a favourite key is an
  address already in the preferences.

## Verification

- `client/build.sh` builds the Mac app; `SIM=1 client/build-ipad.sh` builds
  and launches the iPad app in the simulator. Both must build with the new
  files listed.
- `make test` (the Go suite, which is where the architecture tests live),
  `gofmt -l .` empty, `go vet ./...` clean.
- The Swift unit tier: the scripts in `client/tests/`, one per new
  SwiftUI-free file, run on every change.
- The XCUITest tier on a reset simulator, run when a change touches
  `client/**`.
- The six named hand checks above, on the maintainer's own Mac and iPad, with
  a second Mac on the network for HC-2, HC-3 and HC-6, recorded in the
  shipping decision line.

## Out of scope

- The three answer styles and the segmented control at the top right
  (itd-2609200850330402, spc-2609200945518522). This spec touches neither the
  styles nor the reply labels; it only keeps the two controls apart.
- The server's stable per-install identity (iss-2609200830185068). The interim
  rule stands until it lands; when it does, `ServerIdentity` gains a case and
  the deduplication function is the one place that changes.
- Chat backgrounds (itd-2609200854205251) and the mark's three forms
  (itd-2609200827202340). The picker's collapsed glyph is deliberately not the
  mark's circle.
- Any server, control-panel or `config.json` change. The Discord bridge keeps
  `/model`.
- No migration of `conversations.json`: the new field is optional and an older
  file decodes unchanged, which is cheaper than a migration and not one.
- No **Auto** router across servers, no ranked fallback array, no reopening of
  the on-device availability message beyond the three states named here.

## Risks and preconditions

- **The XCUITest tier does not exist yet.** It was decided on 2026-09-20 for
  iss-2609181116225273 and no bundle is in the repository. Seven of the
  sixteen criteria name it, so standing it up — the bundle, the narrowed
  scope condition the decision describes, and the `client/**` path filter in
  CI — is a precondition of this intent, not a part of it. If it is still
  absent when this is built, it is built first or those criteria are held by
  recorded hand checks and the record says so.
- **The intent's "Apple mark" is read as `apple.intelligence`.** That is the
  symbol the client already uses for its own model in the toolbar and on a
  conversation card, and it carries no trademark question, which
  `apple.logo` would. If the maintainer meant the logo, one symbol name
  changes and nothing else in this spec does.
- **The offer line widens the browse.** It moves the local-network permission
  prompt to first launch for a new install and amends an architecture test and
  a README sentence, as the offer-line section sets out. The alternative —
  showing the offer only once Bob has opened the picker — was rejected because
  it contradicts the press release, where the line arrives the afternoon a
  server appears.
