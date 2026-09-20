#!/bin/bash
# The sidebar's search, checked without a device: SidebarSearch.swift imports
# Foundation and nothing of the client's, so `swiftc` compiles it with the main
# beside this script and runs the result. A test in the server's suite runs
# this when the Swift toolchain is here, and says so loudly when it is not.
set -euo pipefail
cd "$(dirname "$0")"

OUT="$(mktemp -d)"
trap 'rm -rf "$OUT"' EXIT

xcrun swiftc -swift-version 6 -o "$OUT/sidebar-search" \
    ../DessauChat/SidebarSearch.swift SidebarSearchTests.swift

"$OUT/sidebar-search"
