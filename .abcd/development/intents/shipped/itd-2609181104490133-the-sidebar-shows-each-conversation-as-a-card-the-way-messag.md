---
id: itd-2609181104490133
slug: the-sidebar-shows-each-conversation-as-a-card-the-way-messag
spec_id: spc-2609181104494855
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609151701196720]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# The sidebar shows each conversation as a card

## Press Release

The sidebar shows each conversation as a card, the way Messages does: a
larger row with an icon for who answered, the title, the date the
conversation started, and a summary that says how many exchanges and how
many words it holds. The selected conversation is drawn in the system's
highlight colour, as a selected row in any Mac sidebar is.

## Why This Matters

A list of titles tells Bob what a chat was about but not when, how long, or
with which model; the card is the glance that saves opening it.

## Mechanism

We expect the card to be the standard sidebar row given more content
because a List with the sidebar style already draws its selection in the
system's highlight, and the row's content is a Label with an icon and two
lines of text — no styling of the client's own. What would show this
wrong: a row height the sidebar list refuses to give, or a selection colour
that is not the system's.

## Scope Conditions

- The icon is the system symbol for who answered last: the Mac's own model <!-- cond: cond-2609181104499203 -->
  or a server; the date is the conversation's creation date in the
  system's short format; the summary counts exchanges (a person's message
  and its reply) and words.
- The selection colour is the sidebar's own; nothing is drawn behind the <!-- cond: cond-2609181104494069 -->
  row by the client.

## Acceptance Criteria

- Given three conversations, when Bob looks at the sidebar, then each row
  shows an icon, the title, the date and "N exchanges · M words", and is
  visibly taller than a single line.
- Given a conversation selected, when Bob looks, then its row is in the
  system's highlight colour and its text reads on it.
- Given the client's source, when the architecture tests run, then the row
  carries the four parts and the list uses the sidebar style.

## Open Questions

_None recorded yet._

## Audit Notes

<!-- abcd-review: OWED receipt=rcp-576a7829485a -->
Fidelity review OWED (receipt rcp-576a7829485a).

## Grounds

- pursued: we expect the sidebar's standard row with more content and the sidebar list style to read as Messages' cards with no drawing of our own; wrong if the row height or the highlight is not the system's
