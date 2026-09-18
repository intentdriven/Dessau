---
id: spc-2609170842081563
slug: gropiuschat-chats-with-the-mac-s-own-model-out-of-the-box-bo
intent: itd-2609170718438919
origin: researcher-authored
production_mode: hand-written
---
# GropiusChat chats with the Mac's own model out of the box

## Summary

The client gains a backend seam and a second backend. `ChatBackend` is one
protocol — start a reply for a transcript, deliver text as it arrives, stop —
with two implementations: `BuiltInBackend` over the Foundation Models
framework (`LanguageModelSession` on `SystemLanguageModel.default`), and
`ServerBackend`, which is today's OpenAI-compatible streaming code moved
behind the protocol unchanged. The typed choice `Answerer` — `.builtIn` or
`.server(model:)` — is stored in one setting, defaults to the built-in model,
and is what every gate in the client reads: the composer, the send button and
the picker's presence no longer look at the server connection, which becomes
`ServerBackend`'s own state.

## Scope

In scope: a new file `client/GropiusChat/Backends.swift` (the protocol, both
backends, the availability reading, the transcript trim); `AppModel` reads the
choice and delegates streaming; the empty state and the composer hints say
what the built-in model's unavailability reason is; the picker names the
built-in model "On this Mac"; `docs/chat-models.md` says the built-in entry is
exempt from the chat rule; `client/README.md` and `README.md` describe a client
that works without a server; the changelog entry says `impact: breaking`; a
decision line records the three-surfaces exception (written 2026-09-17).

Out of scope: guided generation (`@Generable`, a macro the build does not
carry and free-text chat does not need); Private Cloud Compute (a managed
entitlement an ad-hoc bundle cannot hold); tool calling; per-conversation
answerers (the choice is global, scope condition); the server side.

## Approach

### The seam

```
protocol ChatBackend {
    func reply(to transcript: [Message], deliver: @MainActor (ReplyEvent) -> Void) async throws
}
enum ReplyEvent { case loading, text(String), reasoning(String), done }
```

`ServerBackend.reply` appends: each SSE delta is `.text(delta)` and the model
appends it, exactly as `stream()` does today. `BuiltInBackend.reply` replaces:
each snapshot from `streamResponse` is delivered as `.text(snapshot.content)`
with a flag that says *replace*, so the reply's text is the snapshot, never a
concatenation of snapshots. The event carries that distinction
(`.text(String, replaces: Bool)`), and `AppModel` is the only writer of
message text. Stop is `Task.cancel()` on the reply task for both.

### Availability and the empty state

`BuiltInBackend.availability` reads `SystemLanguageModel.default.availability`
and `supportsLocale(.current)`; the four reasons map to four sentences in the
empty state, each with what would fix it (turn on Apple Intelligence in
System Settings; wait for the model to finish downloading; this Mac is not
eligible; the language is not supported), followed by "or pick a server from
the model picker". When unavailable and no server is chosen, the composer is
disabled with that reason as its hint. Nothing is sent in that state.

### Instructions, prompts and the window

A fixed instruction string (trusted, the client's own words) goes to the
session; the person's text goes only in the prompt. Before each reply the
transcript is trimmed: the client reads `contextSize` and counts the turns
with `tokenCount(for:)` from the most recent backwards, keeping every turn
that fits under the size less a reserve for the answer; the session is
rebuilt from a `Transcript` of those entries (`.prompt`/`.response`) so the
model sees the same history the person does, minus the oldest turns. On
`LanguageModelError.contextSizeExceeded` mid-stream the trim runs once more
with a smaller budget and the request is retried once; a second failure is
shown as an error row. `refusal` shows the model's own explanation;
`guardrailViolation`, `unsupportedLanguageOrLocale`, `rateLimited` and
`concurrentRequests` each show one plain sentence. `GenerationError` is not
used: the 27 SDK deprecates it.

### The picker and the chat rule

The picker's first row is "On this Mac", present whenever the framework is
available, exempt from `ChatRule` because it is not a served model; the
served models keep today's filtering (`chatModels = … .filter { rule.offers }`,
which the architecture test holds). `selectedModel` stays a string for the
server model; the typed choice adds `@AppStorage("answerer")` with the values
`builtin` and `server`, read together.

### Tests and docs

`TestChatClientBuiltInBackendTouchesNoNetwork` in `internal/archtest` reads
`Backends.swift` and fails if the built-in backend's type body mentions
`URLSession`, `URLRequest` or `import Network`. The existing chat-client tests
keep their patterns: `residencyLoaded`, `let state: String?`, `chat ?? true`,
the picker filter and the stored rule defaults stay in `GropiusChat.swift`.
Docs: `docs/chat-models.md` gains one sentence under "In the chat client";
`client/README.md`'s Use section and `README.md`'s client paragraph are
rewritten. Manual checks (no traffic; the refusal path; the trim) are recorded
in the shipping decision line.

## How the acceptance criteria are met

1. Composer enabled and "On this Mac" selected at first launch — the default
   `answerer` and the availability reading; the no-network architecture test.
2. Stop keeps the partial reply — the reply task is cancelled and the text
   written so far stays.
3. Unavailable reasons in the empty state — the four sentences above.
4. Overflow keeps the most recent turns — the trim and the one retry.
5. A refusal shows the model's explanation — the error mapping.
6. Exempt from the chat rule — the first row bypasses `ChatRule`; the sentence
   in `docs/chat-models.md`.

## Verification

`make test`, `gofmt -l .`, `go vet ./...`; `client/build.sh` builds; the
manual checks above on the maintainer's Mac, recorded in the decision line.
