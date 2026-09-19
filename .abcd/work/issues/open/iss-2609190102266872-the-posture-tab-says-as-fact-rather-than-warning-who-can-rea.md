---
schema_version: 1
id: "iss-2609190102266872"
slug: "the-posture-tab-says-as-fact-rather-than-warning-who-can-rea"
severity: "minor"
category: "observation"
source: "impl-review"
found_during: "the adversarial security review adr-2609181004167097 obliges before a bridge lands"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/ui/static/index.html"
---

The Posture tab says, as fact rather than warning, who can reach the server and by which addresses, what is announced, and what is recorded — and it is described in README.md as 'one page that says what is on'. The Discord bridge is now a thing that can be on and that the Posture tab says nothing about. It is outbound only, so none of the page's existing claims about who can reach this Mac become untrue; what is missing is the fact that, while the bridge is on, conversations are leaving the Mac to a third party. That belongs on the page that exists to say what is on. Left out of the bridge's own change deliberately: the spec (spc-2609181018499470) scopes the bridge's surfaces to the Settings pane, the docs page and the changelog, and adding a Posture line is a judgement about what that page is for.

## Deferral 2026-09-19

Deliberately left out of the bridge's own change. spc-2609181018499470 scopes
the bridge's surfaces to the Settings pane, `docs/discord-bridge.md` and the
changelog, and what the Posture tab is for — inbound reach, or everything that
is on — is a decision about that page rather than about the bridge. It needs
the maintainer to say which, and the wording that follows from it.
