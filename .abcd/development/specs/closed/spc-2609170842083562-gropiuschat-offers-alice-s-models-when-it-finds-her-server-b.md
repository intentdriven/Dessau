---
id: spc-2609170842083562
slug: gropiuschat-offers-alice-s-models-when-it-finds-her-server-b
intent: itd-2609170718430553
origin: researcher-authored
production_mode: hand-written
---
# GropiusChat offers Alice's models when Bob looks for them

## Summary

The model picker becomes a toolbar button that shows the current answerer's
name and presents a standard popover holding a `List` with two sections: "On
this Mac", and "Servers on your network", filled by `ServerBrowser` while the
popover is presented and cleared when it is dismissed. Picking a server
resolves it (the existing `ServiceResolver`), fetches its models list, and
lists its chat models under the server's name; picking a model sets the typed
choice to `.server(model:)`. Nothing is browsed at launch, nothing switches by
itself, and the API key is sent only to the server it was saved for.

## Scope

In scope: `ModelPickerView` (a new file `client/GropiusChat/Picker.swift`),
the popover's presented state driving `ServerBrowser.start/stop`, the
key-scoping change in `AppModel.request`, the Settings pane keeping its
typed-address field and the browse list moving out of it, `client/README.md`'s
Use section, the architecture test on the default address gaining a new
comment.

Out of scope: any server change; auto-connect; remembering more than one
server (the last picked server's address is the stored one, as today).

## Approach

### The popover and the browse

`ChatDetail`'s toolbar holds `Button(currentName) { pickerShown = true }`
with `.popover(isPresented: $pickerShown) { ModelPickerView(model:) }`. The
picker view calls `browser.start()` in `.onAppear` and `browser.stop()` in
`.onDisappear`, which is exactly the lifetime of the popover; the browse
therefore runs only while the picker is open, and macOS raises its Local
Network prompt the first time it starts, i.e. the first time Bob opens the
picker. The section shows "Looking…" with a small progress indicator while
the browse runs and nothing is found, and "No server found" after five
seconds without a result; a failed browse shows its reason as it does in
Settings today.

### Picking a server, then a model

A server row resolves through `ServiceResolver` and, on an address, calls
`connect()` against it: `serverURL`/`serverPath` are set the way `use(_:resolvedAddress:)`
sets them today, the models list is fetched, and the row expands to list
`chatModels` under the server's name; an empty `chatModels` shows the
existing "none of this server's models can hold a conversation" sentence
under the name. A model row sets `answerer = .server` and `selectedModel`,
and the popover stays open until Bob dismisses it. "On this Mac" sets
`answerer = .builtIn`. No chat request is sent by any of this; the first
completion goes when Bob sends.

### The API key

The key is stored with the host it was entered for: `@AppStorage("apiKeyHost")`
is set to `serverURL`'s host when the key is saved in Settings, and
`request(_:)` attaches the bearer token only when the request's host equals
it. A 401 from a newly picked server reads "This server needs an API key; add
it in Settings", and Settings shows which server the key is for.

### Carrying the conversation

`ServerBackend` sends the whole transcript; `BuiltInBackend` trims to the
window it read (the built-in spec). A failed request on a server that has
gone shows the failure on that message; the picker button is one click away
and the choice does not change by itself.

### Tests and docs

`TestChatClientOpensOnTheServersOwnDefaultAddress` keeps its check on the
typed address's default and its comment says the address is the one a picked
or typed server starts from, not a connection made at launch. A new
architecture test holds the picker's browse to the popover's lifetime by
reading `Picker.swift` for `browser.start()` inside `.onAppear` and
`browser.stop()` inside `.onDisappear`, and holds the key scoping by reading
`AppModel.request` for the host comparison. `client/README.md`'s Use section
is rewritten. The Local Network check runs by hand on a Mac whose grant has
been reset and is recorded in the shipping decision line.

## How the acceptance criteria are met

1. Browse starts and stops with the popover — `.onAppear`/`.onDisappear`.
2. Picking a server fetches models without a chat request — `connect()` only.
3. The whole conversation goes to Alice's model — `ServerBackend`.
4. Back to "On this Mac" trims to the window — `BuiltInBackend`.
5. "No server found" after a few seconds — the five-second timer.
6. A server that left shows the failure on the message — the error row.
7. A server with no chat models says so — the existing sentence.

## Verification

Suite green; `client/build.sh` builds; the manual browse and key checks on the
maintainer's Mac, recorded in the decision line.
