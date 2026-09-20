package archtest_test

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The iPad client (itd-2609180943290800) is the Mac client's own Swift files
// built a second way, and two of its promises live outside Swift: the bundle
// an iPad will actually run, and the discovery file both clients share. Neither
// is compiled by `go test`, so both are read out of the client's own files
// here, as the Mac client's promises are.

// ipadSigningGuard matches the guard's own test expression, not a mention of
// the two variables: an earlier spelling of this matched any use of the names
// followed by any later `exit 1`, which the script has several of, so deleting
// the whole refusal left it green. The refusal has to be a refusal: a script
// that carried on would write a bundle no iPad installs and say so only at
// `devicectl` time, in the installer's words rather than ours.
var ipadSigningGuard = regexp.MustCompile(
	`if \[ -z "\$\{IPAD_SIGNING_IDENTITY:-\}" \] \|\| \[ -z "\$\{IPAD_PROFILE:-\}" \]; then`)

// ipadSigningRefusalExits matches that guard ending in a refusal rather than a
// warning: the guard's body reaches `exit 1` before the `fi` that closes it.
var ipadSigningRefusalExits = regexp.MustCompile(
	`(?s)if \[ -z "\$\{IPAD_SIGNING_IDENTITY:-\}" \].*?\n[ \t]*exit 1\n[ \t]*fi`)

// ipadDeviceFamily matches the one device family an iPad-only bundle declares.
// UIDeviceFamily is an array of integers, so it is read from the plist source
// rather than through plistStringArray, which reads <string> members.
var ipadDeviceFamily = regexp.MustCompile(
	`(?s)<key>UIDeviceFamily</key>\s*<array>(.*?)</array>`)

// clientServiceTypeDeclarations matches every declaration of the browsed
// service type across the client's sources, so that the shared discovery file
// can be held to being its one home.
var clientServiceTypeDeclarations = regexp.MustCompile(
	`(?m)^\s*let\s+dessauServiceType\s*=`)

// TestChatClientIPadBuildIsSignedForADevice holds the iPad build script to the
// signing the intent's scope condition fixes: the maintainer's own free
// personal team, over USB, to their own iPad.
//
// A device bundle is only installable when it carries a development signature
// and the matching provisioning profile. An ad-hoc signature (`--sign -`) is
// accepted by codesign, produces a bundle that looks built, and is refused by
// every iPad -- so the script must never reach for it, and must refuse loudly
// when the identity or the profile is not in the environment. The launch
// screen and the device family are the other two keys an iPad bundle cannot
// start without: no UILaunchScreen and the app is letterboxed as an iPhone
// app; no UIDeviceFamily 2 and it is not an iPad app at all.
func TestChatClientIPadBuildIsSignedForADevice(t *testing.T) {
	root := repoRootDir(t)
	script := readRepoFile(t, root, filepath.Join("client", "build-ipad.sh"))

	t.Run("refuses without an identity and a profile", func(t *testing.T) {
		if !ipadSigningGuard.MatchString(script) {
			t.Error("client/build-ipad.sh does not test IPAD_SIGNING_IDENTITY and IPAD_PROFILE before " +
				"a device build; it would write a bundle no iPad installs")
		}
		if !ipadSigningRefusalExits.MatchString(script) {
			t.Error("client/build-ipad.sh checks for the identity and the profile but does not stop " +
				"without them; a warning leaves an uninstallable bundle behind")
		}
	})

	t.Run("never ad-hoc signs", func(t *testing.T) {
		if strings.Contains(script, "--sign -") {
			t.Error("client/build-ipad.sh ad-hoc signs the bundle; an iPad refuses an ad-hoc signature, " +
				"and the build would look like it had succeeded")
		}
	})

	t.Run("app intents are processed for iOS", func(t *testing.T) {
		if !strings.Contains(script, "--platform-family iOS") {
			t.Error("client/build-ipad.sh does not tell the App Intents processor --platform-family iOS; " +
				"the metadata would describe the wrong system and Shortcuts would list nothing")
		}
	})

	plist := readRepoFile(t, root, filepath.Join("client", "Info-iPad.plist"))

	t.Run("launch screen", func(t *testing.T) {
		if !strings.Contains(plist, "<key>UILaunchScreen</key>") {
			t.Error("client/Info-iPad.plist declares no UILaunchScreen; iPadOS then runs the app " +
				"in a compatibility window rather than full screen")
		}
	})

	t.Run("iPad only", func(t *testing.T) {
		m := ipadDeviceFamily.FindStringSubmatch(plist)
		if m == nil {
			t.Fatal("client/Info-iPad.plist declares no UIDeviceFamily array; the bundle claims no device")
		}
		members := regexp.MustCompile(`<integer>\s*([0-9]+)\s*</integer>`).FindAllStringSubmatch(m[1], -1)
		if len(members) != 1 || members[0][1] != "2" {
			t.Errorf("client/Info-iPad.plist declares UIDeviceFamily %v; the iPad client is family 2 alone "+
				"(an iPhone layout is out of scope)", m[1])
		}
	})
}

// TestChatClientIPadDeclaresOneFloor holds the iPad bundle's minimum to the
// version its build script compiles against, as the macOS floor guard holds
// the Mac pair.
//
// The dangerous direction is a deployment target above the bundle minimum:
// iPadOS admits the iPad on the plist's word and dyld then kills the app at
// exec, which reads to the person as an app that will not open.
func TestChatClientIPadDeclaresOneFloor(t *testing.T) {
	root := repoRootDir(t)
	floor := plistString(t, filepath.Join(root, "client", "Info-iPad.plist"), "MinimumOSVersion")
	script := readRepoFile(t, root, filepath.Join("client", "build-ipad.sh"))

	m := regexp.MustCompile(`(?m)^DEPLOYMENT_TARGET="([0-9][0-9.]*)"`).FindStringSubmatch(script)
	if m == nil {
		t.Fatal("client/build-ipad.sh sets no DEPLOYMENT_TARGET; the floor it compiles against is unchecked")
	}
	if m[1] != floor {
		t.Errorf("client/build-ipad.sh compiles against iOS %s; client/Info-iPad.plist declares MinimumOSVersion %q",
			m[1], floor)
	}
	// And the triples it builds are that same floor, whichever SDK is chosen.
	targets := regexp.MustCompile(`-apple-ios([0-9][0-9.]*)`).FindAllStringSubmatch(script, -1)
	for _, target := range targets {
		if target[1] != floor {
			t.Errorf("client/build-ipad.sh names the target %s; the bundle's minimum is %q", target[0], floor)
		}
	}
}

// macOSOnlyCalls are the calls and the import that exist on macOS alone. Each
// one outside an os(macOS) branch is an iPad build that fails to compile --
// found at build time on a Mac, which is the last place anyone looks.
var macOSOnlyCalls = []string{"import AppKit", "NSPasteboard", "NSApp", "controlActiveState"}

// TestChatClientGuardsTheMacOnlyCalls holds the other half of the iPad
// intent's last criterion: one source, two systems, with every macOS-only call
// inside a branch that only the Mac compiles.
func TestChatClientGuardsTheMacOnlyCalls(t *testing.T) {
	root := repoRootDir(t)
	for name, src := range clientSources(t, root) {
		// A tiny reader of the conditional-compilation blocks: for each line,
		// whether some enclosing branch is the macOS one. Nesting is why this
		// is a stack rather than a flag.
		type branch struct{ guardsMac, inMacBranch bool }
		var stack []branch
		inMac := func() bool {
			for _, b := range stack {
				if b.guardsMac && b.inMacBranch {
					return true
				}
			}
			return false
		}
		for n, line := range strings.Split(src, "\n") {
			trimmed := strings.TrimSpace(line)
			switch {
			case strings.HasPrefix(trimmed, "#if "):
				condition := strings.TrimPrefix(trimmed, "#if ")
				stack = append(stack, branch{
					guardsMac:   strings.Contains(condition, "os(macOS)"),
					inMacBranch: condition == "os(macOS)",
				})
				continue
			case trimmed == "#else":
				if len(stack) > 0 {
					top := &stack[len(stack)-1]
					top.inMacBranch = top.guardsMac && !top.inMacBranch
				}
				continue
			case trimmed == "#endif":
				if len(stack) > 0 {
					stack = stack[:len(stack)-1]
				}
				continue
			case strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "///"):
				// A comment may name any of them; only code has to compile.
				continue
			}
			if inMac() {
				continue
			}
			for _, call := range macOSOnlyCalls {
				if strings.Contains(line, call) {
					t.Errorf("client/DessauChat/%s:%d uses %s outside an `#if os(macOS)` branch; "+
						"the iPad build would not compile", name, n+1, call)
				}
			}
		}
	}
}

// TestChatClientSharesDiscoveryWithoutNetService holds the one discovery file
// both clients compile.
//
// NetService is deprecated on every platform at 27.2 and is not available to
// the iPad client at all (iss-2609180959409672), so the resolver is on the
// Network framework and the whole of discovery lives in one file. Two copies
// of the service type is the failure this guards: the copy the iPad compiles
// could drift from the server's type, and a browse for the wrong type returns
// an empty list rather than an error.
func TestChatClientSharesDiscoveryWithoutNetService(t *testing.T) {
	root := repoRootDir(t)
	all := clientSources(t, root)

	if _, ok := all["Discovery.swift"]; !ok {
		t.Fatal("client/DessauChat/Discovery.swift is missing; the two clients share no discovery file")
	}

	for name, src := range all {
		if strings.Contains(src, "NetService") {
			t.Errorf("client/DessauChat/%s mentions NetService; it is deprecated on macOS and absent "+
				"on iPadOS, so discovery resolves on the Network framework", name)
		}
	}

	var declared []string
	for name, src := range all {
		if clientServiceTypeDeclarations.MatchString(src) {
			declared = append(declared, name)
		}
	}
	if len(declared) != 1 || declared[0] != "Discovery.swift" {
		t.Errorf("dessauServiceType is declared in %v; it belongs to the shared discovery file alone",
			declared)
	}
}
