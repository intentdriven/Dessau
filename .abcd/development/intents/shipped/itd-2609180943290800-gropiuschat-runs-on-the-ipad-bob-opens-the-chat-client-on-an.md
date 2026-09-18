---
id: itd-2609180943290800
slug: gropiuschat-runs-on-the-ipad-bob-opens-the-chat-client-on-an
spec_id: spc-2609181023476174
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609151701196720, itd-2609170718438919, itd-2609170718430553]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# GropiusChat runs on the iPad

## Press Release

GropiusChat runs on the iPad. Bob opens the chat client on an iPad running
iPadOS 27 and it is the app he knows from the Mac, made for the iPad: the
system's own controls, a sidebar of chats, the composer at the bottom, the
model picker in the toolbar, a menu bar when a keyboard is attached. When
the iPad has a model of its own — an iPad with an M1 or later, or the iPad
mini with an A17 Pro, with Apple Intelligence on — it answers on the device;
when it does not, as on the maintainer's own 2020 iPad Pro, the picker
offers Alice's Gropius server exactly as itd-2609170718430553 offers it on
the Mac. Alice, running the server, sees the same requests she sees from a
Mac.

The app is installed on Bob's own iPad from the Mac it was built on, signed
with his own Apple ID's personal team; it is not published for others.

## Why This Matters

An iPad on the same network is the couch: the everyday way to use Alice's
models without sitting at the Mac. The client is a handful of SwiftUI files
that compile for both systems, so the iPad app is what the Mac client
already is, on the device Bob actually holds.

## Mechanism

We expect the iPad app to be the Mac client's files plus a second build
target because everything the client does — SwiftUI views, the Foundation
Models framework, the Network framework, the Keychain, URLSession — exists
on iPadOS 27 in the same shape, and the whole set typechecks for iPadOS with
only the AppKit import and four macOS-only calls (the Settings scene, the
active-window state, the pasteboard, the app icon) to guard. What would show
this wrong: a free personal team's certificate, profile and
`embedded.mobileprovision` that cannot be minted and applied without an
Xcode target, so that the bundle the script builds cannot be installed
with `devicectl` or dies at the seven-day expiry; or a control that needs
a UIKit rebuild to read as native.

## Scope Conditions

- iPadOS 27 or later, iPad only (device family 2): iPhones are not a target, <!-- cond: cond-2609181023475013 -->
  and a resizable iPhone layout is not what "made for the iPad" means.
- The on-device model exists only on an iPad with an M1 or later or the <!-- cond: cond-2609181023475336 -->
  A17 Pro mini with Apple Intelligence on; on any other iPad the client
  reports "device not eligible" and offers a server. The maintainer's own
  iPad (a 2020 iPad Pro) is one of those, so the server path is the one
  their hardware exercises; the on-device path is checked in the simulator
  on the build Mac, which runs the Mac's model.
- Installed on the maintainer's own iPad only, signed with a free personal <!-- cond: cond-2609181023479725 -->
  team's certificate and a profile that expires after seven days, over USB
  from the build Mac; nothing is published for other people, and the
  release carries the build script and an unsigned bundle at most. The
  maintainer's decision at the interview, 2026-09-18.
- Built by a script of its own beside the Mac client's, against the iOS <!-- cond: cond-2609181023470128 -->
  SDK, with no Xcode project: a flat bundle rather than `Contents/`, the
  App Intents processor told `iOS` and the iOS deployment target, the icon
  compiled by `actool` from the Icon Composer source, a launch-screen key,
  and development signing with the personal team's certificate and profile
  in place of the Mac's ad-hoc signature; one Xcode sign-in is what mints
  the certificate and the profile.
- The architecture tests that hold the client's promises follow the shared <!-- cond: cond-2609181023475373 -->
  discovery file rather than one file's name, and the Bonjour service-type
  check covers every client plist, so an iPad plist cannot ship without
  `NSBonjourServices` and browse to a silent empty list.
- The installer and the release's asset set are untouched: the iPad bundle <!-- cond: cond-2609181023472129 -->
  is not a release asset and `install.sh` knows nothing of it.
- Discovery is re-cut on the Network framework for both clients, refining <!-- cond: cond-2609181023470257 -->
  itd-2609170718430553: the resolver the Mac client borrowed from
  NetService is deprecated at 27.2 on every platform and does not exist for
  the iPad (iss-2609180959409672); one discovery file is shared.
- The server side is untouched. <!-- cond: cond-2609181023474856 -->

## Acceptance Criteria

- Given the on-device path, when the client runs in the iPad simulator on
  the build Mac (which lends it the Mac's model), then the composer is
  enabled before any connection is made, the picker reads "On this iPad",
  and the first message is answered without a request; eligibility on a
  real iPad is not verified by this project, which owns none, and is
  recorded as such.
- Given an iPad without the on-device model, when Bob opens the client,
  then the empty chat says the iPad cannot answer and offers a server, and
  nothing is sent until he picks one; checked on the maintainer's iPad.
- Given Alice's server on the same network, when Bob opens the picker, then
  the iPad asks for local-network permission once, lists her server, and
  picking a model carries the conversation there; a reply streams as on the
  Mac.
- Given a keyboard attached, when Bob presses Cmd-N, then a new chat opens,
  and the menu bar carries the same actions and shortcuts as the Mac's.
- Given the build script, when it runs with the iOS SDK, then it produces
  a flat iPad bundle with the launch-screen key, the App Intents metadata
  and the icon, signs it with the personal team's certificate and profile,
  and installs it on the connected iPad; and a simulator run launches it.
- Given the client's shared source, when the architecture tests run, then
  the macOS-only calls are guarded and the discovery file imports no
  NetService.

## Open Questions

- Whether Shortcuts on the iPad lists the three actions from a personally
  signed bundle; checked after the first install and recorded.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-416c47bd8e6c -->
Fidelity review — receipt rcp-416c47bd8e6c (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:a1325946fb5e2493c7f3f8927eabaf82d00019f4123780b172591868350c501b
Input attestations: diff:c428103..c2bd0ed -- client internal/archtest CHANGELOG.md client/README.md (PR 62)@sha256:9cf3a6de197f9b100f3b17935d2dcc235f88250d3fc27e42eaa9bc756b1ed076;

Acceptance rollup: MET 1 · MET_WITH_CONCERNS 0 · NOT_MET 0 · INCONCLUSIVE 5

Per-criterion verdicts:
- ac-1 — INCONCLUSIVE: the code supports every clause (displayName is "On this iPad", canSend is true for .builtIn with no connection), and the real-iPad caveat is recorded as the criterion asks, but the record's only simulator result is build/install/launch/still-running: no run recorded the composer enabled, the picker reading "On this iPad" or a first message answered, and no simulator is available here
  evidence: client/GropiusChat/Backends.swift:78 — "static let displayName = "On this \(deviceNoun)""
  evidence: .abcd/work/DECISIONS.md:285 — "SIM=1 client/build-ipad.sh builds, installs and launches on an iPad Pro 13-inch simulator and is still running after eight seconds"
  evidence: .abcd/work/DECISIONS.md:285 — "NOT verified, and owed to the maintainer's own iPad: ... the on-device model on an iPad eligible for Apple Intelligence"
- ac-2 — INCONCLUSIVE: the empty state does render cannotSend ("This iPad cannot run the system's own model.") with a "Choose Who Answers…" button and send is guarded by canSend, but the criterion's own qualifier -- "checked on the maintainer's iPad" -- has no recorded result: the record lists the device build and install as still owed
  evidence: client/GropiusChat/Backends.swift:121 — "reason: "This \(deviceNoun) cannot run the system's own model.","
  evidence: client/GropiusChat/GropiusChat.swift:1154 — "Button("Choose Who Answers…", action: choose)"
  evidence: client/GropiusChat/GropiusChat.swift:512 — "guard !prompt.isEmpty, canSend, let idx = index(of: convoID) else { return }"
  evidence: .abcd/work/DECISIONS.md:285 — "NOT verified, and owed to the maintainer's own iPad: the device build's signing and install over USB"
- ac-3 — INCONCLUSIVE: the iPad bundle declares NSLocalNetworkUsageDescription and the _gropius._tcp browse type, and the shared Discovery.swift browses and resolves for both clients, but the record puts "the local-network prompt at the first picker open" in the owed column, so the once-only prompt, the listing and the carried conversation are unverified on an iPad
  evidence: client/Info-iPad.plist:56 — "< key>NSLocalNetworkUsageDescription< /key>"
  evidence: client/GropiusChat/Discovery.swift:109 — "let browser = NWBrowser("
  evidence: .abcd/work/DECISIONS.md:285 — "NOT verified ... the local-network prompt at the first picker open"
- ac-4 — INCONCLUSIVE: one shared .commands block gives both systems the same New Chat / Cmd-N, Send, Stop, Choose Model, Reconnect and Delete Chat (plus an iPad-only Cmd-, for Settings), so the source promises parity, but the record names "Cmd-N and the rest of the menu bar with a keyboard attached" as owed to the maintainer's iPad and no device is available here
  evidence: client/GropiusChat/GropiusChat.swift:739 — ".keyboardShortcut("n", modifiers: .command)"
  evidence: .abcd/work/DECISIONS.md:285 — "NOT verified ... Cmd-N and the rest of the menu bar with a keyboard attached"
- ac-5 — INCONCLUSIVE: the script demonstrably produces the flat bundle (Info.plist copied to the bundle root), the App Intents metadata, the actool icon and the launch-screen key, refuses without an identity and a profile, and its simulator leg is recorded as built-installed-launched; the device leg -- signing with the personal team's certificate and installing over USB via devicectl -- is explicitly recorded as not verified
  evidence: client/build-ipad.sh:115 — "cp Info-iPad.plist "$BUNDLE/Info.plist""
  evidence: client/build-ipad.sh:188 — "codesign --force --sign "$IPAD_SIGNING_IDENTITY" \"
  evidence: client/build-ipad.sh:199 — "xcrun devicectl device install app --device "$IPAD_DEVICE" "$BUNDLE""
  evidence: .abcd/work/DECISIONS.md:285 — "NOT verified, and owed to the maintainer's own iPad: the device build's signing and install over USB"
- ac-6 — MET: `go test ./internal/archtest/` passes here: TestChatClientGuardsTheMacOnlyCalls walks the #if stack and fails any AppKit import, NSPasteboard, NSApp or controlActiveState outside an os(macOS) branch, and TestChatClientSharesDiscoveryWithoutNetService requires client/GropiusChat/Discovery.swift and rejects any NetService mention in any client source
  evidence: internal/archtest/chat_client_ipad_test.go:137 — "var macOSOnlyCalls = []string{"import AppKit", "NSPasteboard", "NSApp", "controlActiveState"}"
  evidence: internal/archtest/chat_client_ipad_test.go:214 — "if strings.Contains(src, "NetService") {"
  evidence: client/GropiusChat/Discovery.swift:16 — "import Network"

Gap audit:
- honoured:
  - one source, two systems: the Mac client's Swift files build a second way against the iOS SDK with the macOS-only calls behind #if os(macOS), and an architecture test holds it
    evidence: internal/archtest/chat_client_ipad_test.go:142 — "func TestChatClientGuardsTheMacOnlyCalls(t *testing.T) {"
    evidence: client/GropiusChat/GropiusChat.swift:18 — "#if os(macOS)"
  - a build script of its own with no Xcode project: flat bundle, App Intents processed for iOS, actool icon, launch-screen key, development signing rather than an ad-hoc signature
    evidence: client/build-ipad.sh:100 — "xcrun appintentsmetadataprocessor \"
    evidence: client/build-ipad.sh:155 — "xcrun actool --compile "$BUNDLE" \"
    evidence: client/Info-iPad.plist:38 — "< key>UILaunchScreen< /key>"
  - iPad only, iPadOS 27 floor: no iPhone layout is offered
    evidence: client/Info-iPad.plist:33 — "< integer>2< /integer>"
    evidence: client/Info-iPad.plist:26 — "< string>27.0< /string>"
  - the picker offers Alice's server on the iPad exactly as on the Mac: one shared Discovery.swift, the service type declared once, and every client plist held to it
    evidence: internal/archtest/chat_client_discovery_test.go:89 — "plists, err := filepath.Glob(filepath.Join(root, "client", "Info*.plist"))"
    evidence: client/Info-iPad.plist:63 — "< key>NSBonjourServices< /key>"
  - nothing is published for other people: no release asset for the iPad bundle and install.sh untouched
    evidence: client/README.md:113 — "published: there is no release asset, and `install.sh` knows nothing of it."
    evidence: client/.gitignore:4 — "dist-ipad/"
  - eligibility on a real iPad is not verified by this project and is recorded as such
    evidence: .abcd/work/DECISIONS.md:285 — "NOT verified, and owed to the maintainer's own iPad ... the simulator cannot answer for any of them"
- diverged:
  - the intent's scope condition says discovery is re-cut on the Network framework for both clients; delivered, only the browse is the Network framework's -- the resolve is DNS-SD's C API (import dnssd, DNSServiceResolve), changed under the security review so the API key's origin binding stays a host name rather than an address
    evidence: client/GropiusChat/Discovery.swift:216 — "let started = DNSServiceResolve("
    evidence: .abcd/work/DECISIONS.md:279 — "The resolve goes to DNS-SD, and the binding is a name again"
  - the intent's press release says the app is installed on Bob's own iPad from the Mac it was built on; delivered, the install path exists in the script but is conditional on IPAD_DEVICE and has never been run -- without it the script only prints the devicectl command
    evidence: client/build-ipad.sh:201 — "echo "Install it with: xcrun devicectl device install app --device < udid> $BUNDLE""
    evidence: .abcd/work/DECISIONS.md:285 — "NOT verified, and owed to the maintainer's own iPad: the device build's signing and install over USB"
  - client/README.md tells the reader the simulator borrows the Mac's own language model so the on-device answer can be tried there, but no such answer is recorded anywhere -- the record's simulator result is launch-and-stayed-running only, so the README states as fact a check nothing in the delivery performed
    evidence: client/README.md:133 — "The simulator borrows the Mac's own language model,"
    evidence: .abcd/work/DECISIONS.md:285 — "SIM=1 client/build-ipad.sh builds, installs and launches on an iPad Pro 13-inch simulator and is still running after eight seconds"
- missing:
  - five of the six acceptance criteria rest on behaviour on a real iPad or a simulator run that answers, and the delivery carries no executable check for any of them: the Swift client has no test target, so the only automated hold on the iPad promises is the Go architecture suite, which checks source and plists, never behaviour
    evidence: internal/archtest/chat_client_ipad_test.go:54 — "func TestChatClientIPadBuildIsSignedForADevice(t *testing.T) {"
    evidence: .abcd/work/DECISIONS.md:285 — "the simulator cannot answer for any of them"
  - the intent's open question -- whether Shortcuts on the iPad lists the three actions from a personally signed bundle, checked after the first install and recorded -- is not answered; the build step only asserts the metadata file was written
    evidence: client/build-ipad.sh:113 — "{ echo "error: the App Intents metadata was not written; Shortcuts would list nothing" >&2; exit 1; }"
    evidence: .abcd/work/DECISIONS.md:285 — "Shortcuts listing the client's actions after a first launch"

Scope-condition dispositions:
- cond-2609181023475013 — survived: the iPad bundle declares device family 2 alone and a 27.0 floor, and the archtest suite passes here with both the iPad-only and one-floor subtests green
  evidence: client/Info-iPad.plist:33 — "< integer>2< /integer>"
  evidence: internal/archtest/chat_client_ipad_test.go:112 — "func TestChatClientIPadDeclaresOneFloor(t *testing.T) {"
- cond-2609181023475336 — untested: the eligibility mapping and the "pick a server" fallback are written, but nothing in the delivery ran on the maintainer's iPad or obtained an on-device answer in the simulator, so the assumption about which hardware answers was neither exercised nor contradicted
- cond-2609181023479725 — narrowed: the personal-team signing regime and the no-publication half hold in the script and the docs, but no install on the maintainer's iPad has happened, so the seven-day profile and the USB install are assumptions the delivery never met
  narrowing: holds over the build script's contract only -- it refuses without a development identity and a profile, and never ad-hoc signs -- not over an actual signed install on the maintainer's iPad, which the record lists as still owed
  evidence: client/build-ipad.sh:34 — "if [ -z "${IPAD_SIGNING_IDENTITY:-}" ] || [ -z "${IPAD_PROFILE:-}" ]; then"
  evidence: .abcd/work/DECISIONS.md:285 — "client/build-ipad.sh with no signing identity refuses before it compiles anything"
- cond-2609181023470128 — survived: build-ipad.sh is a script of its own beside build.sh with no Xcode project: it compiles against the iOS SDK, writes a flat bundle, tells the App Intents processor iOS and the deployment target, compiles the icon with actool and signs with the profile's own entitlements
  evidence: client/build-ipad.sh:106 — "--platform-family iOS \"
  evidence: client/build-ipad.sh:155 — "xcrun actool --compile "$BUNDLE" \"
  evidence: client/build-ipad.sh:179 — "cp "$IPAD_PROFILE" "$BUNDLE/embedded.mobileprovision""
- cond-2609181023475373 — survived: the tests follow the shared discovery file and its single gropiusServiceType declaration rather than a file name, and the Bonjour check globs every client Info*.plist, so an iPad plist without NSBonjourServices fails the suite -- run here and green
  evidence: internal/archtest/chat_client_ipad_test.go:205 — "func TestChatClientSharesDiscoveryWithoutNetService(t *testing.T) {"
  evidence: internal/archtest/chat_client_discovery_test.go:89 — "plists, err := filepath.Glob(filepath.Join(root, "client", "Info*.plist"))"
- cond-2609181023472129 — survived: the delivered range touches neither install.sh nor any release workflow, the iPad output directory is gitignored, and the README states the bundle is not a release asset
  evidence: client/.gitignore:4 — "dist-ipad/"
  evidence: client/README.md:113 — "published: there is no release asset, and `install.sh` knows nothing of it."
- cond-2609181023470257 — narrowed: one Discovery.swift is shared by both clients and NetService is gone, held by a test, but the framework the condition names carries only half the work
  narrowing: holds for the browse, which is NWBrowser on the Network framework; the resolve is not -- it is DNS-SD's C API (import dnssd, DNSServiceResolve), adopted after the security review because the Network framework's established path reports an address rather than the published name
  evidence: client/GropiusChat/Discovery.swift:17 — "import dnssd"
  evidence: client/GropiusChat/Discovery.swift:216 — "let started = DNSServiceResolve("
  evidence: internal/archtest/chat_client_ipad_test.go:214 — "if strings.Contains(src, "NetService") {"
- cond-2609181023474856 — survived: the audited delivery (client, internal/archtest, CHANGELOG.md) contains no server file; the internal/discovery, internal/gateway and internal/lifecycle commits inside the wider commit range are separately titled fixes for other issues, not this intent's work
  evidence: .abcd/work/DECISIONS.md:285 — "One source, two systems: the Mac client's own Swift files build a second way against the iOS SDK"
  evidence: CHANGELOG.md:50 — "**GropiusChat runs on the iPad.** The client's Swift files build a second"
## Grounds

- pursued: we expect the shared SwiftUI files to become an iPad app with little more than a build target and a settings sheet, and the iPad on the same network to be the everyday way to use Alice's server from the sofa; wrong if the iPad build needs a rewrite or an Xcode project, or if the local-network browse and the picker do not feel as immediate as on the Mac
