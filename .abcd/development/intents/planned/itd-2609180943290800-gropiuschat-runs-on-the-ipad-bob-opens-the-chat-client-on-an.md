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

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: we expect the shared SwiftUI files to become an iPad app with little more than a build target and a settings sheet, and the iPad on the same network to be the everyday way to use Alice's server from the sofa; wrong if the iPad build needs a rewrite or an Xcode project, or if the local-network browse and the picker do not feel as immediate as on the Mac
