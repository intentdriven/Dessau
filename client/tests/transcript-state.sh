#!/bin/bash
# The transcript state behind the picker's icon, checked without a device:
# which models the models list says keep no transcript, joined folded, and
# which said nothing. TranscriptState.swift imports
# nothing, so `swiftc` compiles it with the main beside this script and runs
# the result. A test in the server's suite runs this when the Swift toolchain
# is here, and says so loudly when it is not.
set -euo pipefail
cd "$(dirname "$0")"

OUT="$(mktemp -d)"
trap 'rm -rf "$OUT"' EXIT

xcrun swiftc -swift-version 6 -o "$OUT/transcript-state" \
    ../DessauChat/TranscriptState.swift TranscriptStateTests.swift

"$OUT/transcript-state"
