---
id: spc-2610031153301961
slug: a-program-that-finished-with-a-model-can-tell-dessau-to
intent: itd-2610031024247803
origin: researcher-authored
production_mode: hand-written
---
# A program can ask Dessau to unload a model it has finished with

## Summary

Delivers itd-2610031024247803 under adr-2610031153127219: `POST
/v1/dessau/unload` for trusted callers, refusing protected models, behind a
setting that is on by default; and documentation of the control panel's own
unload route.

## Approach

- **Route.** Registered inside `routes()` so `withAuth` and `pairedOnly` apply;
  admitted only when `entitled(r)` holds; requires `Content-Type:
  application/json`; body `{"model": "<id>"}` decoded with the control plane's
  existing bound and resolved like a chat request's model. No `DELETE` on `/v1`;
  an archtest asserts nothing reachable from `/v1` calls the registry's or the
  pool's delete paths.
- **Refusals (409, at once).** Not loaded; pinned; serving or loading a
  request; inside its eviction-grace window. The pool gains one call that
  checks these under its lock and unloads (beside the operator's `Unload`,
  which keeps its own semantics); it never interrupts idle work. A caller not
  entitled gets the gateway's generic refusal and learns nothing.
- **Setting.** `api_unload_off` in `config.json` (absent means on), a switch in
  the panel's Settings, the three-surfaces tests, never refused over an
  untouched save.
- **Records.** A new statistics stop reason (`released`) with the kind of
  caller (this Mac, API key, paired client by name), never the key or the
  source tag; the operational log says the same.
- **Docs.** A reference page for both unload routes: who can use each, the
  refusals, and that the panel route follows the panel-access setting.

## Steps

1. The pool's checked unload and the route
   - packages: internal/runtime, internal/gateway
   - tests: an idle unpinned model is unloaded for a caller on this Mac; another device on a keyless network is refused and learns nothing; a key holder and a paired client may unload; a cross-site browser request is refused; pinned, busy, loading and in-grace models are refused at once; DELETE on /v1 changes nothing
2. The setting, statistics and docs
   - packages: internal/config, internal/gateway, internal/ui, internal/stats, docs
   - tests: the switch off refuses the route and leaves the panel's Unload working; the three surfaces agree and an absent key means on; the new stop reason and caller kind are recorded without the key; the docs page names both routes

## Footprint

- packages: internal/runtime, internal/gateway, internal/config, internal/ui, internal/stats
- tests: the eleven acceptance criteria, held as listed in the steps

## Out of scope

Loading or pre-loading through the API; deleting models; tracking which
client used a model.
