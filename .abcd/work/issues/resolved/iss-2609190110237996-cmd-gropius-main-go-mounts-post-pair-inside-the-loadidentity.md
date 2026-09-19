---
schema_version: 1
id: "iss-2609190110237996"
slug: "cmd-gropius-main-go-mounts-post-pair-inside-the-loadidentity"
severity: "major"
category: "security"
source: "impl-review"
found_during: "adversarial security review of the client-pairing diff"
origin: researcher-authored
production_mode: hand-written
found_at: "cmd/gropius/main.go"
resolution: "The pairing route is mounted only when acquireTLSBind returned a listener, so a server with tls_port -1 or a held port offers no pairing at all. Held by TestPairingIsNotOfferedWithoutAListenerToPairOnto, which pins the order of the two statements and the guard between them."
impact: fix
---

cmd/gropius/main.go mounts POST /pair inside the LoadIdentity success branch, before acquireTLSBind has decided whether any TLS listener exists. With tls_port set to -1, or with the TLS bind failing because the port is held, nothing serves paired clients — but /pair still mints certificates, writes the clients block into config.json and answers a port number nothing is listening on. An operator who has turned the feature off still exposes an unauthenticated endpoint that writes their settings file.

## Grounds

- pursued: we expect the fix named here to close what the adversarial review reproduced, because each finding came with a repro that now has a test; wrong if the test passes over the fault the way a revocation test that reopens its connection would
