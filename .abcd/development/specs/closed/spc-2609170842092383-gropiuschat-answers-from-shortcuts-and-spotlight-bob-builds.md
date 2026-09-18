---
id: spc-2609170842092383
slug: gropiuschat-answers-from-shortcuts-and-spotlight-bob-builds
intent: itd-2609151836194280
origin: researcher-authored
production_mode: hand-written
---
# GropiusChat answers from Shortcuts and Spotlight

## Summary

Three App Intents in a new file `client/GropiusChat/Intents.swift` —
`AskIntent` (a prompt, returns the reply as text, keeps the exchange as a new
chat), `NewChatIntent` (opens the client on a new chat) and `OpenChatIntent`
(a `ChatEntity` chosen by title through an `EntityStringQuery`) — plus an
`AppShortcutsProvider` with one phrase per intent. The build script writes
`Metadata.appintents` into the bundle with the toolchain's
`appintentsmetadataprocessor`, fed by the compiler's const-values extraction,
and fails the build if the metadata is missing. `AskIntent` conforms to
`LongRunningIntent` so the system's default limit does not apply.

## Scope

In scope: `Intents.swift`; the two build-script steps (`-emit-const-values`
with the protocol list in `client/appintents-protocols.json`, and the
processor); `AppModel.ask(prompt:)` which creates a chat, sends, and returns
the finished reply; a `TestChatClientBundleCarriesAppIntentsMetadata` in
`internal/archtest` that reads `client/build.sh` for the processor step and
its loud failure; the client README's "Shortcuts and Spotlight" section.

Out of scope: an `IndexedEntity` (nothing is donated to the system index);
interactive snippets; an intents extension (the intents run in the app's own
process, which the system launches in the background when it is not
running).

## Approach

The intents reach the running `AppModel` through a single shared instance
(`AppModel.shared`, which the root view also uses as its `@State` holder's
source), so the conversations file has one writer. `AskIntent.perform` calls
`await model.ask(prompt:)`: it creates a conversation titled from the prompt,
sends it to the current answerer, awaits the reply task, and returns
`.result(value: replyText)`. `NewChatIntent` and `OpenChatIntent` set
`openAppWhenRun = true`, create or select the chat, and return. `ChatEntity`'s
query returns titles from `model.conversations` on demand. Phrases: "Ask
\(.applicationName)", "New chat in \(.applicationName)", "Open a chat in
\(.applicationName)".

The build: `swiftc -c -wmo -emit-const-values -Xfrontend
-const-gather-protocols-file …` then `xcrun appintentsmetadataprocessor
--output Contents/Resources …` before `codesign`; the script fails with a
message if `Metadata.appintents/extract.actionsdata` is absent (verified
outside Xcode on the build Mac, 2026-09-17: the processor wrote the metadata
for a probe intent).

## How the acceptance criteria are met

1. The three actions listed — the metadata in the bundle; checked by hand in
   Shortcuts after the first launch and recorded.
2. "Ask" returns the reply and keeps the chat — `AppModel.ask`.
3. Spotlight offers "New Chat" — the shortcut phrase and `NewChatIntent`;
   checked by hand.
4. A long reply completes — `LongRunningIntent`; checked by hand with a cold
   Gropius model.
5. The build fails loudly without metadata — the script's check and the
   architecture test that reads it.

## Verification

Suite green; `client/build.sh` builds and the bundle carries
`Metadata.appintents`; the manual checks recorded in the decision line.
