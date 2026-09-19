---
schema_version: 1
id: "iss-2609181213191220"
slug: "the-sidebar-search-matches-the-whole-trimmed-query-as-one-li"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "fidelity audits 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/GropiusChat.swift"
resolution: "The sidebar's filter now splits the query on whitespace and keeps a conversation only when every word is found, case- and accent-insensitively, in its title or any message — so words that appear apart, or in another order, find the chat. The match moved to client/GropiusChat/SidebarSearch.swift, a Foundation-only file that client/tests/sidebar-search.sh compiles and runs, with internal/archtest running that script and pinning the sidebar to it."
impact: fix
---

The sidebar search matches the whole trimmed query as one literal substring, so a multi-word query matches only contiguous text and never words found separately, against the press release's promise that the list filters to conversations containing the words.

## Grounds

- pursued: we expect a multi-word search to narrow the list to the chats that contain all the words because that is what itd-2609181104498312 promises and what a search bar elsewhere on the Mac does; wrong if people type multi-word queries meaning an exact phrase, in which case a phrase that is present would stop being distinguishable from its words scattered about.
