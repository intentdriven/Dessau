---
id: spc-2610031016319710
slug: alice-decides-who-on-her-mac-can-open-dessau-s-control-panel
intent: itd-2610031004535845
origin: researcher-authored
production_mode: hand-written
---
# Alice decides whether other accounts on her Mac can open the control panel

## Summary

Delivers itd-2610031004535845: a setting, "who can open the control panel"
(anyone on this Mac, the default; or this account only), enforced by asking the
kernel which account owns the connecting socket (adr-2610031016411351). The
model API, the network posture and the default behaviour are unchanged.

## Approach

### Attribution
On each accepted connection to the gateway's listeners (via the server's
`ConnContext`), Dessau looks up the peer socket by its full address pair among
the sockets of the processes it may inspect (libproc), caching the result for
the connection's life: "ours" when the owning process runs as the serving
account, otherwise "not ours". Another account's processes cannot be inspected,
so they read as "not ours"; so does any lookup that fails. The lookup lives in
one darwin file behind a small interface; the Linux build reports it
unsupported, which refuses (Dessau runs only on macOS).

### Admission
The control plane keeps its four checks (`fromThisMachine`) and, under "this
account only", adds one: the connection is "ours". The decision is per request
against the live setting. Under "this account only", `/api/events` streams from
connections that are not ours are closed when the setting narrows. `/api/instance`
keeps its own rule (200 only to the server's own account). Refusals name no
account.

### The setting
`panel_access` in `config.json`: `"machine"` (anyone on this Mac, the default)
or `"account"` (this account only). An unrecognised value reads as `"account"`.
The panel's Settings shows the choice; a save carrying the field unchanged is
accepted from anyone the panel admits, and a change to it is accepted only from
a connection that is "ours", so the operator's own browser has proven itself
before the panel can narrow. The three-surfaces tests cover the field.

### Recovery
The menu-bar app (running as the serving account) offers the panel and can
reset the choice to "anyone on this Mac"; a hand edit of `config.json` does the
same at the next start. Recovery never needs the panel.

## Steps

1. The attribution spike
   - packages: internal/gateway (a darwin-only peer lookup and its test)
   - tests: a loopback connection from this account is attributed; one from a process the test cannot inspect is not; the lookup's cost per connection
   - hand: on an Apple Silicon Mac, confirm connections from Safari, Chrome and Firefox, the menu-bar app and the `dessau` command are attributed to the serving account, and record the result in the spec before step 2; a Linux session can write the code but not run this
2. The setting and admission
   - packages: internal/config, internal/gateway, internal/ui
   - tests: refusal of a connection not ours under "account", today's behaviour under "machine", the untouched-field save, the change only from an attributed connection, unknown values fail closed, event streams closed on narrowing, the API unchanged, another machine refused as today, the three surfaces agree
3. Recovery and docs
   - packages: internal/ui (menu bar), docs
   - tests: the menu bar resets the choice; docs/getting-started.md, docs/posture-reference.md, docs/bind-address.md and docs/statistics-explained.md say who can open the panel under each choice

## Footprint

- packages: internal/gateway, internal/config, internal/ui
- tests: the nine acceptance criteria, held as listed in the steps

## Out of scope

The network option (declined); authenticating API clients on this Mac; any
protection from an administrator account.
