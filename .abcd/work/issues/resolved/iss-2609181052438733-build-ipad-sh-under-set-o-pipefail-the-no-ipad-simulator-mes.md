---
schema_version: 1
id: "iss-2609181052438733"
slug: "build-ipad-sh-under-set-o-pipefail-the-no-ipad-simulator-mes"
severity: "nitpick"
category: "bug"
source: "user-observation"
found_during: "security review of feat/ipad-client, 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "client/build-ipad.sh"
resolution: "build-ipad.sh: the swiftconstvalues fallback is '|| true' with a loud message of its own, and the simulator listing is too, so the 'no iPad simulator' refusal is reachable."
impact: fix
resolved_by:
  intent: "itd-2609180943290800"
---

build-ipad.sh: under set -o pipefail the 'no iPad simulator' message is unreachable (a non-matching grep kills the script before it), and the swiftconstvalues fallback 'ls | head -1' dies without a message on a non-matching glob; both need '|| true' as the script already does for its job list.
