---
id: spc-2609181100463204
slug: the-chat-client-s-text-size-is-bob-s-to-choose-in-settings-h
intent: itd-2609181100459593
origin: researcher-authored
production_mode: hand-written
---
# The chat client's text size is Bob's to choose

## Summary

One stored setting, `textSize`, holding one of five names, is turned into a
`DynamicTypeSize` and applied with `.dynamicTypeSize(_:)` at the root of the
window scene and the Settings scene. Nothing else changes: every standard
control and every `Text` reads the environment, so the whole window scales
with the system's own type metrics.

## Scope

In scope: `client/GropiusChat/GropiusChat.swift` — a `TextSize` enum (five
cases mapping to `.small`, `.large`, `.xLarge`, `.xxLarge`, `.xxxLarge`), an
`@AppStorage("textSize")` read at the App, the modifier on both scenes, and a
`Picker` in Settings under a "Text" section; an architecture test that reads
the enum's five cases and the two modifier sites; the client README's Settings
sentence; the changelog. The iPad client applies the same modifier at its
window root through the shared file.

Out of scope: a point size; per-surface sizes; the effects renderer's
metrics (it reads the same layout).

## Approach

`enum TextSize: String, CaseIterable { case smaller, standard, larger,
extraLarge, huge }` with a `dynamicType` computed property; the App reads
`@AppStorage("textSize") var textSize = TextSize.standard.rawValue` and
applies `.dynamicTypeSize(TextSize(rawValue:)?.dynamicType ?? .large)` to the
window content and the Settings content. The picker is a segmented or menu
`Picker("Text size", selection:)` over `TextSize.allCases` with the labels
above.

## How the acceptance criteria are met

1. The whole window grows — the modifier at both scene roots.
2. Persists — `@AppStorage`.
3. Default is unchanged — `.large` is the system default.
4. The test — reads the enum and the two modifier sites.

## Verification

`make test`, gofmt, vet, docs lint; the Mac client builds and launches; the
sizes checked by hand and recorded in the shipping decision line.
