package archtest_test

import (
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// build/Info.plist's LSMinimumSystemVersion is the SERVER's declaration of the
// macOS floor and client/Info.plist's is the CLIENT's, and the product has ONE
// floor: macOS 27 on Apple Silicon, for both apps (the maintainer's decision,
// 2026-09-20). So the two plists must declare the same value, and every other
// surface that names a minimum is checked against it -- the installer's refusal,
// the client build script's deployment target, and the requirement sentence on
// each page. A second copy of the number cannot drift from the plists
// unnoticed. Prose that names the version some other way is not read here and
// still needs a human edit when the floor moves. Drift is the failure this
// guard exists to prevent:
//
//   - a bundle minimum below the floor lets an unsupported Mac install the app
//     and discover the problem at runtime, instead of being refused at launch;
//   - a deployment target above the bundle minimum is worse, because then
//     Launch Services admits a supported Mac and dyld kills the app at exec;
//   - a page stating a different number sends the reader to the wrong Mac;
//   - a CI runner below the floor turns the installer gate -- the only thing
//     that executes install.sh -- into a silent skip.
//
// So raising the floor is one edit to each Info.plist plus whatever this test
// then reports as out of step. Keep each value a plain "27.0"-style string:
// other tests read them from here as the source of truth.

const (
	minimumSystemVersionKey = "LSMinimumSystemVersion"
	// installerFloorAssignment names the one shell variable install.sh gates
	// both installs on.
	installerFloorAssignment = "MIN_MACOS_MAJOR"
)

// withdrawnInstallerNames are the names the two-floor era needed: a
// client-specific floor, the tag of the one older release a Mac between the
// floors was served from, and the separate release the placer came from. One
// floor needs none of them, and each is listed here so the concept cannot come
// back by halves -- a name reintroduced without the rest is a second release
// path with nothing holding its checksums to it.
var withdrawnInstallerNames = []string{
	"MIN_MACOS_MAJOR_CLIENT",
	"KEPT_CLIENT_TAG",
	"PLACER_RELEASE_PATH",
}

// TestEverySurfaceDeclaresTheSameMacOSFloor holds every declared minimum to the
// one floor the two plists agree on.
func TestEverySurfaceDeclaresTheSameMacOSFloor(t *testing.T) {
	root := repoRootDir(t)

	serverFloor := plistString(t, filepath.Join(root, "build", "Info.plist"), minimumSystemVersionKey)
	clientFloor := plistString(t, filepath.Join(root, "client", "Info.plist"), minimumSystemVersionKey)
	if serverFloor != clientFloor {
		t.Fatalf("build/Info.plist declares %s %q and client/Info.plist %q; the product has one floor for both apps",
			minimumSystemVersionKey, serverFloor, clientFloor)
	}
	major, _, found := strings.Cut(serverFloor, ".")
	if !found || major == "" {
		t.Fatalf("the bundles declare %s %q, which is not a major.minor version", minimumSystemVersionKey, serverFloor)
	}

	t.Run("chat client deployment target", func(t *testing.T) {
		raw := readRepoFile(t, root, filepath.Join("client", "build.sh"))
		targets := regexp.MustCompile(`-apple-macos([0-9][0-9.]*)`).FindAllStringSubmatch(raw, -1)
		if len(targets) == 0 {
			t.Fatal("client/build.sh names no -apple-macos deployment target; the floor it compiles against is unchecked")
		}
		for _, m := range targets {
			// A target above the bundle minimum is the dangerous direction: the
			// bundle admits the Mac and the binary then refuses to start on it.
			if m[1] != major+".0" {
				t.Errorf("client/build.sh compiles against %s; the bundles declare %q", m[0], serverFloor)
			}
		}
	})

	t.Run("installer refusal", func(t *testing.T) {
		raw := readRepoFile(t, root, "install.sh")
		gates := regexp.MustCompile(`(?m)^`+installerFloorAssignment+`=([0-9]+)`).FindAllStringSubmatch(raw, -1)
		if len(gates) != 1 {
			t.Fatalf("install.sh makes %d %s assignments, want exactly one: an unsupported Mac is downloaded to before Launch Services refuses the app, and two floors are how the second release path grew",
				len(gates), installerFloorAssignment)
		}
		if gates[0][1] != major {
			t.Errorf("install.sh gates on %s; the plists declare major %q", gates[0][0], major)
		}
	})

	t.Run("update verb floor", func(t *testing.T) {
		// The third route onto a Mac. install.sh refuses below the floor and
		// Launch Services refuses to open the bundle; `dessau update` goes
		// through neither, and it quits the server and swaps the bundle aside
		// before the system is asked anything. Its constant is a fourth copy of
		// this number and is held here with the rest.
		raw := readRepoFile(t, root, filepath.Join("internal", "lifecycle", "floor.go"))
		m := regexp.MustCompile(`(?m)^const minMacOSMajor = ([0-9]+)$`).FindStringSubmatch(raw)
		if m == nil {
			t.Fatal("internal/lifecycle/floor.go declares no minMacOSMajor; `dessau update` would replace a working install on a Mac that cannot open the new bundle")
		}
		if m[1] != major {
			t.Errorf("internal/lifecycle declares minMacOSMajor = %s; the plists declare major %q", m[1], major)
		}
	})

	t.Run("user-facing prose", func(t *testing.T) {
		// Each page states the requirement in the same words, so a reader who
		// meets it twice meets one number. \b keeps "macOS 27" from being
		// satisfied by "macOS 275".
		phrase := "Requires macOS " + major
		stated := regexp.MustCompile(regexp.QuoteMeta(phrase) + `\b`)
		for _, rel := range []string{
			"README.md",
			filepath.Join("docs", "getting-started.md"),
			filepath.Join("client", "README.md"),
		} {
			if !stated.MatchString(readRepoFile(t, root, rel)) {
				t.Errorf("%s does not state %q", rel, phrase)
			}
		}
	})

	t.Run("no withdrawn floor on any page", func(t *testing.T) {
		// What the one floor withdrew must not survive as a sentence: a page
		// naming the old floor, the release a Mac below it was served from, or
		// a universal client, sends a reader to a Mac this product refuses --
		// and site-src/ui.json is read here too, because the labels it carries
		// are printed on the landing page.
		withdrawn := []*regexp.Regexp{
			regexp.MustCompile(`macOS 26\b`),
			regexp.MustCompile(`v0\.6\.0`),
			// Narrow on purpose: "Universal Clipboard" is a true thing this
			// client does, and it is not a build that was withdrawn.
			regexp.MustCompile(`(?i)universal (build|binary|client|slice)`),
		}
		for _, rel := range []string{
			"README.md",
			filepath.Join("docs", "getting-started.md"),
			filepath.Join("client", "README.md"),
			filepath.Join("site-src", "ui.json"),
		} {
			raw := readRepoFile(t, root, rel)
			for _, re := range withdrawn {
				if m := re.FindString(raw); m != "" {
					t.Errorf("%s names %q; the floor is macOS %s on Apple Silicon and nothing is kept below it", rel, m, major)
				}
			}
		}
	})
}

// TestTheInstallerKeepsNoSecondReleasePath is what makes "no fallback" a
// checkable claim rather than a sentence. Two floors gave install.sh a second
// release to fetch a bundle from, which meant a second SHA256SUMS.txt fetched
// in the same run -- the one place where one run verified two origins. One
// floor means one release path, one checksums file, and no name left over that
// could quietly reintroduce either.
func TestTheInstallerKeepsNoSecondReleasePath(t *testing.T) {
	raw := readRepoFile(t, repoRootDir(t), "install.sh")

	for _, name := range withdrawnInstallerNames {
		if strings.Contains(raw, name) {
			t.Errorf("install.sh still names %s; the kept-client release path is withdrawn (DECISIONS.md 2026-09-20)", name)
		}
	}
	if got := len(regexp.MustCompile(`(?m)^RELEASE_PATH=`).FindAllString(raw, -1)); got != 1 {
		t.Errorf("install.sh assigns RELEASE_PATH %d times, want exactly one: the bundle and the placer come from the same release", got)
	}
	if strings.Contains(raw, "download/$") {
		t.Error("install.sh builds a release path from a tag; every asset comes from the latest release")
	}
	if got := len(regexp.MustCompile(`fetch "SHA256SUMS\.txt"`).FindAllString(raw, -1)); got != 1 {
		t.Errorf("install.sh fetches SHA256SUMS.txt %d times, want exactly one: a second checksums file is a second origin verified in one run", got)
	}
}

// macOSRunnerFloors maps every runner label this repository may use for a
// macOS job onto the macOS major that image runs. It is an ALLOW-LIST and the
// test below fails on a label that is not in it, because the failure being
// guarded against is a label whose macOS major is lower than the floor OR
// unknown -- `macos-latest` lags the newest system by a year or more, and a new
// preview label says nothing about its system at all. A label's major is a
// claim about GitHub's image that the checkout cannot verify, so each entry
// names how it was verified.
var macOSRunnerFloors = map[string]int{
	// Verified in this repository's own release run of 2026-09-19, whose job
	// log reports "Image: xcode-27-arm64" under "Operating System: macOS 27.0".
	"xcode-27": 27,
	// GitHub's dated images, whose label carries the major.
	"macos-26": 26,
	"macos-15": 15,
	"macos-14": 14,
}

// nonMacOSRunners are the labels that run no macOS job, so the floor says
// nothing about them.
var nonMacOSRunners = map[string]bool{"ubuntu-latest": true, "ubuntu-24.04": true, "ubuntu-22.04": true}

// macOSGatedJobs are the workflow jobs that must run on a Mac at or above the
// floor, named as "<workflow>:<job>". Each one runs the Go suite, and the
// installer gate inside it is the ONLY thing anywhere that executes install.sh
// -- on a runner below the floor that guard skips itself, and on a runner that
// is not a Mac at all it skips even earlier, before the fatality that is meant
// to make the skip loud. Naming the jobs here is what stops the guard being
// removed by editing one label.
var macOSGatedJobs = []string{"ci.yml:check", "release.yml:verify", "release.yml:release", "release.yml:rehearsal"}

// TestEveryMacOSRunnerIsAtOrAboveTheFloor closes the coupling the installer
// gate's own comment records: that gate is the only thing that executes
// install.sh, and a runner image below the bundle's floor turns it into a
// silent no-op -- reported, if at all, as a message about runner provisioning
// rather than about the floor that was raised. Here it is caught in the suite,
// beside the plist that moved, and a label nobody has checked is a failure
// rather than a shrug.
func TestEveryMacOSRunnerIsAtOrAboveTheFloor(t *testing.T) {
	root := repoRootDir(t)
	floor := plistString(t, filepath.Join(root, "build", "Info.plist"), minimumSystemVersionKey)
	major, err := strconv.Atoi(strings.SplitN(floor, ".", 2)[0])
	if err != nil {
		t.Fatalf("build/Info.plist declares %s %q, which is not a major.minor version", minimumSystemVersionKey, floor)
	}

	runners := workflowRunners(t, root)
	if len(runners) == 0 {
		t.Fatal(".github/workflows names no runner, so this guard reads nothing")
	}

	// Every label, whatever job it belongs to: an unknown one is a failure,
	// because this test cannot say whether it is a Mac at all.
	seen := map[string]bool{}
	for job, label := range runners {
		if nonMacOSRunners[label] {
			continue
		}
		known, ok := macOSRunnerFloors[label]
		if !ok {
			t.Errorf("%s runs on %q, a label this test knows nothing about: add it to macOSRunnerFloors with how its macOS major was verified, or to nonMacOSRunners", job, label)
			continue
		}
		seen[label] = true
		if known < major {
			t.Errorf("%s runs on %s (macOS %d), below the bundle floor of %s: the installer gate is the only thing that executes install.sh, and it would be skipped silently there",
				job, label, known, floor)
		}
	}

	// And the jobs that must be on such a Mac are, by name. A label edit that
	// moved one of them to Linux would otherwise pass the loop above.
	for _, job := range macOSGatedJobs {
		label, ok := runners[job]
		if !ok {
			t.Errorf("%s is not in .github/workflows any more; it runs the Go suite, whose installer gate is the only thing that executes install.sh", job)
			continue
		}
		known, ok := macOSRunnerFloors[label]
		if !ok || known < major {
			t.Errorf("%s runs on %q, which is not a Mac at or above the floor of %s", job, label, floor)
		}
	}
	if len(seen) == 0 {
		t.Errorf("no workflow job runs on a macOS runner, so nothing executes install.sh in CI")
	}
}

// workflowRunners reads every job's runs-on out of .github/workflows, keyed
// "<file>:<job>". The workflows here are plain enough to read with an indent
// rule -- a job is a two-space key under `jobs:` and its `runs-on:` is four
// spaces in -- which keeps a YAML parser out of the test suite for one field.
func workflowRunners(t *testing.T, root string) map[string]string {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(root, ".github", "workflows", "*.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal(".github/workflows holds no workflow, so this guard reads nothing")
	}
	jobHeading := regexp.MustCompile(`^  ([A-Za-z0-9_-]+):\s*$`)
	runsOn := regexp.MustCompile(`^    runs-on:\s*(\S+)\s*$`)
	out := map[string]string{}
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		file, job, inJobs := filepath.Base(path), "", false
		for _, line := range strings.Split(string(raw), "\n") {
			if line == "jobs:" {
				inJobs = true
				continue
			}
			if !inJobs {
				continue
			}
			if m := jobHeading.FindStringSubmatch(line); m != nil {
				job = m[1]
				continue
			}
			if m := runsOn.FindStringSubmatch(line); m != nil && job != "" {
				out[file+":"+job] = m[1]
			}
		}
	}
	return out
}

// repoRootDir is the checkout root, two levels up from internal/archtest.
func repoRootDir(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func readRepoFile(t *testing.T, root, rel string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// plistString returns the <string> value an XML property list declares for key
// at the top level of its bundle dictionary. It walks the token stream rather
// than unmarshalling into a struct because a plist <dict> is a flat run of
// sibling <key>/<value> pairs -- a shape the xml package cannot express as a
// struct. Only the standard library is used: reading the build assets must not
// cost the test suite a dependency.
func plistString(t *testing.T, path, key string) string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	dec := xml.NewDecoder(f)
	var (
		// depth 0 is outside <plist>, 1 inside it, 2 inside the bundle <dict>.
		depth  int
		wanted bool
	)
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("parsing %s: %v", path, err)
		}
		switch elem := tok.(type) {
		case xml.StartElement:
			// Only the bundle dictionary's own keys count. A key of the same
			// name nested inside a value (client/Info.plist's
			// NSAppTransportSecurity dict, say) must not satisfy the lookup
			// while the top-level key is missing.
			if depth == 2 && (elem.Name.Local == "key" || elem.Name.Local == "string") {
				var v string
				if err := dec.DecodeElement(&v, &elem); err != nil {
					t.Fatalf("parsing %s: %v", path, err)
				}
				// DecodeElement consumed the closing tag, so depth is unchanged.
				if elem.Name.Local == "key" {
					wanted = strings.TrimSpace(v) == key
				} else if wanted {
					return strings.TrimSpace(v)
				}
				continue
			}
			if depth == 2 {
				// Any other element is this key's value, so a key whose value
				// is not a string can never pick up a later, unrelated one.
				wanted = false
			}
			depth++
		case xml.EndElement:
			depth--
		}
	}
	t.Fatalf("%s declares no top-level %s", path, key)
	return ""
}

// plistStringArray returns the <array> of <string> values an XML property list
// declares for key at the top level of its bundle dictionary. It is the array
// sibling of plistString and walks the token stream the same way, for the same
// reason: a plist <dict> is a flat run of sibling <key>/<value> pairs, not a
// shape the xml package can unmarshal into a struct.
func plistStringArray(t *testing.T, path, key string) []string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	dec := xml.NewDecoder(f)
	var (
		// depth 0 is outside <plist>, 1 inside it, 2 inside the bundle <dict>.
		depth  int
		wanted bool
	)
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("parsing %s: %v", path, err)
		}
		switch elem := tok.(type) {
		case xml.StartElement:
			// As in plistString, only the bundle dictionary's own keys count,
			// so a key of the same name nested inside some other value cannot
			// satisfy the lookup while the top-level key is missing.
			if depth == 2 && elem.Name.Local == "key" {
				var v string
				if err := dec.DecodeElement(&v, &elem); err != nil {
					t.Fatalf("parsing %s: %v", path, err)
				}
				wanted = strings.TrimSpace(v) == key
				continue
			}
			if depth == 2 && elem.Name.Local == "array" && wanted {
				var arr struct {
					Values []string `xml:"string"`
				}
				if err := dec.DecodeElement(&arr, &elem); err != nil {
					t.Fatalf("parsing %s: %v", path, err)
				}
				for i, v := range arr.Values {
					arr.Values[i] = strings.TrimSpace(v)
				}
				return arr.Values
			}
			if depth == 2 {
				// Any other element is the current key's value, so a key whose
				// value is not an array can never pick up a later, unrelated one.
				wanted = false
			}
			depth++
		case xml.EndElement:
			depth--
		}
	}
	t.Fatalf("%s declares no top-level %s array", path, key)
	return nil
}
