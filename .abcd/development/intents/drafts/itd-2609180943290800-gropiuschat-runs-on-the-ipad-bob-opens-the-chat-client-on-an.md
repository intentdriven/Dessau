---
id: itd-2609180943290800
slug: gropiuschat-runs-on-the-ipad-bob-opens-the-chat-client-on-an
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609151701196720, itd-2609170718438919, itd-2609170718430553]
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# GropiusChat runs on the iPad

## Press Release

> _Seeded from a quoted-text intent capture. Expand into the full press-release narrative before planning._

## Why This Matters

GropiusChat runs on the iPad: Bob opens the chat client on an iPad running iPadOS 27 and it is the same app he knows from the Mac, made for the iPad — the system's own controls, a sidebar of chats, the composer at the bottom, the model picker in the toolbar. It chats with the iPad's own model when the iPad has one (Apple Intelligence on an eligible iPad), and when it does not, or when Bob asks, it offers Alice's Gropius server from the local network, exactly as the Mac client does. Nothing is sent anywhere until Bob picks a server. Alice, running the server, sees the same requests she sees from a Mac.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

- Distribution: an iPad does not run an ad-hoc signed app. Reaching Bob's
  iPad needs a provisioning profile from an Apple Developer account, and
  reaching anyone else's needs TestFlight or the App Store; the Mac client's
  zip-and-installer path does not exist here. Which of these is in scope
  gates the intent.
- The build: whether the "no Xcode project" condition of the Mac trunk can
  hold for an iPad bundle (entitlements, signing, the iOS SDK through
  `swiftc`), or whether this is the first Xcode project in the repository.
- Whether the iPad ships a Foundation Models model at all on the devices
  Bob has: the framework exists on iPadOS 26 and later on an iPad with an M1
  or later or an A17 Pro, behind Apple Intelligence; the client's
  availability check already answers "device not eligible" and falls back to
  the picker.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
