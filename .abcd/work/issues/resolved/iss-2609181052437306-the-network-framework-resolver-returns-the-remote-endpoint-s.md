---
schema_version: 1
id: "iss-2609181052437306"
slug: "the-network-framework-resolver-returns-the-remote-endpoint-s"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "security review of feat/ipad-client, 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/GropiusChat/Discovery.swift"
resolution: "ServiceResolver resolves through DNSServiceResolve (import dnssd) and stores the SRV target host NAME again, validated as before and built into the URL field by field; the service reference is deallocated on the callbacks' own queue. Verified against a live server on this network: a .local name and port 11535, a cancelled resolve delivers nothing, a missing service delivers one timeout."
impact: fix
resolved_by:
  intent: "itd-2609180943290800"
  spec: "spc-2609181023476174"
---

The Network-framework resolver returns the remote endpoint's IPv4 literal, never the .local host name the NetService resolver returned, so a picked server is stored as an address and the API key's origin binding degrades to something an address takeover can inherit: a hostile advertiser at a reused address is handed the bearer token when the row is tapped, because the picker resolves, stores and connects with no further interaction. Resolve through DNS-SD (DNSServiceResolve gives the SRV target host name and port, on every platform, not deprecated), validate the name as before, and keep the binding to a name.
