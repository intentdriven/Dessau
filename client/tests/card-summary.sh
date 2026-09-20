#!/bin/bash
# The other piece of the client that can be checked without a device: the
# exchange count behind a sidebar card's summary. CardSummary.swift imports
# nothing, so `swiftc` compiles it with the main beside this script and runs
# the result. A test in the server's suite runs this when the Swift toolchain
# is here, and says so loudly when it is not.
set -euo pipefail
cd "$(dirname "$0")"

OUT="$(mktemp -d)"
trap 'rm -rf "$OUT"' EXIT

xcrun swiftc -swift-version 6 -o "$OUT/card-summary" \
    ../DessauChat/CardSummary.swift CardSummaryTests.swift

"$OUT/card-summary"
