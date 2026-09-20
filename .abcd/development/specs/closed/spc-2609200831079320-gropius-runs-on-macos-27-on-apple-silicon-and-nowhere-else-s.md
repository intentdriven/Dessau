---
id: spc-2609200831079320
slug: gropius-runs-on-macos-27-on-apple-silicon-and-nowhere-else-s
intent: itd-2609200823510967
origin: researcher-authored
production_mode: hand-written
---
# Gropius runs on macOS 27 on Apple Silicon and nowhere else

## Summary

One floor, macOS 27 on Apple Silicon, for both apps. `build/Info.plist`
declares 27.0 beside `client/Info.plist`'s, `install.sh` keeps one
`MIN_MACOS_MAJOR` and one release path, and the architecture test that held
two floors and a kept-client tag holds one floor and the absence of that tag.
The pages state one requirement. CI's macOS runners move to an image at or
above the floor, because the installer gate is the only thing that executes
`install.sh` and a runner below the floor turns it into a silent skip.

## Scope

In scope: `build/Info.plist` (LSMinimumSystemVersion 27.0); `install.sh`
(one floor, the Intel branch as a plain refusal, the kept-client release
path and its second `SHA256SUMS.txt` fetch removed, the header comment
rewritten); `internal/archtest/minimum_macos_test.go` (one floor; the
kept-client subtest inverted to hold the absence of the concept; a new guard
on the workflows' macOS runner labels); `.github/workflows/ci.yml` and
`.github/workflows/release.yml` (macOS runner labels, and the arch-check
comment); `site-src/ui.json` (the install comment and the asset note that
call the client universal); `internal/sitetest` (a stale comment);
`README.md`, `docs/getting-started.md`, `client/README.md` and
`.abcd/development/procedures/installer-authorisation-panel.md`; the
changelog.

Out of scope: the iPad client's floor (`client/Info-iPad.plist`), which is
its own; `client/build.sh`'s deployment target, which is already 27 and
already one slice; `internal/runtime`'s pinned `uv` table, which still
carries an `amd64` entry keyed on `GOARCH` — reachable by a `go build` on
darwin/amd64 but unreached by any shipped build, and removing it is a change
inside a trust boundary with no user-visible effect, so it is left with this
note and a follow-on issue rather than taken along; deleting the v0.6.0 release, which is never an
agent's act.

## Approach

### One number, two plists

`build/Info.plist` rises to 27.0. `TestEverySurfaceDeclaresTheSameMacOSFloor`
stops reading two floors: it reads both plists, refuses them if they differ,
and holds every other declaration against that one value — the installer's
`MIN_MACOS_MAJOR`, `client/build.sh`'s `-apple-macos` target, and the
requirement sentence on each of the three pages. The constants for the
client-specific floor and the kept-client tag go; a new subtest holds that
neither name appears in `install.sh` any more, together with the rest of the
second-release-path machinery, so the concept cannot come back by halves.

### The installer

`MIN_MACOS_MAJOR=27` is the only floor. The version gate is unchanged in
shape: read `sw_vers`, refuse below the major, name the floor. The
client-specific branch that pointed `RELEASE_PATH` at
`download/$KEPT_CLIENT_TAG` goes, and with it `PLACER_RELEASE_PATH` as a
separate value and the conditional second `SHA256SUMS.txt` fetch: there is
one release path, so the placer and the bundle are verified against the one
checksums file already fetched. The architecture refusal loses the paragraph
about the kept universal build and becomes one sentence per mode naming
Apple Silicon and, for the client, the manual route.

### CI

`ci.yml`'s check job and `release.yml`'s `verify` and `rehearsal` jobs move
from `macos-26` to `xcode-27`, which the release run of 2026-09-19 recorded
as macOS 27.0 (`Image: xcode-27-arm64`, `Operating System: macOS 27.0`) and
which `release.yml`'s build job already uses. A new architecture test reads
every `runs-on:` in `.github/workflows/` and fails on any `macos-<major>`
label below the floor, so the coupling the installer gate's comment records
is caught by the suite rather than by a red CI job with a message about
runner provisioning.

### The pages

`README.md`'s status line and requirement paragraph, `docs/getting-started.md`
section 1 and `client/README.md`'s opening each state one requirement:
macOS 27, Apple Silicon. The paragraph about which build an older Mac gets
goes. `site-src/ui.json` is a closed list of labels the landing page may add;
its install comment and its asset note both call the client universal, and
both are corrected — the requirement sentences on that page are spans of
`docs/getting-started.md` and follow it. The hand-check procedure for the
authorisation panel names the floor as a precondition and is corrected too.

## How the acceptance criteria are met

1. A Mac below the floor is refused before any download — one
   `MIN_MACOS_MAJOR`, held to the plists by the floor guard; the refusal's
   wording is the named hand check, because `install.sh` names `sw_vers` by
   absolute path and a test cannot make this Mac look older.
2. An Intel Mac is refused with no fallback — the architecture refusal, and
   `TestTheInstallerKeepsNoSecondReleasePath`, which is what makes "no
   fallback" checkable.
3. Both bundles declare 27.0 and every other minimum agrees —
   `TestEverySurfaceDeclaresTheSameMacOSFloor`.
4. No second release path — `TestTheInstallerKeepsNoSecondReleasePath`.
5. No macOS runner below the floor —
   `TestEveryMacOSRunnerIsAtOrAboveTheFloor`.
6. The three pages state one requirement and name none of the dropped
   things — the floor guard's prose subtest and `abcd docs lint`.
7. The landing page and the bundle agree —
   `TestPageAndBundleAgreeOnTheMinimumMacOS` in `internal/sitetest`, which
   derives the number from the plist.

## Verification

`make test`, `gofmt -l .`, `go vet ./...`, `abcd docs lint`,
`abcd intent ready`. Each new or changed test was watched to fail on the
unchanged tree and pass after. Both review agents (security and ruthless)
read the diff, because `install.sh` is a trust boundary.
