package archtest_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// The bootstrap hands over to the binary IT VERIFIED, and never to the bundle
// it is replacing.
//
// This is the trick the release gate already uses for `open`: a stub in a
// temporary directory that records how it was called. Here the stub IS the
// downloaded application's binary — the asset directory carries a bundle whose
// Contents/MacOS/gropius is a shell script — so what it records is the path the
// bootstrap executed and the arguments it passed.
//
// Why it matters: an ad-hoc-signed bundle's identity changes with every build,
// and only one release is published at a time. A bootstrap that executed the
// INSTALLED binary would be asking whatever is already on the Mac — possibly
// months old, possibly a different account's copy in the machine-wide
// applications directory — to install the release just downloaded. Handing over
// inside the verified directory makes the script and the binary one build.

// handoverFixture builds an asset directory holding a Gropius.app whose binary
// is the given shell stub, with the checksums beside it.
type handoverFixture struct {
	dir, assets, home, stubs, record, reached string
	appsBefore                                []string
}

// archiveShape is how a fixture lays out the bundle it archives.
type archiveShape int

const (
	// plainBundle: Contents/MacOS is a real directory, which is what every
	// archive the release workflow builds carries.
	plainBundle archiveShape = iota
	// linkedMacOSDir: Contents/MacOS is a symbolic link pointing OUT of the
	// archive, at a directory holding the executable.
	//
	// This is the shape a test on the LEAF cannot see
	// (iss-2609190032572500). `ditto -x -k` restores a symbolic link at any
	// component, and the path each half of the handover execs has four of
	// them; with the link at the third, `[ ! -L … ]` on the fourth finds an
	// ordinary executable file and the exec lands outside the directory the
	// checksum covered.
	linkedMacOSDir
)

func newHandoverFixture(t *testing.T, stubBody string) *handoverFixture {
	t.Helper()
	return newServerHandoverFixture(t, stubBody, plainBundle)
}

// newServerHandoverFixture builds the server's asset directory: one
// Gropius.app.zip, in the shape asked for, with the checksums beside it.
func newServerHandoverFixture(t *testing.T, stubBody string, shape archiveShape) *handoverFixture {
	t.Helper()
	if runtime.GOOS != "darwin" {
		t.Skip("install.sh refuses anything but macOS, and it needs ditto and shasum")
	}
	if runtime.GOARCH != "arm64" {
		t.Skip("install.sh refuses the server on anything but Apple Silicon")
	}
	root := repoRootDir(t)
	floor := plistString(t, filepath.Join(root, "build", "Info.plist"), minimumSystemVersionKey)
	if hostMacOSMajor(t) < majorOf(t, floor) {
		t.Skipf("install.sh refuses this Mac before it reaches the handover (it declares a floor of %s)", floor)
	}

	fx := &handoverFixture{dir: t.TempDir()}
	fx.assets = mkdirAll(t, filepath.Join(fx.dir, "assets"))
	fx.home = mkdirAll(t, filepath.Join(fx.dir, "home"))
	fx.stubs = mkdirAll(t, filepath.Join(fx.dir, "stub-bin"))
	fx.record = filepath.Join(fx.dir, "handover.txt")
	fx.reached = filepath.Join(fx.dir, "network-was-reached")

	// The bundle the asset directory carries, with a stub where the real binary
	// would be. Everything else about it is what install.sh looks for: the
	// bundle name, and an executable at Contents/MacOS/gropius.
	staged := stageBundle(t, fx.dir, "Gropius", "gropius", stubBody, shape)
	runIn(t, fx.dir, "ditto", "-c", "-k", "--keepParent", staged, filepath.Join(fx.assets, "Gropius.app.zip"))
	sums := runIn(t, fx.assets, "shasum", "-a", "256", "Gropius.app.zip")
	if err := os.WriteFile(filepath.Join(fx.assets, "SHA256SUMS.txt"), []byte(sums), 0o644); err != nil {
		t.Fatal(err)
	}

	// Any reach for the network leaves a trace, and `open` is stubbed so a
	// future change to this test cannot launch anything on a developer's Mac.
	// install.sh names its own tools by absolute path, so these stubs catch
	// what a REGRESSION would reach for rather than what the script uses.
	for _, name := range []string{"curl", "gh", "open"} {
		stub := "#!/usr/bin/env bash\ntouch " + strconv.Quote(fx.reached) + "\nexit 1\n"
		if err := os.WriteFile(filepath.Join(fx.stubs, name), []byte(stub), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	// The real /Applications is recorded and compared after the run, as the
	// asset-directory fixture next door does. Nothing in this test should write
	// there: the stub binary is what would have placed the bundle, and it
	// places nothing.
	fx.appsBefore = applicationsListing(t)
	return fx
}

// run executes install.sh in its server form, under the fixture's sandboxed
// environment, and asserts the real /Applications is untouched.
func (fx *handoverFixture) run(t *testing.T) (string, error) {
	t.Helper()
	return fx.runMode(t)
}

// runClient is the same run in the script's client form, which is where the
// second half of the handover lives: the placer this script downloads,
// verifies and execs to put the client bundle in place.
func (fx *handoverFixture) runClient(t *testing.T) (string, error) {
	t.Helper()
	return fx.runMode(t, "client")
}

func (fx *handoverFixture) runMode(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := repoRootDir(t)
	cmd := exec.Command("bash", append([]string{filepath.Join(root, "install.sh")}, args...)...)
	cmd.Dir = fx.dir
	cmd.Env = append(envWithout(os.Environ(), "PATH", "HOME", "GROPIUS_ASSET_DIR", "GITHUB_ACTIONS"),
		"PATH="+fx.stubs+string(os.PathListSeparator)+os.Getenv("PATH"),
		"HOME="+fx.home,
		"GROPIUS_ASSET_DIR="+fx.assets,
		"GITHUB_ACTIONS=true",
		"GROPIUS_HANDOVER_RECORD="+fx.record,
	)
	out, err := cmd.CombinedOutput()
	if after := applicationsListing(t); !equalStrings(fx.appsBefore, after) {
		t.Errorf("running install.sh changed the real /Applications (%v -> %v); this test must never install "+
			"anything on the machine it runs on", fx.appsBefore, after)
	}
	return string(out), err
}

// stageBundle lays out <dir>/staging-<app>/<app>.app with the given stub where
// its executable belongs, in the shape asked for, and returns the path of the
// .app to archive.
//
// linkedMacOSDir puts the executable OUTSIDE the staged bundle and links to it,
// which is exactly what a crafted archive does: ditto stores the link and
// restores it, so the extracted tree carries a component that points away from
// the bytes the checksum answered for.
func stageBundle(t *testing.T, dir, app, exe, stubBody string, shape archiveShape) string {
	t.Helper()
	bundle := filepath.Join(dir, "staging-"+app, app+".app")
	contents := mkdirAll(t, filepath.Join(bundle, "Contents"))
	macos := filepath.Join(contents, "MacOS")
	if shape == linkedMacOSDir {
		outside := mkdirAll(t, filepath.Join(dir, "outside-"+app, "MacOS"))
		writeStub(t, filepath.Join(outside, exe), stubBody)
		if err := os.Symlink(outside, macos); err != nil {
			t.Fatal(err)
		}
		return bundle
	}
	writeStub(t, filepath.Join(mkdirAll(t, macos), exe), stubBody)
	return bundle
}

func writeStub(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
}

// newClientHandoverFixture builds the CLIENT's asset directory: a GropiusChat
// bundle that is what it claims to be, and the placer archive — the server's
// own Gropius.app.zip, which the client half downloads, verifies and execs to
// place the bundle — in the shape asked for.
func newClientHandoverFixture(t *testing.T, placerStub string, placerShape archiveShape) *handoverFixture {
	t.Helper()
	if runtime.GOOS != "darwin" {
		t.Skip("install.sh refuses anything but macOS, and it needs ditto and shasum")
	}
	if runtime.GOARCH != "arm64" {
		t.Skip("install.sh refuses either app on anything but Apple Silicon")
	}
	root := repoRootDir(t)
	floor := plistString(t, filepath.Join(root, "build", "Info.plist"), minimumSystemVersionKey)
	if hostMacOSMajor(t) < majorOf(t, floor) {
		t.Skipf("install.sh refuses this Mac before it reaches the handover (it declares a floor of %s)", floor)
	}
	// The client half quits a running copy of the app before it downloads the
	// placer, and the destination it picks is the real /Applications whenever
	// that is writable. A developer running GropiusChat from there would have
	// it quit by this test, so the test stands down instead: what it holds is
	// checked in CI and on every Mac that is not using the app right now.
	if exec.Command("/usr/bin/pgrep", "-qf", "/Applications/GropiusChat.app/Contents/MacOS/").Run() == nil {
		t.Skip("a GropiusChat is running from /Applications and the client half would quit it")
	}

	fx := &handoverFixture{dir: t.TempDir()}
	fx.assets = mkdirAll(t, filepath.Join(fx.dir, "assets"))
	fx.home = mkdirAll(t, filepath.Join(fx.dir, "home"))
	fx.stubs = mkdirAll(t, filepath.Join(fx.dir, "stub-bin"))
	fx.record = filepath.Join(fx.dir, "handover.txt")
	fx.reached = filepath.Join(fx.dir, "network-was-reached")

	// The client bundle carries no binary of its own — install.sh only checks
	// that the archive holds GropiusChat.app — and the placer is the archive
	// this test is about.
	chat := stageBundle(t, fx.dir, "GropiusChat", "GropiusChat", "#!/usr/bin/env bash\nexit 0\n", plainBundle)
	runIn(t, fx.dir, "ditto", "-c", "-k", "--keepParent", chat, filepath.Join(fx.assets, "GropiusChat.app.zip"))
	placer := stageBundle(t, fx.dir, "Gropius", "gropius", placerStub, placerShape)
	runIn(t, fx.dir, "ditto", "-c", "-k", "--keepParent", placer, filepath.Join(fx.assets, "Gropius.app.zip"))
	sums := runIn(t, fx.assets, "shasum", "-a", "256", "GropiusChat.app.zip", "Gropius.app.zip")
	if err := os.WriteFile(filepath.Join(fx.assets, "SHA256SUMS.txt"), []byte(sums), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"curl", "gh", "open"} {
		stub := "#!/usr/bin/env bash\ntouch " + strconv.Quote(fx.reached) + "\nexit 1\n"
		writeStub(t, filepath.Join(fx.stubs, name), stub)
	}
	fx.appsBefore = applicationsListing(t)
	return fx
}

// recordingStub is a binary that writes down the path it was executed from and
// the arguments it was given, then exits with the given status.
func recordingStub(status int) string {
	return "#!/usr/bin/env bash\n" +
		"printf '%s\\n' \"$0\" >> \"$GROPIUS_HANDOVER_RECORD\"\n" +
		"printf '%s\\n' \"$*\" >> \"$GROPIUS_HANDOVER_RECORD\"\n" +
		"exit " + strconv.Itoa(status) + "\n"
}

// The handover runs the binary inside the VERIFIED directory, with the verified
// bundle named as what to place.
func TestTheBootstrapHandsOverToTheBinaryItVerified(t *testing.T) {
	fx := newHandoverFixture(t, recordingStub(0))

	out, err := fx.run(t)
	if err != nil {
		t.Fatalf("install.sh failed against a stub that accepted the handover: %v\n%s", err, out)
	}

	recorded, readErr := os.ReadFile(fx.record)
	if readErr != nil {
		t.Fatalf("install.sh never executed the downloaded binary at all:\n%s", out)
	}
	lines := strings.Split(strings.TrimSpace(string(recorded)), "\n")
	if len(lines) < 2 {
		t.Fatalf("the stub recorded %q, want its own path and its arguments", recorded)
	}
	executed, args := lines[0], lines[1]

	if !strings.Contains(executed, "/extract/Gropius.app/Contents/MacOS/gropius") {
		t.Errorf("the bootstrap executed %q, which is not the binary inside the directory it verified", executed)
	}
	for _, installed := range []string{"/Applications/Gropius.app", filepath.Join(fx.home, "Applications")} {
		if strings.HasPrefix(executed, installed) {
			t.Errorf("the bootstrap executed %q — the INSTALLED bundle, which need not be the build it just "+
				"verified; an ad-hoc signature's identity changes with every build and only one release is "+
				"published at a time", executed)
		}
	}
	if !strings.Contains(args, "install --bundle ") {
		t.Errorf("the handover's arguments were %q, which do not ask the binary to install the verified bundle", args)
	}
	if !strings.Contains(args, "/extract/Gropius.app") {
		t.Errorf("the handover named %q as the bundle to place, which is not the one that was verified", args)
	}
	// Under CI the handover places the bundle and stops: a runner has no
	// console for an authentication panel and no business provisioning a
	// runtime, and the script says so twice rather than passing in silence.
	if !strings.Contains(args, "--place-only") {
		t.Errorf("the handover's arguments were %q; in CI it must place and stop", args)
	}
	for _, warning := range []string{"CI has no console", "says NOTHING about the firewall grant"} {
		if !strings.Contains(out, warning) {
			t.Errorf("the run does not say what it did not exercise (%q missing):\n%s", warning, out)
		}
	}
	if _, err := os.Stat(fx.reached); err == nil {
		t.Errorf("install.sh reached for the network although GROPIUS_ASSET_DIR named the assets:\n%s", out)
	}
}

// A bundle whose binary predates the verbs refuses the handover with exit 2 —
// the code cmd/gropius has always used for an argument it does not know — and
// the bootstrap reports a version mismatch. In no case does a handover start a
// server.
func TestTheBootstrapRefusesAnOldBinarysHandover(t *testing.T) {
	fx := newHandoverFixture(t, recordingStub(2))

	out, err := fx.run(t)
	if err == nil {
		t.Fatalf("install.sh reported success although the downloaded binary refused the handover:\n%s", out)
	}
	for _, want := range []string{"does not carry the lifecycle verbs", "exit 2", "Nothing was launched"} {
		if !strings.Contains(out, want) {
			t.Errorf("the refusal does not carry %q, so a person cannot tell a version mismatch from a broken "+
				"install:\n%s", want, out)
		}
	}
	if _, err := os.Stat(fx.reached); err == nil {
		t.Errorf("install.sh launched something, or reached the network, after a refused handover:\n%s", out)
	}
}

// AND NEITHER HALF EXECS OUT OF AN ARCHIVE THAT CARRIES A SYMBOLIC LINK.
//
// `ditto -x -k` restores links at any path component, and both halves used to
// test only the LEAF of the path they exec: `[ ! -L "$VERIFIED_BIN" ]` and
// `[ ! -L "$PLACER" ]` on Contents/MacOS/gropius, with nothing said about
// Gropius.app, Contents or MacOS. A link at any of those three sends the exec
// outside the directory the checksum covered while the leaf test passes on an
// ordinary executable file — one component of four, guarded as if it closed
// the case (iss-2609190032572500).
//
// What closes it is the whole tree rather than the path: neither bundle this
// product publishes contains a symbolic link, so an archive that carries one
// anywhere is not a Gropius archive and nothing is run out of it.
//
// It takes a release whose bytes verify to plant one, so the checksum is the
// control that really stands here. These two hold the belt-and-braces guard to
// being what it says it is.
func TestTheServerHalfRefusesAnArchiveLinkedAtAnIntermediateComponent(t *testing.T) {
	fx := newServerHandoverFixture(t, recordingStub(0), linkedMacOSDir)

	out, err := fx.run(t)
	if err == nil {
		t.Errorf("install.sh reported success against an archive whose Contents/MacOS is a symbolic link:\n%s", out)
	}
	if _, statErr := os.Stat(fx.record); statErr == nil {
		recorded, _ := os.ReadFile(fx.record)
		t.Fatalf("install.sh EXECUTED a binary reached through a symbolic link out of the verified directory "+
			"(the stub recorded %q); the leaf test passes on the ordinary file the link points at, so the "+
			"refusal has to be the whole extracted tree:\n%s", recorded, out)
	}
	if !strings.Contains(out, "symbolic link") {
		t.Errorf("the refusal does not say a symbolic link is why, so a person cannot tell it from a broken "+
			"download:\n%s", out)
	}
	if _, err := os.Stat(fx.reached); err == nil {
		t.Errorf("install.sh launched something, or reached the network, after refusing the archive:\n%s", out)
	}
}

// The client half's placer is the same exec through the same four components,
// out of a second archive, and it is refused the same way — before `gropius
// place` is reached and so before anything is put anywhere.
func TestTheClientHalfRefusesAPlacerLinkedAtAnIntermediateComponent(t *testing.T) {
	fx := newClientHandoverFixture(t, recordingStub(0), linkedMacOSDir)

	out, err := fx.runClient(t)
	if err == nil {
		t.Errorf("install.sh reported success against a placer archive whose Contents/MacOS is a symbolic link:\n%s", out)
	}
	if _, statErr := os.Stat(fx.record); statErr == nil {
		recorded, _ := os.ReadFile(fx.record)
		t.Fatalf("install.sh EXECUTED a placer reached through a symbolic link out of the verified directory "+
			"(the stub recorded %q):\n%s", recorded, out)
	}
	if !strings.Contains(out, "symbolic link") {
		t.Errorf("the refusal does not say a symbolic link is why:\n%s", out)
	}
	if _, err := os.Stat(fx.reached); err == nil {
		t.Errorf("install.sh launched something, or reached the network, after refusing the placer archive:\n%s", out)
	}
}
