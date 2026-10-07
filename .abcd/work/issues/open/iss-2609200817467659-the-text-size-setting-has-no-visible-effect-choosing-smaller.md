---
schema_version: 1
id: "iss-2609200817467659"
slug: "the-text-size-setting-has-no-visible-effect-choosing-smaller"
severity: "minor"
category: "bug"
source: "manual-test"
found_during: "maintainer manual test of the chat client, 2026-09-20"
origin: researcher-authored
production_mode: hand-written
found_at: "client/DessauChat/DessauChat.swift"
remedy: "Scale the chat's fonts ourselves on macOS: map the five Text size names to a scale factor and apply it to every text style the chat window and Settings use (a scaled-font helper every view goes through), with an architecture test that no chat text uses an unscaled style; then check by hand that Smaller and Huge visibly change both windows."
---

The Text size setting has no visible effect: choosing Smaller, Larger, Extra Large or Huge in Settings leaves the chat window's text at the same size. The setting stores one of five names and applies them through SwiftUI's dynamicTypeSize modifier, which on macOS does not scale the system text styles the way it does on iOS, so the modifier is honoured in the view tree and changes nothing on screen. Seen in the macOS client at Smaller with no change in either window.
