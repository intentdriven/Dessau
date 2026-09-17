---
id: itd-2609151701196720
slug: gropiuschat-is-a-native-macos-27-app-bob-opens-the-chat-clie
spec_id: spc-2609161738006056
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
impact: breaking
origin: researcher-authored
production_mode: hand-written
---

# GropiusChat is a native macOS 27 app: Bob opens the chat client on a Mac running macOS 27 and it looks, moves and behaves like an app made for that system — the system's own windows, toolbar, controls and design language, nothing borrowed from a browser or an older macOS — while everything it does today (finding a Gropius server on the network, picking a model, streaming a reply, keeping the API key in the keychain) still works unchanged. Alice, running the server, sees no difference: the client speaks the same OpenAI-compatible API as before, and the server side never knows which version of the client is talking to it.

## Press Release

GropiusChat is a native macOS 27 app. Bob opens the chat client on a Mac
running macOS 27 and cannot tell it from an app Apple shipped with the
system. Its window, sidebar, toolbar and controls are macOS 27's own, with no
styling of the client's left over. It has a real menu bar: Cmd-N starts a
chat, Cmd-, opens Settings, and every action has a shortcut. It remembers its
windows across a relaunch, and a text file dropped on the composer becomes
part of the prompt.

Everything it does today works unchanged: finding a Gropius server on the
network, picking a model, streaming a reply, keeping the API key in the
keychain. Bob on macOS 26 keeps the last client built for it, published
beside the current release and never updated. Alice, running the server, sees
no difference: the client speaks the same API as before.

## Why This Matters

The client is the only part of Gropius a person who is not running the
server ever touches, and it is the part that has had no user-facing change in
the record. A native app that looks like the system it runs on is trusted
where a styled one is tolerated; the system's conventions (menus, shortcuts,
window restoration, drag and drop) are what a Mac user reaches for without
reading anything. Moving the floor to macOS 27 lets the client take what the
system offers instead of carrying its own styling, and is the trunk the
system-text-intelligence, App Intents and text-effects intents build on.

## Mechanism

We expect the client to read as system-native on macOS 27 because SwiftUI
on that system renders its standard controls in the system's own design when
no custom modifier overrides them, so removing the client's styling is enough.
What would show this wrong: a control that still needs a custom style to look
right on 27, or a view that has to be rebuilt in AppKit to match the system.

## Scope Conditions

- Built with the command-line SDK, not Xcode 27: `client/build.sh` keeps <!-- cond: cond-2609161738009947 -->
  building with `swiftc` against the macOS 27 SDK that ships with the
  Command Line Tools, and Xcode 27 is not a requirement to build or ship the
  client. Wrong if a 27 API the client adopts needs Xcode's own toolchain
  (an asset catalog, App Intents metadata extraction) to work.
- The server side is untouched: the server keeps its macOS 26 floor and its <!-- cond: cond-2609161738009497 -->
  OpenAI-compatible API, and nothing this intent ships changes anything under
  `internal/` or `cmd/`. Wrong if the client needs a new endpoint or a new
  field from the server to feel native.

## Acceptance Criteria

- Given the client running on macOS 27, when Bob opens it, then every window,
  toolbar, sidebar and control is a standard SwiftUI control in the system's
  own design, and a test over the client's source finds no custom style
  modifier left in it.
- Given the client is open, when Bob presses Cmd-N, then a new chat opens;
  when he presses Cmd-,, then Settings opens; and every action in the menu
  bar carries a keyboard shortcut, declared through the standard Commands
  scene rather than hand-built views.
- Given Bob has chats open in more than one window, when he quits and
  relaunches the client, then the same windows come back where they were;
  and when he drops a text file on the composer, then its content is part of
  the next prompt he sends.
- Given a Gropius server on the network, when Bob uses the client, then
  finding the server, picking a model, streaming a reply and keeping the API
  key in the keychain all work exactly as they did before; and given a Mac on
  macOS 26, when Bob runs `install.sh`, then it installs the kept 26-floor
  client release instead of failing on `releases/latest`.

## Open Questions

- The one-release rule gains a written exception for the kept macOS 26 client
  release; it is decided (DECISIONS.md, 2026-09-15) but not yet ratified as
  an ADR, because `abcd decide` needs abcd v0.8.0 and this machine runs
  v0.7.1. Ratify once the maintainer has updated abcd.
- `xcrun --show-sdk-version` fails on this Mac because the Command Line
  Tools' SDK symlink is broken, while the macOS 27 SDK directory itself is
  present; `client/build.sh` must point `swiftc` at the SDK explicitly
  (`SDKROOT` or `-sdk`) or the link must be repaired before the first build.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: we expect the chat client to read as an app made for macOS 27 once its own styling is removed and its menus, shortcuts and window restoration come from SwiftUI's standard scenes, because the system draws standard controls in its own design when nothing overrides them; what would show it wrong is a control that still needs a custom style on 27, a convention that needs AppKit, or a 27 API that needs Xcode 27 rather than the command-line SDK
