---
schema_version: 1
id: "iss-2609190103062546"
slug: "internal-gateway-control-go-s-applysettings-computes-whether"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "adversarial review of the client-pairing diff before landing"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gateway/control.go"
resolution: "tls_port joins the restart expression in applySettings, so a save that moves the port paired clients use answers restart=true. Held by TestASaveThatChangesTheTLSPortAsksForARestart, watched failing first with 'a save that moved the port paired clients use reported restart=false'."
impact: fix
---

internal/gateway/control.go's applySettings computes whether a save needs a restart from port, host, bind_mode, advertise, decode_concurrency and idle_timeout_sec, but not from tls_port. The TLS listeners are acquired once in runServer from the configuration at launch, so changing the port paired clients use does nothing until the next start — and the save reports restart=false, which tells the operator the opposite. This is the silence iss-2609091751184914 recorded for advertise, in a second field.

## Grounds

- pursued: we expect naming tls_port in the restart expression to stop a save claiming an effect it does not have, because the listener is acquired once at launch; wrong if the listeners ever become re-bindable, at which point the line becomes false in the other direction
