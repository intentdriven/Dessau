---
id: spc-2609201459377326
slug: bob-searches-the-web-from-dessau-chat-with-his-own-brave-key
intent: itd-2609201407580721
origin: researcher-authored
production_mode: hand-written
---

# Web search in Dessau Chat: Bob's own Brave key, the client's own loop

## Summary

This spec delivers itd-2609201407580721 in the chat client, plus two
passthrough tests on the server. Bob turns on **Web search** in Settings and
pastes his own Brave Search API key, which lives in his device's Keychain; a
model the server records as tool-capable can then ask for a search through a
tool call the client runs, and the answer arrives with its sources listed
beneath it. A search-first switch on the composer fetches results before the
question is sent, for models that never call tools. The server holds no key,
reaches no new host, and relays the tool traffic as it already relays
`tools`. Impact: additive.

## Scope

In: `client/DessauChat/` — a new `WebSearch.swift`, `Backends.swift`,
`Composer.swift`, `DessauChat.swift` (the `Message` type, `MessageRow`, the
Settings pane) — `internal/archtest/chat_client_websearch_test.go`, two new
tests in `internal/gateway/systemmerge_test.go`, and `client/README.md`. Out:
the gateway's code, which does not change; the control panel and
`config.json`, which gain nothing (cond-2609201459373651); the Discord
bridge; and the operator's key, which is the separate sidecar intent
itd-2609201407587936.

The shape is the first-ranked option of
`.abcd/development/research/notes/2026-09-20-brave-search-key-custody-sota.md`
and the one the 2026-09-10 ideate verdict left standing for a client: a
per-person key with the loop executed on the person's own machine. The
server-held key in the gateway stays dead, on three grounds that record names
— Brave's terms landing on the operator, ninety-day third-party retention of
prompt-derived text, and adr-2609061610102325, which grants the gateway one
reading of prompt content and no more.

## Approach

**The key, in the Keychain.** `client/DessauChat/WebSearch.swift` declares
`enum BraveKey` with `load()`, `store(_:)` and `forget()`, built on the
generic-password pattern `Pairing.swift` already uses: `kSecClassGenericPassword`,
service `sh.intentdriven.dessau.chat`, account `brave-api-key`,
`kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly`, and the
update-then-add sequence that avoids a duplicate item. It is never written to
`UserDefaults`, never to `@AppStorage`, never to a file, and never into a
request to Alice's server (cond-2609201459373651). OWASP MASWE-0004 is the
reason the key is the person's own rather than one shipped in the bundle, and
the person is Brave's customer (cond-2609201459371730).

**The search client.** `WebSearch.swift` also declares `struct BraveSearch`:
one function that takes a query and returns `[SearchResult]` (title, host,
URL, snippet), sending a GET to Brave's web-search endpoint with the key in
the `X-Subscription-Token` header, through a session that is not the paired
one — Alice's server is not in this path at all. Results are bounded: a fixed
maximum count, a byte cap on the response, and a timeout. `struct
FullPage` fetches a result's page and reduces it to text under the same caps,
used only when the per-question full-pages toggle is on
(cond-2609201459374164).

**The tool loop.** `WebSearchTool` declares the function tool `web_search`
with a single `query` string parameter, in the same shape
`FollowUps.tool` uses. `ServerBackend.reply` adds it to the request's
`tools` array when web search is on, a key is set, and the chosen model's
`tool_calling` field from the models list reads `yes`
(cond-2609201459379646, itd-2609201445423499). When the stream returns a
`web_search` call, the client runs the search, appends to the in-flight
message array the assistant turn carrying the tool call and a `tool` role
message holding the results, and re-sends. The loop is bounded: at most two
searches per turn, after which the model answers with what it has. No
`system` role is ever added, so the answer-styles rule
(spc-2609200945518522) holds.

**Search-first.** A switch on the composer, and a per-question full-pages
toggle beside it. When on, the client searches before sending and composes
the newest user turn as the results followed by a blank line and Bob's
question — the same place the answer style's text goes, so there is one rule
for what may be prefixed to a turn and no new role on the wire. It is always
available, whatever the model's recorded capability
(cond-2609201459379646), which is what makes a model that never calls tools
still answer with sources.

**What is kept.** `Message` gains `var sources: [Source]? = nil`, a Codable
struct of title, host and URL. The snippets and any fetched page text live in
the in-flight message array and nowhere else: they are not written to
`conversations.json`, not exported, and not logged
(cond-2609201459378573). `MessageRow` draws the sources under the reply as a
small list, each naming its title and host.

**The on-device model.** `BuiltInBackend` is given search only when a second
Settings switch, off by default, says so (cond-2609201459374478), and then
only search-first: the framework's session is handed the composed turn, not a
tool loop, because the on-device path has no `tool_calling` field to read and
the intent asks for search-first there.

**Settings.** `@AppStorage("webSearchEnabled") = false`,
`@AppStorage("webSearchOnDevice") = false` — both off (cond-2609201459378409)
— a secure field for the key, and the sentences the intent requires: the key
is Bob's, it stays in this device's Keychain, and Brave bills his account for
what he searches. The "Powered by Brave" mark sits on that screen
(cond-2609201459376027). The condition records that Brave's terms speak of
conspicuous attribution where results are shown; this spec implements the
maintainer's decision — the mark in Settings only — and carries the caveat
forward rather than resolving it, so a reviewer who challenges it is
challenging a recorded reading and not an oversight.

**The server, which does not change.** The gateway already relays `tools`
untouched, with a test at `internal/gateway/systemmerge_test.go`. Two tests
join it: one sending a request whose messages include a `tool` role message
with a `tool_call_id`, asserting the upstream receives it with decoded-value
equality; one whose upstream streams `tool_calls` deltas across several SSE
frames, asserting the client receives those frames byte-identical. No
gateway code is written for either; they exist so a later change cannot
quietly break the loop. That no query text is logged follows from the
gateway never reading message content at all, held by
`TestOnlyTheMergeReadsPromptContent`, which this change leaves untouched.

## Acceptance Criteria, and what holds each

The client has no XCUITest target (itd-2609170718438919), so custody,
composition and gating are held by architecture tests over the Swift source,
and the live searches by recorded hand checks against Bob's own key.

- **The key is stored in the Keychain under the client's own service, never
  in a file, and the screen says it is his and that Brave bills him**:
  architecture test — `TestTheBraveKeyLivesOnlyInTheKeychain` in
  `internal/archtest/chat_client_websearch_test.go` asserts the key is read
  and written only through `BraveKey`, that no file under
  `client/DessauChat/` names `brave` beside `@AppStorage` or `UserDefaults`,
  and that the Settings pane carries the two sentences and the Brave mark. A
  recorded hand check pastes a key and inspects Keychain Access.
- **A tool-capable model's search runs in the client and the reply lists each
  source's title and host**: architecture test — the test asserts the tool is
  added only under the three-way guard (switch on, key present, `tool_calling
  == "yes"`), that the loop appends a `tool` role message, and that
  `MessageRow` draws `message.sources`. The live loop: a recorded hand check
  against a served tool-capable model, with the request and reply kept in the
  shipping line.
- **After a relaunch the answer and links survive and no snippet is stored**:
  architecture test — the test asserts `Message` has a `sources` property and
  no snippet property, and that `SearchResult.snippet` is named nowhere in
  the store's encode path. A recorded hand check relaunches and greps the
  stored `conversations.json` for a snippet's words.
- **Search-first sends snippets, or full pages under the per-question
  toggle, so a model that never calls tools still answers with sources**:
  architecture test — the test asserts search-first composes the newest user
  turn and adds no `system` role, and that `FullPage` is reached only from
  the toggle. A recorded hand check against a model recorded as `no`.
- **Search off or no key: the request carries no tools and nothing about
  search appears**: architecture test — the same guard, asserted as the only
  place the tool is added; plus a recorded hand check comparing the request
  body against the server's log with the switch off.
- **Tool messages and streamed tool-call deltas arrive byte-identical and the
  server logs no query**: the two new tests in
  `internal/gateway/systemmerge_test.go`, beside the existing `tools`
  passthrough; and `TestOnlyTheMergeReadsPromptContent`, unchanged and green.
- **The on-device model gets search only under its own switch, and then
  search-first**: architecture test — the test asserts `BuiltInBackend` is
  handed a composed turn only under `webSearchOnDevice` and is never given
  `WebSearchTool`, and that it still names no `URLSession`. A recorded hand
  check in both switch positions.
- **The "Powered by Brave" mark is shown in Settings when search is on**:
  architecture test, as above; and a recorded hand check of the screen.
- **On the iPad the key lives in that device's Keychain and the behaviour is
  the same**: the same architecture tests, which are platform-independent
  source checks, plus a recorded hand check on the iPad simulator to the
  terms of itd-2609180943290800.

Before the pull request: a security review of the key's custody, the search
client's bounds and the loop's injection surface — fetched pages enter the
model's context, which the ideate record names as a hazard — and a
docs-currency pass over `client/README.md`.

## Departures

None at the time of writing.
