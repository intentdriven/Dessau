#!/bin/bash
#
# One-line installer for Gropius.
#
#   Server (menu-bar, Apple Silicon only):
#     curl -fsSL https://raw.githubusercontent.com/intentdriven/Gropius/main/install.sh | bash
#
#   Client (GropiusChat, Apple Silicon, macOS 27 — the same floor as the server):
#     curl -fsSL https://raw.githubusercontent.com/intentdriven/Gropius/main/install.sh | bash -s -- client
#
# This is a BOOTSTRAP, and only a bootstrap: it does what has to happen before a
# Gropius binary exists on this Mac. It downloads the latest release, verifies
# it against the checksums published beside it, clears the quarantine attribute,
# and then hands over to a `gropius` binary it verified. For the server that is
# `gropius install` inside the bundle it just downloaded, and everything after
# it is the binary's own work: the staged swap, the firewall grant, the MLX
# runtime, the per-user command and the launch. A binary fetched by curl is not
# Gatekeeper-quarantined, so no right-click-to-open dance.
#
# The client carries no binary of its own, so it hands over too — to `gropius
# place`, from the server's archive, verified the same way. Nothing here moves a
# bundle with `mv`: it nests into a destination that already exists as a
# directory and follows one that is a symbolic link, and no test-then-move in
# shell closes that (iss-2609111454146700).
#
# THE INTERPRETER TOO. The shebang is /bin/bash rather than /usr/bin/env bash,
# which is the one program a script naming everything else by absolute path
# would otherwise still resolve through PATH. It is unreachable through the
# documented `curl … | bash`, where the shebang is a comment, and reachable the
# moment somebody makes this file executable and runs it. macOS ships
# /bin/bash, and nothing here needs a version newer than it.
#
# EVERY COMMAND IS NAMED BY ABSOLUTE PATH. `curl | bash` runs with the invoking
# user's PATH, which routinely puts user-writable directories ahead of /usr/bin,
# and on a Mac several accounts share that is an account-to-account boundary
# (iss-2609081435387952). It is also a defence against ambiguity with no
# attacker in it at all: a release step once resolved a name to a tool that was
# not the tool meant, invisibly (iss-9). The rule is held by
# TestInstallerPinsEveryCommandItRuns, which refuses any command here that is
# not written as a path.
#
# NOTHING READS STANDARD INPUT. Under `curl … | bash` the remaining text of this
# script IS standard input, so a read would consume the rest of the installer.
# The one step that needs consent — the firewall grant — is asked for by the
# binary, through the system authorisation panel, which is also the only way a
# standard account can answer it at all. The one `read` below is the checksums
# scan, and its loop is fed by a REDIRECT from a file: a redirection replaces
# the loop's standard input, so the script's own is never touched.
set -euo pipefail

REPO="intentdriven/Gropius"

# Trust model: the download is checked against the SHA256SUMS.txt published on
# the same GitHub Release, and every asset carries a GitHub build-provenance
# attestation binding it to the release workflow run. There is no offline
# signing key. To check provenance yourself before running this script:
#   gh attestation verify Gropius.app.zip --repo intentdriven/Gropius
# Building from source (see the README) is the escape hatch.
#
# That sentence has exactly one exception, and it is refused outside CI. When
# GITHUB_ACTIONS=true, GROPIUS_ASSET_DIR points this run at a local directory,
# and then BOTH the bundle and the SHA256SUMS.txt it is checked against are read
# from that directory: the verification proves the directory is self-consistent
# and NOTHING about where its contents came from. Anywhere else the seam is a
# hard refusal (see ASSET_DIR below), because a caller who can set one
# environment variable would otherwise substitute the whole integrity control
# silently — this is the script the README tells people to pipe into bash.

mode="${1:-server}"
case "$mode" in
server)
	APP="Gropius"
	ASSET="Gropius.app.zip"
	;;
client)
	APP="GropiusChat"
	ASSET="GropiusChat.app.zip"
	# The archive the placer is taken from. The client has no binary of its
	# own, so the one that places its bundle is the server's.
	PLACER_ASSET="Gropius.app.zip"
	;;
*)
	echo "usage: install.sh [server|client]" >&2
	exit 2
	;;
esac

die() {
	echo "error: $*" >&2
	exit 1
}

[ "$(/usr/bin/uname -s)" = "Darwin" ] || die "Gropius is macOS only."

# Both bundles declare the same minimum, so Launch Services refuses either on
# anything older. Refuse here instead — before the download and before anything
# is written — so an unsupported Mac is turned away rather than half-installed.
# The major lives in this one place; build/Info.plist and client/Info.plist are
# the values it must match, and a test in the server's suite holds the three
# together. There is one floor for both apps and nothing below it: a Mac under
# it is refused, not served an older build (DECISIONS.md 2026-09-20).
MIN_MACOS_MAJOR=27
# `|| macos_version=""` is load-bearing: under `set -e` a bare assignment takes
# the command substitution's status, so a missing sw_vers would abort the script
# with no message at all instead of reaching the refusal below. An unreadable or
# non-numeric version leaves the major empty or unusable, and `[` refuses then
# too.
macos_version="$(/usr/bin/sw_vers -productVersion 2>/dev/null)" || macos_version=""
macos_major="${macos_version%%.*}"
[ "${macos_major:-0}" -ge "$MIN_MACOS_MAJOR" ] ||
	die "$APP requires macOS $MIN_MACOS_MAJOR (this Mac runs ${macos_version:-an unreadable version})."
# Where every asset comes from: the latest release, and only ever that one. The
# client's bundle and the placer that puts it in place are both fetched from
# here, so the SHA256SUMS.txt fetched once below is the checksums file for
# everything this run verifies — one origin per run, and an integrity control
# that cannot differ from itself because it is not written twice.
RELEASE_PATH="latest/download"

# Apple Silicon, for either mode, and there is no build for anything else: the
# floor above runs on no Intel Mac, the server needs Metal for MLX, and both
# bundles are placed by a `gropius` binary that is an Apple Silicon build. So an
# Intel Mac is turned away here, before anything is downloaded, with the floor
# named rather than being left half-installed.
# `uname -m` reports x86_64 in a Rosetta-translated shell (common with x86_64
# Homebrew), so also ask the kernel whether the hardware is Apple Silicon.
if [ "$(/usr/bin/uname -m)" != "arm64" ] &&
	[ "$(/usr/sbin/sysctl -n hw.optional.arm64 2>/dev/null)" != "1" ]; then
	die "$APP requires macOS $MIN_MACOS_MAJOR on Apple Silicon (this Mac is $(/usr/bin/uname -m)). macOS $MIN_MACOS_MAJOR runs on no Intel Mac, and there is no build for one."
fi

tmp="$(/usr/bin/mktemp -d)"
trap '/bin/rm -rf "$tmp"' EXIT
zip="$tmp/$ASSET"

# Where the release assets come from. Normally the latest published Release;
# GROPIUS_ASSET_DIR points this run at a local directory holding the same asset
# names instead. That is what lets the release workflow run THIS script against
# the artefacts it has just built, before they are published — the only moment
# an installer broken in the tagged tree can still be stopped
# (iss-2609081257343394).
#
# It is a CI-ONLY seam and it is refused everywhere else, because `fetch` serves
# BOTH the bundle and the SHA256SUMS.txt the bundle is verified against: point
# it at a directory and the checksum step compares bytes with their own digest,
# which is no integrity control at all. An attacker-authored zip plus a matching
# checksums file would otherwise install, have its quarantine cleared, get a
# firewall rule and be launched, printing "Checksum OK." on the way past.
#
# GITHUB_ACTIONS is a weak gate — it is only an environment variable, and a
# caller who sets one can set two. It is not trying to stop that caller; it
# stops the seam from being reachable by accident, by a stray export, or by a
# tutorial that tells someone to set it, and it makes the substitution loud when
# it does happen.
ASSET_DIR="${GROPIUS_ASSET_DIR:-}"
if [ -n "$ASSET_DIR" ]; then
	[ "${GITHUB_ACTIONS:-}" = "true" ] ||
		die "GROPIUS_ASSET_DIR is a CI-only seam for the release workflow's installer gate, and is refused outside GitHub Actions. It makes this script install from a local directory and verify the checksums against a file in that same directory, so the verification would prove nothing about where the bundle came from. Unset it and rerun to install the published release."
	echo "warning: GROPIUS_ASSET_DIR is set — installing from $ASSET_DIR, NOT from the published GitHub Release." >&2
	echo "warning: the checksums are read from that same directory, so the \"Checksum OK.\" below proves only that the directory is self-consistent. It proves NOTHING about the origin of what is being installed, and no attestation is checked." >&2
fi

# fetch <asset-name> <dest>: download a release asset.
#
# The GitHub CLI fallback this function used to carry is gone. `gh` is a
# third-party tool with no fixed location — it cannot be named by absolute path
# on a machine nobody here controls — so keeping it meant one PATH-resolved
# binary in the middle of the only download path there is, for a fallback that
# helped a private fork and a transient error. Building from source and `make
# install` cover both, and neither is a script the README tells people to pipe
# into bash.
# Every asset comes from the one release named in RELEASE_PATH: the bundle and
# the placer are fetched by the same function, from the same release, and
# verified against the one checksums file published with it. Neither can skip
# the checks around the other, and no run verifies two origins.
fetch() {
	local name="$1" dest="$2" release="$RELEASE_PATH"
	if [ -n "$ASSET_DIR" ]; then
		/bin/cp "$ASSET_DIR/$name" "$dest" ||
			die "could not read $name from $ASSET_DIR."
		return 0
	fi
	# -q first: ignore any curlrc that could re-point the connection while the
	# URL still reads github.com; --proto pins HTTPS end to end, redirects
	# included. The asset and the checksums that verify it come from this same
	# origin, so the transport is the thing to pin.
	/usr/bin/curl -q --proto =https --proto-redir =https -fsSL -o "$dest" "https://github.com/$REPO/releases/$release/$name" ||
		die "could not download $name from releases/$release. Check your network and retry, or build from source (see the README)."
}

echo "Downloading ${APP}…"
fetch "$ASSET" "$zip"

# Verify the download is exactly what the release workflow built, BEFORE
# unpacking it, clearing its quarantine, or placing it. The checksums file comes
# from the same Release as the asset.
echo "Verifying checksum…"
fetch "SHA256SUMS.txt" "$tmp/SHA256SUMS.txt"

# THE SCOPE, decided here and not by shasum.
#
# `shasum -c` answers for the files the checksums file NAMES. This used to carry
# `--ignore-missing`, to skip the other app's line — and with it, names that are
# absent are skipped while names that are present and irrelevant are verified and
# reported as a pass. Nothing in that invocation asserted that the archive it
# protects was in scope, so a checksums file naming any other readable file with
# a correct digest (/dev/null will do) exited 0 with the download never looked
# at, and the unverified archive went on to be unpacked, have its quarantine
# cleared and hand its own binary the install (iss-2609120417422598). The
# reassurance was the worst part: the run printed "Checksum OK." on its way past.
#
# So the file is narrowed to the line for THIS archive before shasum sees it,
# and --ignore-missing goes with the narrowing: there is then exactly one name,
# it is the file just downloaded, and a pass is a statement about that file.
# Narrowing the input is not making the verdict — the digest is still computed
# and compared by /usr/bin/shasum, which stays the only thing in this script
# that decides whether bytes match. The same three moves are taken by the
# update verb, in internal/lifecycle/updatefetch.go.
#
# A line is `<digest><separator><name>`, where the separator is two spaces for a
# text-mode digest and " *" for a binary one. The name must be EXACTLY the
# archive: a line naming a PATH is refused rather than matched, because
# "/somewhere/$ASSET" would verify a file this script never downloaded.
#
# No new tool to name by absolute path: the scan is bash's own `read`, and it is
# fed by a REDIRECT from the checksums file, so it does not touch the standard
# input that `curl … | bash` is feeding this script from.
#
# It is a function because the client fetches TWO archives — its own bundle and
# the placer that puts the bundle in place — and an integrity control written
# twice is an integrity control that will differ once.
#
# verify <asset-name> <checksums-file> <directory>: refuse unless the bytes of
# <directory>/<asset-name> match the line <checksums-file> carries for exactly
# that name.
verify() {
	local asset="$1" sums="$2" dir="$3"
	local line digest name sums_line="" scoped=".gropius-checksum" checksum_output verified=""
	while IFS= read -r line || [ -n "$line" ]; do
		line="${line%$'\r'}"
		digest="${line%% *}"
		name="${line#* }"
		name="${name# }"
		name="${name#\*}"
		[ "${#digest}" -eq 64 ] || continue
		case "$digest" in
		*[!0-9a-fA-F]*)
			continue
			;;
		esac
		# Belt and braces: an exact match against $asset already refuses a name
		# with a directory in it, and this says so where a reader is looking.
		case "$name" in
		*/*)
			continue
			;;
		esac
		[ "$name" = "$asset" ] || continue
		sums_line="$line"
		break
	done <"$sums"
	[ -n "$sums_line" ] ||
		die "the checksums published with the release carry no line for $asset — an empty or truncated file, a page that is not a checksums file at all, or a checksums file that answers about something else. Nothing verified the download. Refusing to install."

	# The one-line file shasum is actually pointed at, in the staging directory
	# mktemp made, which nothing else can write.
	printf '%s\n' "$sums_line" >"$dir/$scoped"

	# /usr/bin/shasum, not shasum: this is the only integrity control in the whole
	# install path, so the binary that runs it must not be one PATH chose. A planted
	# shim is the hostile case, but the ordinary one matters too — a Homebrew
	# coreutils or another implementation earlier on PATH need not behave the same,
	# and a checksum check that silently stops checking is worse than none, because
	# it still prints reassurance. Verified against this exact binary: the archive's
	# own line and matching bytes exit 0, while a wrong hash exits non-zero, and an
	# empty file, an HTML error page, a checksums file naming no downloaded file and
	# one naming a file that is not the download are all refused above, before
	# shasum is reached.
	#
	# The output is captured rather than discarded so the failure says which cause
	# fired. Sending it to /dev/null made every cause look identical, and this is
	# the one message a user most needs to be able to act on.
	if ! checksum_output="$( cd "$dir" && /usr/bin/shasum -a 256 -c "$scoped" 2>&1 )"; then
		echo "$checksum_output" >&2
		die "checksum mismatch for $asset — the download is corrupt or tampered. Refusing to install."
	fi

	# And the pass is read as a pass for THAT FILE, by its own line, rather than as
	# an exit status meaning "nothing I was told about was wrong" or as a substring:
	# a line for "/somewhere/$asset" ends in the same characters as the one this is
	# looking for. The newlines around both sides are what make it a line match.
	case $'\n'"$checksum_output"$'\n' in
	*$'\n'"$asset: OK"$'\n'*)
		verified="yes"
		;;
	esac
	if [ -z "$verified" ]; then
		echo "$checksum_output" >&2
		die "/usr/bin/shasum did not report $asset as verified, so the download was not checked against the checksums published with the release. Refusing to install."
	fi
}

# refuse_symlinks <asset-name> <directory>: refuse the archive if the tree
# `ditto -x -k` has just restored out of it carries a symbolic link ANYWHERE.
#
# The whole tree, and not the path that is about to be executed, because a path
# test answers for one component at a time. `ditto -x -k` restores a link at any
# component and `-x` follows one, and each half of this script execs something
# four components deep: a link at Gropius.app, at Contents or at MacOS sends the
# exec outside the directory the checksum covered while a test on the leaf finds
# an ordinary executable file and passes. That is what the two leaf tests here
# used to be, and it closed one component of four (iss-2609190032572500).
#
# It is a rule the archives can carry because neither bundle this product
# publishes contains a symbolic link — no embedded frameworks, no
# Versions/Current — so an archive with one anywhere in it is not an archive
# this script should be running a binary out of, whatever component it sits at.
# Should a bundle ever need to ship a link, this refusal is where that shows up,
# loudly, rather than in what gets executed.
#
# Only a release whose bytes verify can plant one, which is what the checksum
# above is for; this costs a scan of a just-unpacked directory and does not
# depend on that being true. `find` is given neither -H nor -L, so it does not
# follow what it finds, and a scan that cannot be completed is a refusal: a tree
# nothing can read is not a tree to execute out of.
refuse_symlinks() {
	local asset="$1" dir="$2" links="" first=""
	links="$(/usr/bin/find "$dir" -type l)" ||
		die "the unpacked $asset could not be read, so nothing here can say where its executables point. Refusing to run anything out of it."
	[ -n "$links" ] || return 0
	first="${links%%$'\n'*}"
	# The name is reported as a path inside the archive, with control characters
	# replaced: it was written by whoever built the archive, and a message about
	# a hostile archive is the last place to let that archive write to somebody's
	# terminal.
	first="${first#"$dir"/}"
	first="${first//[[:cntrl:]]/?}"
	die "$asset carries a symbolic link ($first), and a Gropius archive carries none. A link at any component of a path this script executes would send that exec outside the directory the checksum covered. Refusing to run anything out of it."
}

verify "$ASSET" "$tmp/SHA256SUMS.txt" "$tmp"
echo "Checksum OK."

# Where the client goes is chosen by this script — the placement itself is not
# (see THE PLACEMENT below). Chosen before unpacking, so the line that says
# where it is going is printed before the work starts.
#
# /Applications is root:admin and group-writable, so a standard (non-admin)
# account cannot write it — and on a Mac several people share, the account that
# most needs the chat client is exactly the one without admin rights.
# ~/Applications is the per-user location macOS already understands: Spotlight
# and Launchpad index it, and it needs no privileges.
#
# Chosen over elevating on purpose. A destination has a per-user equivalent, and
# elevating to write /Applications would install the app for every account when
# only one asked for it. The server's own copy of this rule lives in
# internal/lifecycle, beside the swap that places both bundles.
if [ "$mode" = "client" ]; then
	if [ -w /Applications ]; then
		DEST="/Applications"
	else
		DEST="$HOME/Applications"
		/bin/mkdir -p "$DEST" || die "no write access to /Applications, and $DEST could not be created."
		echo "No write access to /Applications (this account is not an administrator) — installing to $DEST instead."
	fi
	echo "Installing ${APP}.app to ${DEST}…"
else
	echo "Unpacking ${APP}…"
fi

/usr/bin/ditto -x -k "$zip" "$tmp/extract" || die "could not unpack $ASSET."
# Before anything reads, walks or executes what was unpacked.
refuse_symlinks "$ASSET" "$tmp/extract"
[ -d "$tmp/extract/$APP.app" ] || die "$ASSET did not contain $APP.app."
# Safe to clear the quarantine now: we have verified this .app is the exact
# artifact the release workflow built. (curl downloads are usually not
# quarantined anyway, but a proxy or prior run might have tagged it, which would
# otherwise block launch.)
/usr/bin/xattr -dr com.apple.quarantine "$tmp/extract/$APP.app" 2>/dev/null || true

if [ "$mode" = "server" ]; then
	# HAND OVER TO THE BINARY THIS SCRIPT VERIFIED — the one inside the bundle
	# in $tmp, never the one already installed. The bootstrap and the binary it
	# calls are then always the same build, which is what makes the two halves
	# of an install a single thing rather than a negotiation between a script
	# from one release and an application from another.
	VERIFIED_BIN="$tmp/extract/$APP.app/Contents/MacOS/gropius"
	# Every component of this path is an ordinary directory or file: the whole
	# extracted tree was scanned for symbolic links above, before anything
	# walked it. A test here would answer for the last component only.
	[ -x "$VERIFIED_BIN" ] ||
		die "$ASSET carries no executable at $APP.app/Contents/MacOS/gropius — refusing to install it."

	handover=("$VERIFIED_BIN" install --bundle "$tmp/extract/$APP.app")
	if [ "${GITHUB_ACTIONS:-}" = "true" ]; then
		# Say so, loudly, rather than passing silently. The release gate runs
		# this script to prove the installer works against the artefacts just
		# built; a runner has no console to answer an authentication panel and
		# no business downloading an MLX runtime, so the handover places the
		# bundle and stops there. A green run that does not say what it did not
		# look at is a false green.
		echo "warning: passing --place-only — CI has no console for an authentication panel and no business provisioning a runtime." >&2
		echo "warning: a green result from this run therefore says NOTHING about the firewall grant, the MLX provisioning or the launch. They are exercised only by a real install on a Mac." >&2
		handover+=(--place-only)
	fi

	# The exit status is read rather than left to `set -e`, because ONE of its
	# values means something specific: a bundle whose binary predates the
	# lifecycle verbs refuses an argument it does not know with exit 2, and that
	# refusal is a version mismatch rather than a failed install. In no case
	# does this script start a server — the binary does that, or nothing does.
	status=0
	"${handover[@]}" || status=$?
	if [ "$status" -eq 2 ]; then
		die "the downloaded $APP does not carry the lifecycle verbs: its binary refused \`install\` with exit 2, which is how a build older than this bootstrap refuses an argument it has never heard of. The script and the bundle are different builds. Nothing was launched."
	elif [ "$status" -ne 0 ]; then
		die "gropius install stopped (exit $status) — the message above says at which stage. Nothing was launched."
	fi
	exit 0
fi

# From here it is the client, whose placement this script hands to a Gropius
# binary below.
#
# Quit a running copy first. LaunchServices' `open` activates an
# already-running process instead of launching the new binary, so an upgrade
# over a live app would report success while the old version keeps running.
if /usr/bin/pgrep -qf "$DEST/$APP.app/Contents/MacOS/" 2>/dev/null; then
	echo "Quitting the running ${APP}…"
	/usr/bin/osascript -e "quit app \"$APP\"" >/dev/null 2>&1 || true
	for _ in $(/usr/bin/seq 1 20); do
		/usr/bin/pgrep -qf "$DEST/$APP.app/Contents/MacOS/" || break
		/bin/sleep 0.5
	done
	if /usr/bin/pgrep -qf "$DEST/$APP.app/Contents/MacOS/" 2>/dev/null; then
		echo "warning: $APP is still running; quit it and relaunch to finish the upgrade." >&2
	fi
fi

# THE PLACEMENT, AND WHY IT IS NOT DONE HERE.
#
# `mv` nests into a destination that already exists as a directory and follows
# one that is a symbolic link, exiting 0 in both cases; it has no dependable
# "fail if the destination exists" mode, and any test-then-move in shell is a
# race by construction. /Applications is drwxrwxr-x root:admin, so on a Mac
# several people share, any admin-group account can win that window, and
# ~/Applications is writable by the same user's other processes. That was the
# server's defect once (iss-2609081310071028) and the client's until now
# (iss-2609111454146700).
#
# Closing it needs os.Rename semantics — rename(2) replaces a symbolic link
# rather than following it, and refuses a destination that is a non-empty
# directory — which is Go, not shell. The swap lives in
# internal/lifecycle/swap.go with its behavioural tests beside it, and `gropius
# place` is the verb that runs it: it stages inside the destination under an
# unguessable name, renames the installed bundle ASIDE, renames the new one in,
# and removes the set-aside copy only once the new one is in place.
#
# GropiusChat carries no binary, so the placer is the server's own — downloaded
# and verified here, and RUN FROM THE DIRECTORY THAT VERIFICATION COVERED. Never
# the gropius already installed on this Mac: that copy may be months old or
# another account's, and "run what you verified" is the same rule the server
# half keeps.
echo "Downloading the placer…"
placer_zip="$tmp/$PLACER_ASSET"
fetch "$PLACER_ASSET" "$placer_zip"
# The checksums already fetched: the placer comes from the same release as the
# bundle, so the file that verified one verifies the other.
verify "$PLACER_ASSET" "$tmp/SHA256SUMS.txt" "$tmp"
/usr/bin/ditto -x -k "$placer_zip" "$tmp/placer" || die "could not unpack $PLACER_ASSET."
# The same rule as the bundle above, and for the same reason: the exec below is
# four components deep in a directory a downloaded archive laid out.
refuse_symlinks "$PLACER_ASSET" "$tmp/placer"
PLACER="$tmp/placer/Gropius.app/Contents/MacOS/gropius"
[ -x "$PLACER" ] ||
	die "$PLACER_ASSET carries no executable at Gropius.app/Contents/MacOS/gropius — refusing to place $APP.app with it."

# The exit status is read rather than left to `set -e`, because ONE of its
# values means something specific: a binary that predates the placement verb
# refuses an argument it does not know with exit 2, and that refusal is a
# version mismatch rather than a failed placement. The verb prints where the
# bundle went, so nothing is echoed here.
status=0
"$PLACER" place --bundle "$tmp/extract/$APP.app" --into "$DEST" || status=$?
if [ "$status" -eq 2 ]; then
	die "the downloaded Gropius does not carry the \`place\` verb: its binary refused the argument with exit 2, which is how a build older than this bootstrap refuses an argument it has never heard of. The script and the release are different builds. $APP.app was not placed."
elif [ "$status" -ne 0 ]; then
	die "gropius place stopped (exit $status) — the message above says why. $APP.app was not placed."
fi

# CI has no desktop to launch into, and a chat client left running on a runner
# outlives the job. Everywhere else this is the last thing the script does, so
# the line above is what proves a run reached the end.
if [ "${GITHUB_ACTIONS:-}" = "true" ]; then
	echo "warning: not launching $DEST/$APP.app (CI) — this run does not exercise the launch." >&2
else
	/usr/bin/open "$DEST/$APP.app"
fi
/bin/cat <<'DONE'

Open GropiusChat, then point it at your Gropius server: the address from the
server's Connect tab without the trailing /v1 (GropiusChat adds the path
itself).
DONE
