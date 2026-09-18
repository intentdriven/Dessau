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
---

The Mac client resolves a browsed Gropius service with NetService, which Apple deprecates at 27.2 on every platform in favour of the Network framework (nw_browser and a connection by name); the iPad client cannot use it at all. Share one Network-framework discovery file between the two clients.
