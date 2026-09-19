---
schema_version: 1
id: "iss-2609190106565336"
slug: "go-mod-carries-github-com-coder-websocket-as-an-indirect-dep"
severity: "nitpick"
category: "tech-debt"
source: "impl-review"
found_during: "the adversarial security review adr-2609181004167097 obliges before a bridge lands"
origin: researcher-authored
production_mode: hand-written
found_at: "go.mod"
---

go.mod carries github.com/coder/websocket as an indirect dependency although internal/bridge/discord imports it directly. It was added with go get before anything imported it and a later go mod tidy moved it to the indirect block; the comment is now wrong about what the module is to this project, and the next tidy will move it back as a diff nobody asked for.
