---
schema_version: 1
id: "iss-2609120438396694"
slug: "gropius-install-prints-five-error-texts-unredacted-install-g"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
resolution: "The four warning and refusal lines in install.go now pass the error text through redact against this account's home, the way the failure line always did; held by TestInstallWarningsAreRedacted, four rows, three watched red. The refusal at the top of RunInstall stays: it is the guard's own sentence or an unresolved home directory, neither carrying a path of this account's. Test aaf435b, fix 087d5ea."
impact: fix
resolved_by:
  commit: "087d5ea"
---

gropius install prints five error texts unredacted: install.go lines 108, 174, 191, 212 and 243 write err.Error() straight to the terminal, and 243 redacts the destination on the same line while the error beside it is not, which defeats the redaction. The update verb's quit warning was fixed on fix/update-verify-scope; the install verb's siblings were not, and there is no test holding any lifecycle warning line to the redaction rule. Found by the security review of that branch.

## Grounds

- pursued: we expect every path an install prints to arrive inside an error sentence somebody else wrote, so redacting the text rather than the operands is what holds; a leak the test table does not cover would show it wrong.
