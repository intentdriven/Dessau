---
schema_version: 1
id: "iss-2609111454146700"
slug: "install-sh-s-client-half-still-places-gropiuschat-app-with-t"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "security review of the lifecycle branch, 2026-09-11"
origin: researcher-authored
production_mode: hand-written
found_at: "install.sh"
resolution: "install.sh's client half no longer moves anything: the new `gropius place` verb runs internal/lifecycle's staged swap over the client bundle, and the bootstrap hands over to it from the server archive it downloaded and verified, the same way the server half hands over to `gropius install`. The verb refuses a destination directory that is a symbolic link, replaces rather than follows a symbolic link at the bundle's own name, and never nests inside a bundle already there. No `mv` is left anywhere in the script."
impact: fix
---

install.sh's client half still places GropiusChat.app with the shell's mv (rename aside, then mv the staged bundle in), which nests into an existing directory and follows a symlink at the destination — the shape iss-2609081310071028 named and the server half no longer has, because the server is placed by gropius install's staged swap in Go. The client has no binary of its own to place itself, so the shell is the only thing that can place it; on a shared Mac whose applications directory another admin-group account can write, the client's swap can still be raced. Recorded in the script's own comment; a Go placer for the client (a second verb on the server binary, or a placement helper the bootstrap calls twice) is the remedy.

## Grounds

- pursued: the client's placement now has the same rename(2) semantics the server's has, because it is the same code; wrong if the script is found moving a bundle again, or if the placer it runs is ever the copy already installed on this Mac rather than the one it verified
