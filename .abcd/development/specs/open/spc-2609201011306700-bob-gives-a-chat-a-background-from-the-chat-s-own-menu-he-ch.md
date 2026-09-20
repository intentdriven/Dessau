---
id: spc-2609201011306700
slug: bob-gives-a-chat-a-background-from-the-chat-s-own-menu-he-ch
intent: itd-2609200854205251
origin: researcher-authored
production_mode: hand-written
---

# A background per chat, in the chat client

## Summary

This spec delivers itd-2609200854205251 in the chat client alone. A chat gets
a background: None, one of the built-in colours, gradients and dynamic
backgrounds the client carries, or a picture Bob hands it through the system's
limited-access photo picker. The choice is made from the Chat menu, stored on
the conversation record, and remembered across a relaunch; Settings holds the
background a new chat starts with and changes no chat that exists. A picture is
copied into the client's own container and referenced by file name; a reference
that no longer resolves draws as None, silently, pre-1.0 and without migration.
On a picture the model's bubble moves onto the system's regular material and
Bob's keeps its solid accent tint; on a plain colour or a gradient both bubbles
are exactly as they are today. The Mac and the iPad offer the same set from the
one source built twice. Nothing about a background reaches the server: no
request body, header, query, log line or statistics record, held by a
source-reading architecture test beside the one that holds the built-in
answerer's "nothing sent anywhere". Impact: additive. The server, the control
panel and `config.json` are untouched.

## Scope

In: `client/DessauChat/` (a new `Backgrounds.swift` and a new `Contrast.swift`,
plus edits to `DessauChat.swift` and `Bubbles.swift`), `client/tests/` (the
Swift unit target's cases for the store, the resolver and the contrast
arithmetic), the client's XCUITest tier, `internal/archtest/`, and
`client/README.md`.

Out: the server, the control panel, `config.json`, the bridge, the pairing and
key handling; the model picker of itd-2609200829199959; the answer styles of
itd-2609200850330402; the Mac's Desktop Pictures and any other system path.

## Approach

### The background model

One declaration, in a new file `client/DessauChat/Backgrounds.swift`:

```swift
enum ChatBackground: RawRepresentable, Hashable, Codable, Sendable {
    case none
    case colour(String)     // a built-in colour, by name
    case gradient(String)   // a built-in gradient, by name
    case dynamic(String)    // a built-in dynamic background, by name
    case picture(String)    // a file name inside the client's Backgrounds folder
}
```

Its `rawValue` is one flat token — `none`, `colour.blue`, `gradient.dusk`,
`dynamic.aurora`, `picture.<file name>` — and `init?(rawValue:)` parses it. Two
things follow from writing it this way and both are why it is written this way:
a `RawRepresentable` whose raw value is a `String` is what `@AppStorage` can
hold, so the Settings default needs no second encoding; and the record's form is
one line of text that an architecture test can read, so "the record holds a
reference and never image data, and never a path" is checkable rather than
promised. A `picture` payload containing `/` is not a valid token and parses to
`nil`, which resolves to None.

### The built-in set

The set is declared once, in `Backgrounds.swift`, with no `#if os(macOS)`
anywhere in the file: assets compiled into both bundles and gradients drawn in
code, never a path into Desktop Pictures or any other system location
(cond-2609201011301249). Every built-in carries two authored faces, a light and
a dark, rather than one stored value given a face by arithmetic. That is the
deliberate difference from a picked bubble colour, which keeps one stored hex
and gains a face per appearance (the decision of 2026-09-19): there the person
picked the colour and the client may not invent the half they did not choose,
whereas here the client is the author of both halves and can simply draw them.

- **Colours** (a flat fill, two authored faces each): Graphite, Blue, Indigo,
  Pink, Orange, Green.
- **Gradients** (a two-stop linear gradient, top to bottom, two authored stop
  pairs each): Sunrise, Dusk, Meadow, Tide.
- **Dynamic** (a `MeshGradient` whose control points drift on a `TimelineView`,
  two authored palettes each): Aurora, Drift. They are drawn from numbers in
  this file and need no system asset, which is what keeps them on the iPad.

`Backgrounds.swift` becomes the fourth named exception in
`TestChatClientCarriesNoStylingOfItsOwn`, with its reason: a background is
drawing the client does itself by definition, and no system control draws one.
The exception stays one file wide — the transcript applies its background
through `.chatBackground(_:)`, a modifier declared in `Backgrounds.swift`
exactly as `.bubble(_:isUser:)` is declared in `Bubbles.swift`, so
`DessauChat.swift` gains none of the banned modifiers.

### Pictures: the picker, the copy, the file layout

Photo… presents SwiftUI's `PhotosPicker` bound to a single `PhotosPickerItem`,
with `photoLibrary: .shared()`. The picker runs out of process and hands back
only what Bob points at, so the client has no photo-library entitlement, reads
no `PHAsset`, and neither `client/Info.plist` nor `client/Info-iPad.plist`
gains `NSPhotoLibraryUsageDescription`. The same call is the picker on both
systems (cond-2609201011303278).

The item is loaded as `Data` and written into the client's own container beside
the conversation store, which is already
`Application Support/DessauChat/conversations.json`:

```
Application Support/DessauChat/Backgrounds/<uuid>.<ext>
```

`<uuid>` is minted at the copy, and `<ext>` comes from the transferable's own
content type (`jpeg`, `png`, `heic`), defaulting to `jpeg`. The record stores
`picture.<uuid>.<ext>` — the last path component and nothing else, so no path
into the library can be held even by accident. The copy is
`BackgroundStore.store(data:contentType:) throws -> String`, a function over a
directory it is handed, which is what lets the unit target exercise it in a
temporary directory. Deleting a chat deletes the file its record named; no other
sweep is written.

### The chat's menu and its sheet

`CommandMenu("Chat")` gains `Button("Background…")` on Cmd-Shift-B, before the
Delete Chat divider. Every action in that menu carries a shortcut and
`TestChatClientMenuActionsCarryShortcuts` counts them, so the shortcut is not
optional. `ChatActions` gains `chooseBackground: () -> Void`; `RootView` holds
`@State private var backgroundShown = false` beside `pickerShown` and passes it
to `ChatDetail`, which presents the sheet — the same shape the model picker
already has, so the iPad needs no second arrangement.

The sheet is `BackgroundPicker`, in `Backgrounds.swift`: **None** first, on its
own and before anything else (so it is never hunted for), then a
`LazyVGrid` of the built-ins under Colours, Gradients and Dynamic, each swatch
drawn in the face for the appearance the window is in and labelled with its
name, the chosen one ringed and carrying `accessibilityAddTraits(.isSelected)`;
then **Photo…**. Choosing anything sets the conversation's background and
dismisses.

### Storage, and the Settings default

`Conversation` gains `var background: ChatBackground? = nil`. Optional on
purpose and for the reason `Message.model` already gives in this file: Codable's
synthesised decoder falls back for a missing key only on an optional property,
so a `conversations.json` written before this change decodes unchanged. `nil`
reads as None everywhere. No compatibility shim and no migration are written
(cond-2609201011305747).

Settings gains, in the Theme section beside the bubble colours,
`@AppStorage("defaultBackground") var defaultBackground: ChatBackground = .none`
and a row that opens the same `BackgroundPicker`. `AppModel.newChat()` sets the
new conversation's background from it. It is read at the creation of a chat and
at no other moment, which is the whole of why an existing chat cannot be changed
by it (cond-2609201011303817).

### Rendering

The transcript's `ScrollView` carries `.chatBackground(resolved)`; the composer
and the sidebar do not. The background is a layer behind the bubbles, not behind
the window's chrome.

`BubbleModifier` gains one parameter, `onPicture: Bool`, and `MessageRow` is
handed the conversation's resolved background so it can pass it:

- `onPicture == false` — a plain colour, a gradient, a dynamic background or
  None. Both bubbles are drawn exactly as they are today: the user's bubble
  opaque in its accent face, the model's at `0.28` of the model colour. Nothing
  about the bubbles changes on a plain background (cond-2609201011302819).
- `onPicture == true` — the user's bubble is unchanged, still the solid accent
  tint of iss-2609200818340380. The model's bubble is drawn as
  `.background(.regularMaterial, in: RoundedRectangle(cornerRadius: 16, style:
  .continuous))` instead of the tinted fill, and its label stays
  `Color.primary` in the window's own appearance, which is what the material's
  vibrancy is specified against. There is no scrim over the picture.

**The contrast arithmetic, and what it can honestly hold.** A model bubble at
28% opacity composites over whatever is behind it, so a background changes the
fill its label is read on — that, and not the picture case, is where the
arithmetic bites. The pure sRGB arithmetic (`srgb`, `readable`, luminance) moves
out of `Bubbles.swift` into a new `Contrast.swift` that imports nothing, so the
Swift unit target compiles it alone the way `client/tests/card-summary.sh`
already compiles `CardSummary.swift`; `Bubbles.swift` calls it and keeps its
SwiftUI wrappers. The unit target then asserts, for every built-in face, in
Light and in Dark: the composite of the model bubble's 28% fill over that face
keeps at least 4.5:1 against the label colour of that appearance, and the user
bubble's accent face keeps the same against its own label. The material case
cannot be computed — the system's blur is the system's — so it is held on two
other terms instead: a source-reading assertion that the model bubble on a
picture is the regular material and not a colour, and the XCUITest tier drawing
a deliberately light and a deliberately dark picture in both appearances. Saying
which of the three tiers holds which half is the point; a unit test cannot prove
a blur and this spec does not claim it does.

### The resolver, and the fallback

`BackgroundStore.resolve(_ background: ChatBackground?) -> Resolved` maps the
record to what the view draws: `nil` and `.none` to `.none`; a built-in name
that is not in the set to `.none`; and `.picture(file)` to the decoded image, or
to `.none` when the file is absent, unreadable or undecodable. Resolution is
attempted at draw time. Nothing is shown, nothing is logged, and **the record is
not rewritten** — a file restored later comes back on its own, and a chat never
refuses to open over its decoration (cond-2609201011305747).

### The iPad

One source, built twice. `Backgrounds.swift` contains no `#if os(macOS)` at all,
which is the form of the equality claim an architecture test can read, and
`TestChatClientGuardsTheMacOnlyCalls` already keeps `import AppKit`,
`NSPasteboard`, `NSApp` and `controlActiveState` out of it. `PhotosPicker`,
`MeshGradient` and `TimelineView` are on both systems at 27. The Mac's menu item
and the iPad's — the iPad reaches the Chat menu from the hardware keyboard and
the same sheet from the picker's place in the chat — present the identical
`BackgroundPicker` (cond-2609201011300784).

### What never changes

`ServerBackend.reply` builds the same request body and the same headers it
builds today; `BuiltInBackend` is untouched; no log line and no statistics
record learns of a background, on either side (cond-2609201011309491,
cond-2609201011307256). The proof is a source-reading architecture test,
`TestChatClientBackgroundNeverLeavesTheDevice`, written beside
`TestChatClientBuiltInBackendTouchesNoNetwork` and in the same style: neither
`Backends.swift` nor any other request-building, logging or statistics type
mentions `ChatBackground`, `BackgroundStore` or `defaultBackground`, and
`Backgrounds.swift` imports no `Network`, no `Foundation` URL loading and no
`Photos` (only `PhotosUI`).

### The client README

`client/README.md` gains a short section under Use: choosing a background from
the Chat menu, the built-in set, a picture of your own through the system's
picker, the Settings default for new chats, that a chat whose picture is gone
simply draws plain again, and one sentence that a background never leaves the
device — no server, no model and no log is told what a chat looks like. Present
tense, British English.

## How each criterion is held

abcd mints ids for scope conditions, not for criteria, so the criteria are cited
by their ordinal in itd-2609200854205251. Every test named here is watched to
fail before the change and pass after. The XCUITest tier runs on the simulator
and only when a change touches `client/**` (the decision of 2026-09-20); the
Swift unit target and the Go architecture tests run on every change.

| # | Criterion | Held by |
|---|---|---|
| 1 | Chat menu offers Background, None first, the set beneath, Photo… | archtest `TestChatClientOffersABackgroundFromTheChatMenu`: the Chat menu declares the item with its shortcut, and `Backgrounds.swift` declares None before the grid and Photo… after it; `TestChatClientMenuActionsCarryShortcuts` keeps the counts equal |
| 2 | Per conversation, the other chat unchanged, both survive a relaunch | unit target over the store (two conversations encoded, decoded, one background each); XCUITest tier for the drawn result |
| 3 | The Settings default applies to a new chat and to no existing chat | unit target over `newChat()` and the default's `RawRepresentable` round trip |
| 4 | None puts the transcript back on the plain window, no residue | XCUITest tier |
| 5 | A picture is copied into the container; the record holds a reference | unit target over `BackgroundStore.store(data:contentType:)` in a temporary directory; archtest that the record's token is a bare file name (no `/`, no image data) |
| 6 | A removed file draws as None, no error, the record not refused | unit target over `BackgroundStore.resolve` with the file deleted, and a decode of a record naming a missing file |
| 7 | On a picture: accent user bubble, material model bubble, contrast in both appearances | unit target over `Contrast.swift` for the composite on every built-in face in Light and Dark; the existing `TestChatClientBubbleColoursFollowTheAppearance` extended to the material case; XCUITest tier for the drawn result on a light and a dark picture |
| 8 | Nothing sent anywhere | archtest `TestChatClientBackgroundNeverLeavesTheDevice`, beside `TestChatClientBuiltInBackendTouchesNoNetwork` |
| 9 | The iPad's set is the Mac's | archtest that `Backgrounds.swift` declares the set with no `#if os(macOS)` branch; plus a named hand check, "the background set and Photo… on the maintainer's own iPad", recorded with its date in the shipping decision line |
| 10 | `client/README.md` documents it | a docs test in the settings-docs family: the README names Background, the Settings default, the fallback to None, and the sentence that nothing leaves the device |

## Security and privacy

A picture never leaves the device. It is read through a picker that runs out of
process, copied into the client's own container under Application Support, and
referenced by a bare file name; no request body, header, query, log line or
statistics record carries it or its name, and no model is told a conversation
has one. The client asks for no photo-library access: no entitlement, no
`NSPhotoLibraryUsageDescription` in either Info.plist, and no `Photos`
framework import — the picker's scope is the single item Bob points at, for as
long as it takes to copy it. The copy is the client's own file from then on, so
tidying the library changes nothing, and deleting the chat deletes it. There is
no new network surface, no new file read outside the container, and no new
parser over remote input; this change touches none of the trust boundaries the
repository names.

## Verification

- `client/build.sh` — the Mac bundle builds.
- `SIM=1 client/build-ipad.sh` — the iPad build compiles and starts on the
  iOS 27 simulator.
- The Swift unit target — the store, the resolver and the contrast arithmetic.
- The XCUITest tier on the simulator — the drawn result for criteria 2, 4
  and 7; run because this change touches `client/**`.
- `make test`, `gofmt -l .` empty, `go vet ./...` clean — the Go suite carries
  the architecture tests.
- By hand on the maintainer's own iPad, over USB: the Chat menu's Background,
  the full built-in set, Photo… through the system picker, and the transcript
  in Light and in Dark. Recorded with its date in the shipping decision line.

## What is deliberately not built

- No Desktop Pictures, no wallpaper folder, no arbitrary file browsing.
- No full photo-library access, no album browsing, no live-photo or video
  background.
- No global background that decorates the app rather than the chats; the
  Settings value is a default for new chats and nothing more.
- No scrim over a picture, and no material behind the user's bubble.
- No per-background text colour, blur strength or opacity control.
- No migration, no compatibility shim, no rewrite of a record whose picture has
  gone.
- No sharing, export or rendering of a decorated transcript — that is the one
  change that would put the background store and the request path back in
  contact, and it is refused here on purpose.
- Nothing on the server, the control panel, `config.json` or the bridge.

## Falsifiers, from the intent's Mechanism

- A background that has to be resolved inside the request path — an export or a
  share that renders the transcript — which would put the drawing and the
  request builder back in contact.
- A picture whose bright region behind a model bubble still fails the contrast
  the arithmetic asserts, in either appearance.
- A decode of the conversation store that fails outright on an unresolvable
  reference rather than falling through to None.
- A dynamic background that cannot be drawn without a private system asset,
  which would be dropped from the set on both systems rather than on one.

Any of these reopens the intent rather than being patched.
