---
id: spc-2609181104491592
slug: bob-finds-a-conversation-from-a-search-bar-at-the-top-of-the
intent: itd-2609181104498312
origin: researcher-authored
production_mode: hand-written
---
# Bob finds a conversation from a search bar in the sidebar

## Summary

The sidebar List carries .searchable with sidebar placement; the shown
conversations are filtered live by a case-insensitive contains over titles
and message text; an empty query shows all.

## Scope

In scope: Sidebar in client/GropiusChat/GropiusChat.swift; an architecture
test. Out of scope: highlighting matches; searching thoughts.

## Approach

A query state, a shown computed list, the ForEach and onDelete over shown;
the selection is untouched by the filter.

## How the acceptance criteria are met

1. Filtering and clearing — shown. 2. The test reads the searchable
modifier and the filter's two fields.

## Verification

Suite, build, docs lint; checked by hand.
