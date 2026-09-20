package lifecycle

import (
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

// THE PRODUCT FLOOR, on the one surface neither the installer nor Launch
// Services covers.
//
// macOS 27 on Apple Silicon is the floor for both apps (DECISIONS.md
// 2026-09-20). Two of the three ways a build reaches a Mac already refuse
// below it: install.sh refuses before it downloads anything, and Launch
// Services refuses to open a bundle whose LSMinimumSystemVersion is above the
// running system. The third is `gropius update`, and it is the destructive
// one — it quits the running server and swaps the installed bundle aside
// before anything about the new build is asked of the system. A Mac below the
// floor that ran that verb would be left with the old bundle gone, a new one
// Launch Services will not open, and a report saying the new version is
// installed: dyld admits the staged binary when it is run directly, because
// the Go linker's minimum is not the bundle's.
//
// So the verb refuses first, by the same rule and in the same words as the
// bootstrap, before the download and before the quit.
//
// minMacOSMajor is a fourth copy of a number that lives in build/Info.plist,
// and TestEverySurfaceDeclaresTheSameMacOSFloor holds it to that plist along
// with the other three.
const minMacOSMajor = 27

// hostMacOSVersion reads this Mac's product version — "27.0", "26.5.2" — the
// same value `sw_vers -productVersion` prints, without a subprocess or a PATH
// lookup. kern.osproductversion is the kernel's own copy of it.
func hostMacOSVersion() (string, error) {
	return unix.Sysctl("kern.osproductversion")
}

// belowFloor says whether a product version is under the floor, and hands back
// the sentence to print when it is. A version that cannot be read or cannot be
// parsed is below the floor: this gate stands in front of a command that
// destroys the installed bundle, so the unreadable case fails closed, exactly
// as the bootstrap's `[ "${macos_major:-0}" -ge … ]` does.
func belowFloor(version string, err error) (string, bool) {
	if err != nil || strings.TrimSpace(version) == "" {
		return "Gropius requires macOS " + strconv.Itoa(minMacOSMajor) +
			", and this Mac's version could not be read.", true
	}
	major, _, _ := strings.Cut(version, ".")
	n, convErr := strconv.Atoi(major)
	if convErr != nil {
		return "Gropius requires macOS " + strconv.Itoa(minMacOSMajor) +
			" (this Mac reports " + Quote(version) + ", which is not a version this can read).", true
	}
	if n < minMacOSMajor {
		return "Gropius requires macOS " + strconv.Itoa(minMacOSMajor) +
			" (this Mac runs " + version + ").", true
	}
	return "", false
}
