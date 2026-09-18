---
id: spc-2609181104494855
slug: the-sidebar-shows-each-conversation-as-a-card-the-way-messag
intent: itd-2609181104490133
origin: researcher-authored
production_mode: hand-written
---
# The sidebar shows each conversation as a card

## Summary

A ConversationCard view is the sidebar's row: a Label whose icon is the
system symbol for who answered last (the Mac's own model, a server, or
none yet) and whose title holds the conversation's title, the creation
date in the system's short format, and a summary of exchanges and words;
the List takes the sidebar style so the selection is the system's
highlight.

## Scope

In scope: ConversationCard and Sidebar in client/GropiusChat/GropiusChat.swift;
an architecture test. Out of scope: avatars, unread marks, pinning.

## Approach

exchanges = the count of the model's replies; words = the whitespace-split
count over every message; icon from the last reply's recorded answerer
name. Vertical padding on the row makes it a card; nothing is drawn behind
it.

## How the acceptance criteria are met

1. Four parts on every row — the card. 2. Highlight — .listStyle(.sidebar).
3. The test reads the card's parts and the list style.

## Verification

Suite, build, docs lint; the look checked by hand.
