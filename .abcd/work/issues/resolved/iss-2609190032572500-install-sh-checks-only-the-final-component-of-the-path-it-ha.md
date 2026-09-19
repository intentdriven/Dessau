---
schema_version: 1
id: "iss-2609190032572500"
slug: "install-sh-checks-only-the-final-component-of-the-path-it-ha"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "adversarial read of the client placement handover"
origin: researcher-authored
production_mode: hand-written
found_at: "install.sh"
resolution: "Both halves of install.sh now call refuse_symlinks on the tree ditto has just unpacked — the server's bundle and the client's placer — and refuse the archive if it carries a symbolic link at any path component, before anything reads, walks or executes it. The two leaf tests on Contents/MacOS/gropius are gone: they answered for one component of four. Held by two archtests that execute the script against an archive whose Contents/MacOS is a link out of the bundle, one per half."
impact: fix
---

install.sh checks only the FINAL component of the path it hands over to: both halves test [ ! -L "$VERIFIED_BIN" ] and [ ! -L "$PLACER" ] on Contents/MacOS/gropius, and nothing tests whether Gropius.app, Contents or MacOS is itself a symbolic link. ditto -x restores symbolic links from the archive, so an archive carrying a link at any intermediate component would send the exec outside the directory the checksum covered, with the leaf test passing. It takes a release whose bytes verify — so the checksum is the control that really stands here and the leaf test is belt-and-braces — but the guard is written as if it closed the case, and it closes one component of four. A walk of the components, or resolving the path and refusing one that leaves the extract directory, would close it.

## Grounds

- pursued: we expect a whole-tree refusal to be the honest shape because a Gropius bundle contains no symbolic link at all, so the rule needs no per-component reasoning and cannot be defeated by a component nobody thought of; wrong if a bundle ever has to ship a link — an embedded framework's Versions/Current — in which case the refusal fires on a legitimate release and the rule has to become a walk of the executed path's components instead.
