---
id: itd-2609151836194280
slug: gropiuschat-answers-from-shortcuts-and-spotlight-bob-builds
spec_id: spc-2609170842092383
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609151701196720]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# GropiusChat answers from Shortcuts and Spotlight

## Press Release

GropiusChat answers from Shortcuts and Spotlight. Bob builds a shortcut
that sends a prompt to the model the client is set to — the Mac's own by
default, one of Alice's if he has chosen it — and gets the reply back as
text; the exchange is kept as a new chat in the client's sidebar. From
Spotlight he starts a new chat or opens an existing one by name. The client
declares its actions as App Intents, so the system lists them by itself and
Bob never configures anything to make them appear.

## Why This Matters

Shortcuts and Spotlight are how a Mac user makes an app part of their day
without opening it. A chat that can be asked from a shortcut is a tool; one
that can only be typed into is a window.

## Mechanism

We expect Shortcuts and Spotlight to list the actions because App Intents
metadata is extracted at build time by the toolchain's metadata processor,
which the build script runs itself the way other non-Xcode build systems do,
and the system reads it from the bundle. What would show this wrong: the
processor refusing a single-file build, or the system not indexing an
ad-hoc-signed, zip-installed bundle until it has been opened once — in which
case the acceptance bar moves to "after the first launch".

## Scope Conditions

- Built with the installed Xcode 27 toolchain: the metadata processor ships <!-- cond: cond-2609170842095773 -->
  only in Xcode, not in the Command Line Tools. The trunk's condition on the
  toolchain was narrowed to exactly this on 2026-09-17 before this intent is
  planned, and the trunk's spec names the processor step as this intent's.
- The actions run inside the app's own process — the system launches it in <!-- cond: cond-2609170842096291 -->
  the background when it is not running — through the same model object the
  windows use, so there is no second writer of the conversations file.
- Chats are looked up on demand through an entity query when Bob picks one; <!-- cond: cond-2609170842094378 -->
  nothing is donated to the system's index, so no title, prompt or reply
  leaves the app's own store.
- The 27 client only; the prompt action answers with whichever model the <!-- cond: cond-2609170842090438 -->
  client is set to, so a Gropius model needs the app's stored server.

## Acceptance Criteria

- Given the client is installed and has been opened once, when Bob opens
  Shortcuts, then "Ask Gropius" (a prompt, returning text), "New Chat" and
  "Open Chat" (a chat, chosen by name) are listed under the app.
- Given a shortcut runs "Ask Gropius", when the model replies, then the
  shortcut receives the reply text, and the exchange appears in the client's
  sidebar as a chat titled from the prompt.
- Given Spotlight, when Bob types "New Chat", then the client's action is
  offered and running it opens the client on a new chat.
- Given the prompt action is declared long-running, when a model takes over
  a minute to answer (checked by hand with a cold Gropius model and recorded
  in the shipping decision line), then the shortcut completes with the reply
  rather than timing out.
- Given the build, when the metadata processor does not run or writes
  nothing, then the build fails and says so; a test holds the bundle to
  carrying the metadata.

## Open Questions

_None recorded yet._

## Audit Notes

<!-- abcd-review: OWED receipt=rcp-4049a6fa90f8 -->
Fidelity review OWED (receipt rcp-4049a6fa90f8).

## Grounds

- pursued: we expect Shortcuts and Spotlight to list the client's actions from metadata the build script writes with the toolchain's own processor, because that step ran and produced the metadata on this Mac outside Xcode; wrong if the system does not index an ad-hoc-signed zip-installed bundle, or if a long-running prompt intent still times out
