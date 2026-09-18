---
id: spc-2609161738006056
slug: gropiuschat-is-a-native-macos-27-app-bob-opens-the-chat-clie
intent: itd-2609151701196720
origin: researcher-authored
production_mode: hand-written
---
# GropiusChat is a native macOS 27 app

## Summary

The chat client's floor moves from macOS 26 to macOS 27, and the client
stops carrying styling of its own: the window, sidebar, toolbar and controls
are the standard SwiftUI controls, drawn by the system in its own design. The
app gains what a Mac user reaches for without reading anything — a real menu
bar with a shortcut on every action through the standard `Commands` scene, a
`Settings` scene behind Cmd-, in place of today's sheet, windows that come
back where they were after a relaunch, and a text file dropped on the composer
becoming part of the prompt. Everything the client does today — finding a
server over Bonjour, choosing a model under the chat rule, streaming a reply,
keeping the API key in the keychain — is unchanged, and so is the server: its
floor stays 26 and its API is not touched.

A Mac on macOS 26 keeps the last 26-floor client. That release stays published
beside the current one as the one written exception to the one-release rule
(DECISIONS.md 2026-09-15, ADR to follow), and `install.sh` in client mode on a
26 Mac fetches from that fixed release rather than from `releases/latest`.

Impact: **breaking** — the client's floor moves, and a 26 Mac stops receiving
client updates.

## Scope

In scope: `client/GropiusChat/GropiusChat.swift`, `client/build.sh`,
`client/Info.plist`, `client/README.md`; the client half of `install.sh`; the
floor guard in `internal/archtest` (`minimum_macos_test.go`), which today holds
every surface to one floor and must hold two; a new architecture test over the
client's source for styling that is not the system's; `README.md`'s
requirement sentence and `CHANGELOG.md`; the release notes that name the kept
26-floor release.

Out of scope, each for its reason:

- **Anything under `internal/` other than `internal/archtest`, and `cmd/`.**
  Scope condition cond-2609161738009497: the server side is untouched. The
  archtest package is the repository's guard over every surface, not server
  code, and the floor guard already reads the client's files.
- **An Xcode project.** Scope condition cond-2609161738009947, as narrowed on
  2026-09-17: the installed Xcode 27's toolchain builds it through `xcrun`,
  with no project, asset catalog or other Xcode-only build step. The App
  Intents metadata step is the Shortcuts intent's (itd-2609151836194280),
  which adds it to the build script as one more toolchain invocation.
- **System text intelligence, App Intents and Shortcuts, text effects.** Each
  is its own intent building on this one (itd-2609151836193724,
  itd-2609151836194280, itd-2609151836194134).
- **Intel Macs.** macOS 27 runs on none, so the 27 client is arm64 only; the
  kept 26-floor client remains the universal one. The release workflow's
  universal check for the client goes with the slice.
- **A hand-built composer.** `ComposerTextView`, the `NSTextView` wrapper that
  existed for Return-to-send, goes: a standard `TextField(axis: .vertical)`
  with `.onSubmit` sends on Return, breaks the line on Option-Return, and
  carries Writing Tools inline — the system's own composer, which is the
  claim. `ComposerNSTextView` and its height measurement go with it.

## Approach

### The floor

`client/build.sh` compiles one slice, `-target arm64-apple-macos27.0`,
through `xcrun swiftc` against the SDK `xcrun --sdk macosx --show-sdk-path`
names — the installed Xcode 27's — because macOS 27 runs on no Intel Mac and
because the 27 SDK's SwiftUI needs the macro plugin only Xcode carries. The
x86_64 build, the `lipo` step and the release workflow's "client missing
x86_64" check go; the workflow keeps its arm64 check. The build passes
`-swift-version 6 -default-isolation MainActor`, which is what Xcode 27
gives a new app and what the 27 `@State` macro expects. `client/Info.plist`
declares `LSMinimumSystemVersion` 27.0. `build/Info.plist` stays at 26.0.

### Two floors, one guard

`TestEverySurfaceDeclaresTheSameMacOSFloor` becomes two: the server's floor,
read from `build/Info.plist`, holds the server bundle, `install.sh`'s server
refusal and the READMEs' server sentence; the client's floor, read from
`client/Info.plist`, holds `client/build.sh`'s two deployment targets,
`install.sh`'s client refusal and the READMEs' client sentence. The test's
comment already says the number is a plain `"26.0"`-style string that other
tests read as the source of truth; each floor keeps that property in its own
plist. `install.sh` gains `MIN_MACOS_MAJOR_CLIENT` beside `MIN_MACOS_MAJOR`,
chosen by mode, and the guard reads both.

### The kept 26-floor release

`install.sh` in client mode on a Mac whose major version is below the client
floor but at or above the server floor does not refuse: it fetches
`GropiusChat.app.zip` and `SHA256SUMS.txt` from the kept release,
`releases/download/<KEPT_CLIENT_TAG>/…`, verifies the checksum the same way,
and says which release it installed and why. `KEPT_CLIENT_TAG` is one
assignment near the floor constants, named in `README.md`'s requirement
paragraph, and the release notes of the first 27-floor release name it too.
Below the server floor the refusal is unchanged. An architecture test holds
the tag assignment to a tag that exists in the checkout and to the value the
README names, so the pointer cannot drift once the release is kept.

### The model layer

`AppModel` and `ServerBrowser` stay `ObservableObject`s. The `@Observable`
migration was considered and set aside: the architecture tests that hold the
client's promises to the server (the stored rule's defaults, the picker
filter, the Settings bindings) pin `@AppStorage` properties on the model by
their spelling, and `@AppStorage` is a view wrapper an `@Observable` class
cannot carry. `ObservableObject` is not deprecated on 27; the model gains a
`shared` instance the App Intents reach.

### The system's look

Every custom style comes out of the source: the `glassButton` extension and
its `.glass`/`.glassProminent` styles (macOS 26's explicit Liquid Glass call,
which on 27 the standard button draws by itself), the message bubbles'
`.background(bubble)` and the composer's `.background(surface)` with their
`strokeBorder` overlays, the status dot's `.shadow`, and the settings form's
`.roundedBorder` text fields (soft-deprecated in 27). What replaces them is
the standard control with no modifier: `Button`, `TextField`, `List` in the sidebar, `Form` in Settings,
messages laid out with the system's spacing and `.secondary` foreground for
the metadata line. Layout modifiers (`frame`, `padding`, `font`,
`foregroundStyle` with a semantic style) are not styling and stay.

A new architecture test, `TestChatClientCarriesNoStylingOfItsOwn` in
`internal/archtest`, reads the client's source and fails on any occurrence of
`.buttonStyle(.glass`, `.buttonStyle(.glassProminent`, `.background(`,
`.overlay(`, `.shadow(`, `.textFieldStyle(`, `cornerRadius`, `clipShape(` and
`Color(nsColor:`, the way `chat_client_*_test.go` already reads that file for
other promises. It is watched failing against today's source before the
styling is removed. The allowed list is the test's own comment; a future
control that genuinely needs one of these names the exception in the test, in
a sentence saying why, which is the falsifier of the mechanism claim made
visible.

### Menus, shortcuts and Settings

`GropiusChatApp` declares a `Settings { SettingsView(model:) }` scene, which
gives Cmd-, and the app menu's Settings item for free, and `.commands { … }`
on the `WindowGroup` with a `CommandGroup(replacing: .newItem)` for **New
Chat** (Cmd-N), a `CommandMenu("Chat")` for **Send** (Cmd-Return), **Delete
Chat** (Cmd-Backspace), **Reconnect** and **Choose Model…**, each carrying a
`.keyboardShortcut`. The `AppModel` becomes an environment object shared by
the scenes, and the `showSettings` sheet, the settings button beside the
composer hint and the sheet's own close button go. Every menu action is a
method that already exists on `AppModel`; no new behaviour is introduced
through a menu.

### Window restoration and file drop

`WindowGroup` restores its windows on relaunch by itself once each window's
state is in `@SceneStorage`: the selected conversation id moves from
`AppModel.selectedID` to `@SceneStorage("selectedConversation")` in the root
view, and `AppModel` keeps the conversations, which are already persisted. A
second window (Cmd-Shift-N through the standard `WindowGroup` new-window item)
shows another conversation of the same model.

The composer gains `.dropDestination(for: URL.self)` accepting `.plainText`
and `.utf8PlainText` UTIs: the file's content is appended to the draft, with
one blank line between the draft and the file, and a file that is not text or
not readable is declined with the drop's own refusal cursor and no alert.

### Docs and the record

`README.md`'s requirement paragraph says the server needs macOS 26 and the
client macOS 27, names the kept release for a 26 Mac and the one command that
picks the right one; `client/README.md` says the SDK the build script needs
and how to name it when `xcrun` cannot. `docs/` has no page for the client's
look and gains none — the client is the client, and what it looks like is not
a server document. The CHANGELOG entry under `[Unreleased]` says `impact:
breaking` and names the kept release.

## How the acceptance criteria are met

1. **No custom styling** — the removal above, held by
   `TestChatClientCarriesNoStylingOfItsOwn`.
2. **Real menu bar and shortcuts** — the `Commands` and `Settings` scenes;
   held by a source-reading test that every `Button` inside `.commands`
   carries `.keyboardShortcut`, and by the build.
3. **Windows restore, files drop in** — `@SceneStorage` and
   `.dropDestination`; restoration is the system's, verified by hand on the
   maintainer's Mac and recorded in the shipping decision line; the drop's
   text handling is a unit of `AppModel` (append with one blank line, decline
   a non-text file) and is tested there.
4. **Nothing regresses, 26 keeps its build** — the existing
   `chat_client_*_test.go` promises stay green; `install.sh`'s client branch
   is held by the two-floor guard and the kept-tag test, and by the installer
   gate in the release workflow, which runs client mode on the runner.

## Verification

`make test`, `go vet ./...`, `gofmt -l .` green; `client/build.sh` builds a
universal bundle on the maintainer's Mac with the Command Line Tools' SDK and
no Xcode 27; `abcd docs lint` clean; the new tests watched failing first.
Manual, recorded in the shipping decision line: open the client on macOS 27
beside a built-in Apple app and compare; quit with two windows open and
relaunch; drop a text file on the composer; run `install.sh client` on a 26
Mac after the first 27-floor release and confirm it installs the kept release.

## Open questions carried from the intent

- The ADR ratifying the kept-release exception waits for abcd v0.8.0 on this
  machine; the decision line of 2026-09-15 is the record until then.
- The Command Line Tools' SDK symlink is broken on the maintainer's Mac; the
  build script naming the SDK explicitly is the fix this spec chooses, so the
  link need not be repaired for the build to work.
