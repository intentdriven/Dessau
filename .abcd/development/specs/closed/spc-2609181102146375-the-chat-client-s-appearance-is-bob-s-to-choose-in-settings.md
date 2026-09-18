---
id: spc-2609181102146375
slug: the-chat-client-s-appearance-is-bob-s-to-choose-in-settings
intent: itd-2609181102147562
origin: researcher-authored
production_mode: hand-written
---
# The chat client's appearance is Bob's to choose

## Summary

One stored setting, `appearance`, holding `system`, `light` or `dark`, is
applied with `.preferredColorScheme(_:)` at the root of the window scene and
the Settings scene: `nil` for System, `.light` or `.dark` otherwise. Every
colour the client uses is the system's, so the whole window follows.

## Scope

In scope: `client/GropiusChat/GropiusChat.swift` — an `Appearance` enum, an
`@AppStorage("appearance")` read at the App, the modifier on both scenes, a
`Picker` in Settings under the "Text" section (renamed "Appearance", holding
both pickers); an architecture test reading the enum's three cases and the
two modifier sites; the client README; the changelog. The iPad client applies
the same modifier at its window root through the shared file.

Out of scope: a per-window appearance; any colour of the client's own.

## Approach

`enum Appearance: String, CaseIterable { case system, light, dark }` with a
`colorScheme: ColorScheme?` property; the App reads
`@AppStorage("appearance") var appearance = Appearance.system.rawValue` and
applies `.preferredColorScheme(Appearance(rawValue:)?.colorScheme ?? nil)`
to both scenes' content. The picker is a segmented `Picker("Appearance",
selection:)` over the three cases.

## How the acceptance criteria are met

1. Every window switches at once — the modifier at both scene roots reads
   one stored value.
2. System follows the Mac — `nil` preference.
3. Persists — `@AppStorage`.
4. The test — the enum's three cases and the two modifier sites.

## Verification

`make test`, gofmt, vet, docs lint; the Mac client builds and launches; the
three appearances checked by hand and recorded in the shipping decision
line.
