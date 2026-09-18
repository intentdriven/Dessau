---
id: itd-2609170718438919
slug: gropiuschat-chats-with-the-mac-s-own-model-out-of-the-box-bo
spec_id: spc-2609170842081563
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609151701196720]
severity: minor
impact: breaking
origin: researcher-authored
production_mode: hand-written
---

# GropiusChat chats with the Mac's own model out of the box

## Press Release

GropiusChat chats with the Mac's own model out of the box. Bob installs the
chat client on a Mac running macOS 27 and starts a conversation at once: no
server to find, nothing to configure. The model picker reads "On this Mac",
and every reply is written by the language model Apple ships with the system,
on the Mac, with nothing sent anywhere. It is the default: the first chat, and
every chat until Bob chooses otherwise, is answered by the Mac itself. The
choice of who answers is one choice for the whole client, as the model choice
is today, and it is a typed choice — the Mac, or a named model on a named
server — never a model name that a server could happen to share.

When the Mac cannot do it — Apple Intelligence is switched off, the model is
still downloading, the Mac is not eligible, or the language is not supported —
the client says which, in plain words, with what would fix it, and offers a
server instead; it never fails silently. Alice, running a Gropius server, sees
no request from the client until Bob chooses one of her models.

## Why This Matters

Today the client is only a window onto a Gropius server: without one it is a
disabled composer and a "not connected" message. A chat client that works the
moment it is opened is what a Mac user expects of a Mac app, and the system
now ships a model every app may use. Making it the default turns the client
from a companion of the server into an app in its own right, and turns Alice's
server into the upgrade — which is the offer the next intent makes.

## Mechanism

We expect the client to answer with no server because the Foundation Models
framework offers the system's on-device model to any app through a session
API that needs no entitlement, no Xcode-only build step and no App Store
distribution, and it streams; verified on the build Mac with a bare `swiftc`
build of a three-line probe. The framework streams cumulative snapshots, not
deltas, so the built-in backend replaces the reply's text with each snapshot
where the server backend appends a chunk. What would show this wrong: the
framework refusing an ad-hoc-signed app, or a chat of ordinary length
overrunning the model's context window so often that keeping only the turns
that fit makes the conversation incoherent.

## Scope Conditions

- macOS 27 on Apple silicon with Apple Intelligence switched on and a <!-- cond: cond-2609170842087565 -->
  supported language: that is where the on-device model exists. Anywhere
  else the client falls back to a server as today, and says why.
- On the device only: the client uses the system's on-device model and never <!-- cond: cond-2609170842084710 -->
  Apple's Private Cloud Compute model, so "nothing sent anywhere" stays true
  and the telemetry decision (adr-2609061503319212) is not crossed. Wrong if
  the framework ever escalates a prompt off the Mac on its own.
- The 27 client only: the kept macOS 26 client keeps its server-first <!-- cond: cond-2609170842088676 -->
  behaviour and never gains this.
- The three-surfaces exception, recorded: a decision line says that chat <!-- cond: cond-2609170842086229 -->
  against the system's own model is client-only functionality with no server
  equivalent and is not a gap, on the kept-release precedent, to be ratified
  as an ADR once `abcd decide` is available. The client calls a system
  service; it hosts no model, and nothing under `internal/` or `cmd/`
  changes.
- Every gate the client has — the composer's enabled state, the send button, <!-- cond: cond-2609170842088141 -->
  the picker's presence — reads the typed backend choice, never the server
  connection, which becomes a property of the server backend alone.
- `client/README.md`'s "until a server answers, the box is greyed out" and <!-- cond: cond-2609170842089840 -->
  first-run-address sentences and `README.md`'s client paragraph are
  rewritten to the built-in default.

## Acceptance Criteria

- Given a Mac where the on-device model is available, when Bob opens the
  client for the first time, then the composer is enabled at once, the model
  picker shows "On this Mac" selected, and his first message is answered as a
  streamed reply; the built-in backend's source holds no URL request and
  imports no networking, held by an architecture test, and the absence of
  traffic is checked by hand once and recorded in the shipping decision line.
- Given a reply is streaming, when Bob presses Stop, then generation stops and
  the partial reply stays in the transcript.
- Given the on-device model is unavailable, when Bob opens the client, then
  the empty state names the reason (Apple Intelligence off, model not ready,
  Mac not eligible, or language not supported) and what would fix it, the
  picker offers a server instead, and nothing is sent.
- Given a conversation longer than the model's context window, when Bob sends
  the next message, then the client sends the instructions and the most
  recent turns that fit, the reply arrives, and no overflow error is shown.
- Given the model declines a prompt, when it does, then the reply row shows
  the model's own explanation as an error state, never an empty bubble.
- Given the built-in model is selected, when the client applies the chat rule
  to the picker, then the built-in entry is exempt from it, and
  `docs/chat-models.md`'s "In the chat client" section says so.

## Open Questions

- The context window's size on a given Mac is read at runtime
  (`contextSize`); Apple's documentation says 4096 tokens and one WWDC26
  sample prints 8192. The client trims by the figure it reads, not a
  constant.
- The refusal path and the context-window trim are checked by hand on the
  maintainer's Mac and recorded in the shipping decision line; the client has
  no test target of its own.

## Audit Notes

<!-- abcd-review: OWED receipt=rcp-06e492691f86 -->
Fidelity review OWED (receipt rcp-06e492691f86).

## Grounds

- pursued: we expect a client that answers with no server to be the thing that makes the client an app rather than a window, and the system's on-device model to be good enough for the first conversation, because it needs no entitlement and streams from a bare swiftc build (verified on this Mac); wrong if people turn it off for a server at once because its context or its refusals get in the way of ordinary chat
