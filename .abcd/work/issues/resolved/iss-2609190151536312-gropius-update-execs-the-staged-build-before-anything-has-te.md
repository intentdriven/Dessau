---
schema_version: 1
id: "iss-2609190151536312"
slug: "gropius-update-execs-the-staged-build-before-anything-has-te"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "adversarial read of the installer's symlink handling, looking for the same defect elsewhere"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/lifecycle/updatefetch.go"
resolution: "checkStagedBundle now opens the extraction directory as its os.Root and reaches Gropius.app through it, so the root's own no-follow rule covers the bundle's name as well as every component below it; the bundle must be a real directory and the program a real file before stagedVersion execs anything."
impact: fix
---

gropius update execs the staged build before anything has tested the bundle component itself for a symbolic link: internal/lifecycle/update.go unpacks the verified archive with ditto into <staging>/extract, and checkStagedBundle (internal/lifecycle/updatefetch.go:350) opens <extract>/Gropius.app with os.OpenRoot, which resolves symlinks in the root path it is given. os.Root then refuses a link on the way to Contents/MacOS/gropius, so every component below the bundle is covered — but a link AT Gropius.app is followed, and stagedVersion (updatefetch.go:392) then execs the binary it found, outside the directory the checksum covered. PlaceBundle refuses a source bundle that is a link (place.go:90), so nothing is installed, but that refusal comes after the exec. It takes a release whose bytes verify, as its install.sh counterpart iss-2609190032572500 does. An os.Lstat on the staged bundle before checkStagedBundle, or opening the extract directory as the root and reaching the bundle through it, would close it.

## Grounds

- pursued: we expect the refusal to arrive before the exec because the root is opened on a directory this command made rather than on a name the archive chose; wrong if ditto can make the extraction directory itself a link, or if a later step re-resolves the staged path by string before the swap.
