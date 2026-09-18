---
id: itd-2609170718430553
slug: gropiuschat-offers-alice-s-models-when-it-finds-her-server-b
spec_id: spc-2609170842083562
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609151701196720, itd-2609170718438919]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# GropiusChat offers Alice's models when Bob looks for them

## Press Release

GropiusChat offers Alice's models when Bob looks for them. Bob is chatting
with the Mac's own model and opens the model picker: beneath "On this Mac" a
section reads "Servers on your network", and as the client finds Alice's
Gropius server it appears there. He picks it, the client asks it for its
models, and her chat models are listed under her server's name. One click and
the conversation carries on where it was, now answered by her model.

The client looks for servers only while the picker is open, so the Mac's
Local Network permission is asked for the first time Bob looks, not the first
time he opens the app. Finding a server is an offer, never a takeover: the
client never switches by itself, and "On this Mac" is always one click away.
Alice sees a models-list request when Bob picks her server, a chat request
once he has chosen a model, and nothing before.

## Why This Matters

The built-in model is the floor; Alice's server is the upgrade — bigger
models, longer windows, the same chat. If the client had to be configured to
find it, most people would never get there. Discovery already exists in the
client, hidden in a Settings pane; putting the offer where Bob is already
choosing a model is what makes the server discoverable in fact.

## Mechanism

We expect the offer to read as part of the app rather than an interruption
because the model picker is a toolbar button that presents a standard popover
holding a `List` with sections, and a popover's presented state is what
starts and stops the Bonjour browse the client already has. What would show
this wrong: a browse that needs longer than the popover stays open to find a
server (then the section needs a "looking…" row and a longer browse), or a
Local Network prompt that macOS raises at launch regardless.

## Scope Conditions

- Discovery runs only while the model picker is open: no browsing at launch, <!-- cond: cond-2609170842082521 -->
  no Local Network prompt before Bob looks. The maintainer's decision at the
  interview, 2026-09-17.
- The server side is untouched: the client uses the models list and the chat <!-- cond: cond-2609170842083752 -->
  endpoint exactly as today; a server's "offer" is what those endpoints
  already publish.
- The 27 client only, building on the built-in default <!-- cond: cond-2609170842089337 -->
  (itd-2609170718438919), whose bar that nothing is sent before Bob chooses
  this intent refines rather than restates.
- The choice of who answers is global to the client, as the model choice is <!-- cond: cond-2609170842085039 -->
  today; switching does not fork the conversation. A transcript that mixes
  both goes to a server whole and to the built-in model as the most recent
  turns that fit.
- The stored API key is sent only to the server it was saved for: a newly <!-- cond: cond-2609170842089482 -->
  found server is asked for its models without a key, a refusal says so and
  points at Settings, and `client/README.md`'s sentence on which machine
  receives the key is rewritten to say this.
- `client/README.md`'s Use section is rewritten: no address is tried at <!-- cond: cond-2609170842082080 -->
  launch, and the Local Network permission is asked the first time the model
  picker is opened. Checking that by hand needs a Mac whose local-network
  grant has been reset; the result is recorded in the shipping decision line.

## Acceptance Criteria

- Given Bob is chatting with the built-in model, when he opens the model
  picker, then the client starts browsing, the picker shows "On this Mac" and
  a "Servers on your network" section that fills as servers are found, and
  the browse stops when the picker closes.
- Given a server appears in the section, when Bob picks it, then the client
  fetches its models list (with the stored API key if there is one), lists
  its chat models under the server's name, and sends no chat request until he
  picks a model.
- Given Bob picks one of Alice's models, when he sends his next message, then
  the whole conversation so far goes to that model and the reply streams as
  it does today, and Alice's log shows that request and nothing earlier.
- Given Bob is on Alice's model, when he picks "On this Mac" again, then his
  next message goes to the built-in model with the most recent turns that fit
  its window, and nothing more is sent to Alice's server.
- Given no server is found, when the picker has been open for a few seconds,
  then the section says "No server found" and the built-in model stays
  selected.
- Given Alice's server leaves the network, when Bob sends, then the failure is
  shown on that message with the picker one click away, and the client does
  not switch by itself.
- Given a server whose models all fall outside the chat rule, when Bob picks
  it, then the section says under its name that none of its models can hold
  a conversation, and the built-in model stays selected.

## Open Questions

- The architecture test that holds "the client opens on the server's own
  default address" keeps its check on the typed address's default; its stated
  reason ("the composer is disabled until a server answers") is no longer
  true and the test's comment says what is.

## Audit Notes

<!-- abcd-review: OWED receipt=rcp-f71b7e2f71f1 -->
Fidelity review OWED (receipt rcp-f71b7e2f71f1).

## Grounds

- pursued: we expect a server offered inside the picker Bob is already using to be found and chosen far more often than one hidden in Settings, because the offer appears at the moment of choosing and costs one click; wrong if the popover's browse is too short to find a server, or if the Local Network prompt at first pick puts people off
