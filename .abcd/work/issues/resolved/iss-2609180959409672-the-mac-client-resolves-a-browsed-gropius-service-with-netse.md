---
schema_version: 1
id: "iss-2609180959409672"
slug: "the-mac-client-resolves-a-browsed-gropius-service-with-netse"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "iPad client research, 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/GropiusChat.swift"
resolution: "NetService is gone: discovery lives in one client/GropiusChat/Discovery.swift that both clients compile, browsing with NWBrowser and resolving with DNSServiceResolve — the DNS-SD C API the deprecated class was written over, present on both systems and touching neither AppKit nor UIKit."
impact: fix
---

The Mac client resolves a browsed Gropius service with NetService, which Apple deprecates at 27.2 on every platform in favour of the Network framework (nw_browser and a connection by name); the iPad client cannot use it at all. Share one Network-framework discovery file between the two clients.

## Grounds

- pursued: the Network framework's own resolve was tried first and shown wrong — an established NWConnection reports the address it connected to (an IPv4 literal carrying its interface), which would have degraded the API key's origin binding from the host the key was entered for to an address a later tenant inherits; DNS-SD returns the SRV target name, so the binding stays a name. Verified against a server advertising on this network: a .local name and port 11535, a cancelled resolve delivering nothing, an absent service delivering one timeout failure.
