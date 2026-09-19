#!/bin/bash
# The one piece of the client that can be checked without a device: the
# context arithmetic of the built-in model's trim. ContextBudget.swift imports
# nothing, so `swiftc` compiles it with the main beside this script and runs
# the result. A test in the server's suite runs this when the Swift toolchain
# is here, and says so loudly when it is not.
set -euo pipefail
cd "$(dirname "$0")"

OUT="$(mktemp -d)"
trap 'rm -rf "$OUT"' EXIT

xcrun swiftc -swift-version 6 -o "$OUT/context-budget" \
    ../GropiusChat/ContextBudget.swift ContextBudgetTests.swift

"$OUT/context-budget"
