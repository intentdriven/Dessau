---
id: spc-2609201245369767
slug: gropius-becomes-dessau-alice-installs-dessau-server-bob-s-de
intent: itd-2609201012167478
origin: researcher-authored
production_mode: hand-written
---
# The Dessau rename: one branch, one PR, then release 0.8.0

## Summary

This spec delivers itd-2609201012167478: the family name Gropius becomes
Dessau everywhere the product speaks, builds, installs, advertises and
publishes, on one branch `rename/dessau` and one pull request, followed by
release 0.8.0 as a breaking pre-1.0 release. It carries out
adr-2609200729102059 with the four points the maintainer settled at the
2026-09-20 planning interview (DECISIONS.md, 2026-09-20). Impact: breaking.
No migration, no shim, no dual advertising.

## Scope

In: the module path and every import; `cmd/dessau` and `cmd/dessau-site`;
the Makefile, `install.sh`, the four workflows, `build/Info.plist`,
`wrangler.jsonc`, `site-src/`; `internal/discovery`, `internal/gateway`,
`internal/config`, `internal/hub`, `internal/bridge/discord`, `internal/ui`,
`internal/archtest`, `internal/sitetest`; `client/` in full; `README.md`,
`docs/`, `client/README.md`, `CHANGELOG.md`; `.abcd/development/IDENTITY.md`,
`.abcd/docs-lint.json`, and the living records under `.abcd/` (`CONTEXT.md`,
the principles, open captures, drafts, planned intents, open specs, the ADR
index rows). The forge repository name, the release and the site deploy.

Out: shipped intents, closed specs, resolved and wontfix captures, ratified
ADR bodies, research notes and the DECISIONS.md ledger, which are dated
history and keep the name they were written under. The IANA registration
and the trademark search, which are the maintainer's.

## Approach

**Names, decided.** Family name Dessau. Components "Dessau Server" and
"Dessau Chat" as display names (`CFBundleDisplayName`, menu-bar tooltip,
About, docs prose where a component is meant). Bundles `DessauServer.app`
and `DessauChat.app`; release assets `DessauServer.app.zip` and
`DessauChat.app.zip`; binary `dessau`, build output `bin/dessau`, entry
point `cmd/dessau`, site generator `cmd/dessau-site`. Bundle identifiers
`sh.intentdriven.dessau.server` and `sh.intentdriven.dessau.chat`; the
codesign identifiers, the client's Keychain service string, its key tag,
its `Logger` subsystem and its resolver queue label all follow the client
identifier. Config directory `~/Library/Application Support/Dessau`, shared
root `/Users/Shared/Dessau` (directory mode `3775`, unchanged).

**Wire identifiers, frozen.** Service type `_dessau._tcp` in the Go
advertiser and both client property lists' `NSBonjourServices`, held to one
value by the architecture test that already pins it. The TXT record keeps
`txtvers` and every other key it has; nothing in it is renamed for
branding. `owned_by` is `dessau`; the response headers are `X-Dessau-State`
and `X-Dessau-Queue-Time`; the hub User-Agent is `dessau/1.0` with the new
repository URL; the Discord identify properties say Dessau. The old service
type is not advertised.

**The instance name.** The advertiser's instance name becomes the Mac's
Computer Name exactly as System Settings shows it, read through a memoized
`scutil --get ComputerName` beside the existing `LocalHostName` reader, with
no product prefix and no parentheses; it falls back to the LocalHostName,
then to `dessau`, when scutil answers nothing. The address-record host label
stays a DNS-safe derived label with the family prefix (`dessau-<host>`),
because claiming the machine's own name renames the machine (the existing
comment stands). A test asserts the instance name is the Computer Name and
carries no prefix; the label-length test keeps holding the 63-byte bound.

**The build and the installer.** `Makefile` `APP := DessauServer`, `BIN :=
bin/dessau`, codesign identifier `sh.intentdriven.dessau.server`, shared
cache target `/Users/Shared/Dessau`. `install.sh` `REPO=intentdriven/Dessau`,
asset names as above, the attestation comment's command updated, the
executable path `Contents/MacOS/dessau`. The release, auto-release and CI
workflows name the new bundles, binary and asset files. `client/build.sh` and
`client/build-ipad.sh` build `DessauChat.app` from `client/DessauChat/`,
signing with the new identifier; the iPad script's launch and entitlement
strings follow.

**The site.** `site-src/ui.json` `out_subdir` becomes `Dessau`, so the page
renders at `site/Dessau/index.html` and the headers rule and the test derive
`/Dessau*` from it. `wrangler.jsonc` names the Worker `dessau` and asserts
two routes, `intentdriven.sh/Dessau*` and `intentdriven.sh/Gropius*`; a
`_redirects` file at the assets root, copied by the generator like
`_headers`, sends `/Gropius` and `/Gropius/*` to `/Dessau/` with a 301, which
is what the host supports (its static-assets `_redirects`). The old Worker
is left for the maintainer to remove from the dashboard. The site workflow's
path assertions follow the new subdirectory.

**Prose and record.** README, every page under `docs/` and
`client/README.md` say Dessau, present tense, one Diátaxis type per page,
British English, with no sentence narrating the change (the changelog does
that). CHANGELOG `[Unreleased]` gains one entry under Changed with
`impact: breaking` telling Alice she re-creates state and Bob re-pairs, and
its opening line names Dessau. `IDENTITY.md` title and pitch say Dessau;
`abcd identity` reports zero surfaces adrift. `.abcd/docs-lint.json` gains
`names/superseded-product` with pattern `(?i)Gropius`, successor `Dessau`,
as the LAST change on the branch, so no earlier commit trips it. Living
records are swept by the same substitution; the dated record is not.

**Sequence.** The forge repository was renamed first, in the authorisation
sitting, so `intentdriven/Gropius` answers 301 before any reference moves.
Then one mechanical substitution over the in-scope tree with the directory
renames and the module path, then three lanes (Go and build; Swift; prose
and record) finishing the parts a substitution cannot do, integrated on
`rename/dessau`. Reviews before the PR: security on `internal/gateway`,
`internal/discovery`, `internal/config` and `install.sh`; a ruthless review
of the whole diff; a docs-currency review of `docs/` and README.

## Acceptance Criteria, and what holds each

- Install command resolves 0.8.0 from the new repository and installs
  Dessau Server: held by the installer gate tests over `install.sh` and by
  the release run, verified with `gh release view v0.8.0` and the documented
  `curl` command after the cut.
- `_dessau._tcp`, instance name the Computer Name, `txtvers` kept, no old
  type: held by the discovery tests (new instance-name assertion), the
  architecture test holding one service type across server and client, and a
  headless `make run` with a browse after the build.
- `owned_by` and the two headers: held by the gateway tests that pin them.
- Module path, identifiers, config directory, and every gate green: `make
  test`, `gofmt -l .`, `go vet ./...`, `client/build.sh`.
- Name guard and identity: `abcd docs lint` clean with Gropius banned, `abcd
  identity` zero adrift.
- Site at /Dessau/ and the old path redirected: the sitetest render, and a
  `curl -I` of both URLs after the release deploys.
- Changelog entry with `impact: breaking` and the re-create and re-pair
  sentence: read on the branch and in the release notes.

## Departures

None at the time of writing. Any departure found during the lanes is
recorded here before the PR opens.
