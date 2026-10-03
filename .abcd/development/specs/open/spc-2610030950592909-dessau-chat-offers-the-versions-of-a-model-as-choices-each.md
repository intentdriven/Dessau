---
id: spc-2610030950592909
slug: dessau-chat-offers-the-versions-of-a-model-as-choices-each
intent: itd-2610030932556747
origin: researcher-authored
production_mode: hand-written
---
# Dessau Chat offers a model's builds as described choices

## Summary

Delivers itd-2610030932556747 in the Swift client under `client/`: the picker
and the Model menu group a server's builds of one model, describe each from the
server's facts, keep "Set as Default" per server, and never switch model by
themselves. Blocked by itd-2610030932551549, whose fields it reads.

## Approach

- **Grouping and wording** live in one pure Swift file (beside
  `CardSummary.swift`) that `client/tests/*.sh` can test without the app. It runs
  on the models that survive Dessau Chat's own chat filter, so a group with one
  chat build left is a plain row. Builds group when `build_of` and `kind` are
  equal. Wording uses only facts: "<bits>-bit · <size>" plus, within a group, a
  relative phrase drawn from size ("quicker to load" for the smallest, "larger
  download" for the largest).
- **The picker and the Model menu** render the same grouped list (the menu
  mirrors the picker, cond-2609201011309498).
- **Defaults per server**: "Set as Default" stores the choice under the server's
  identity, replacing today's single global value; nothing carries over
  (pre-1.0, no migration). A pick inside a chat changes that chat only.
- **A deleted default** is shown as gone, with the other builds of its model
  offered; nothing changes until Carol picks.
- **Old servers**: no `build_of` in the list means today's flat list, unchanged.

## Steps

1. Grouping and wording
   - packages: client/DessauChat (a new pure Swift file), client/tests
   - tests: grouping by build_of and kind after the chat filter; one-build groups as plain rows; wording from facts only; a list without the new fields stays flat
2. Picker, Model menu and defaults
   - packages: client/DessauChat
   - tests: the picker and the Model menu show the same groups; a pick changes this chat only; "Set as Default" per server; a deleted default shown as gone with its siblings
   - hand: the Swift client builds and its UI is checked on a Mac with Xcode (`client/build.sh`); a Linux session can write the code and the shell-tested pure file, not run the app

## Footprint

- packages: client/DessauChat, client/tests
- tests: the six acceptance criteria, held as listed in the steps

## Out of scope

Server-side changes (itd-2610030932551549); remembering the last pick as the
default; switching builds automatically.
