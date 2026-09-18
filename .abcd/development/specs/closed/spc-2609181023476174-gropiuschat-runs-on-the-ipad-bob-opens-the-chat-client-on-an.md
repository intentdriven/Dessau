---
id: spc-2609181023476174
slug: gropiuschat-runs-on-the-ipad-bob-opens-the-chat-client-on-an
intent: itd-2609180943290800
origin: researcher-authored
production_mode: hand-written
---
# GropiusChat runs on the iPad

## Summary

The Mac client's Swift files become an iPad app: the same sources, guarded
in four places for what iPadOS lacks, a shared discovery file on the
Network framework in place of the deprecated NetService resolver, a second
build script beside the Mac's that compiles against the iOS SDK into a flat
bundle with a launch-screen key, an asset-catalogue icon and the App Intents
metadata, and signs it with the maintainer's free personal team for their
own iPad. A simulator run on the build Mac exercises the on-device path,
which the Mac's model lends the simulator; the maintainer's 2020 iPad Pro
exercises the server path.

## Scope

In scope: `client/GropiusChat/*.swift` (platform guards; the pasteboard,
the icon, the Settings presentation and the key-window follow), a new
`client/GropiusChat/Discovery.swift` (browse with `NWBrowser`, resolve with
`NWConnection` to name and port, shared by both clients; `ServerBrowser`
and `ServiceResolver` move there and lose NetService), `client/Info-iPad.plist`,
`client/build-ipad.sh`, the two client architecture tests that read
`GropiusChat.swift` by name (they follow the shared file) and the plist
check (every client plist), a new `TestChatClientIPadBuildIsSignedForADevice`,
`client/README.md` (an iPad section), the changelog.

Out of scope: an iPhone layout (`UIDeviceFamily` is 2); any release asset
or installer change; TestFlight, the App Store or any signing beyond the
personal team (the maintainer's decision, 2026-09-18); a Mac Catalyst or
Xcode project.

## Approach

### One source, two systems

`#if os(macOS)` guards the `AppKit` import, the `Settings` scene (on iPad,
Settings is a sheet from a toolbar button, the same `SettingsView` in a
`NavigationStack` with a Done button), `controlActiveState` (on iPad the
one window follows an intent's selection unconditionally), the pasteboard
(`UIPasteboard.general.string`) and the empty state's icon (the bundle's
icon image through `UIImage(named:)`). `.commands` builds the iPad's menu
bar for a keyboard as it does the Mac's. Nothing else changes: the
backends, the picker, the markdown, the effects and the intents compile as
they are (typechecked for iPadOS 27 on the build Mac, 2026-09-18).

### Discovery on the Network framework

`Discovery.swift` holds `gropiusServiceType`, `DiscoveredServer`,
`ServerBrowser` (as today, on `NWBrowser`) and a `ServiceResolver` that
opens an `NWConnection` to the browsed endpoint, reads the resolved
endpoint's host and port from the established path, validates the host the
way the NetService resolver did, and closes; the host name a Gropius server
publishes keeps resolving after its address changes because the connection
is made by name. The Mac client uses the same file, which resolves
iss-2609180959409672.

### The iPad build

`client/build-ipad.sh`: `xcrun swiftc` against `iphoneos` (or
`iphonesimulator` with `SIM=1`) at `arm64-apple-ios27.0`, the same
`-swift-version 6 -default-isolation MainActor`, whole-module with
`-emit-const-values`; a flat bundle (`GropiusChat.app/GropiusChat`,
`Info.plist` at the root); the App Intents processor with
`--platform-family iOS`, the iOS deployment target and triple; an asset
catalogue written by the script from the existing icon PNG and compiled by
`actool` into `Assets.car` with the icon keys merged into the plist; a
`UILaunchScreen` key; `UIDeviceFamily` 2, `LSRequiresIPhoneOS`,
`MinimumOSVersion` 27.0, `NSBonjourServices`, `NSLocalNetworkUsageDescription`.
For a device: `IPAD_SIGNING_IDENTITY` (an "Apple Development" certificate
the personal team minted through one Xcode sign-in) and `IPAD_PROFILE` (the
matching `.mobileprovision`, copied in as `embedded.mobileprovision`), an
entitlements file with the application identifier and team, `codesign`
with both; the script refuses loudly when either is missing rather than
producing a bundle no iPad will run, and installs with `xcrun devicectl
device install app` when `IPAD_DEVICE` names one. For the simulator: no
signing; `xcrun simctl boot`, `install` and `launch --console`.

### Tests

The discovery and load-state tests read `Discovery.swift` for the service
type and the browser, and `GropiusChat.swift` for the rest; the Bonjour
plist check runs over both client plists. The new test reads
`build-ipad.sh` for the loud refusal without an identity and a profile, the
launch-screen key in `Info-iPad.plist`, the iOS platform family for the
processor, and the absence of `--sign -`.

## How the acceptance criteria are met

1. The on-device path in the simulator — the simulator target and the
   shared backends; the "On this iPad" name from the platform guard.
2. An ineligible iPad offers a server — the same availability reading,
   checked on the maintainer's iPad and recorded.
3. Local-network permission and the picker — the shared discovery file and
   the iPad plist's keys; checked on the iPad.
4. Cmd-N and the menu bar with a keyboard — `.commands`; checked on the
   iPad.
5. The build produces, signs and installs a flat bundle — `build-ipad.sh`
   and its test; a simulator launch on the build Mac.
6. Guards and no NetService — the architecture tests.

## Verification

`make test`, gofmt, vet, docs lint; `client/build.sh` still builds the Mac
client; `SIM=1 client/build-ipad.sh` builds, installs and launches on an
iPad simulator on the build Mac; the device build and the checks on the
maintainer's iPad by hand, recorded in the shipping decision line.
