package archtest_test

import (
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// build/Info.plist's LSMinimumSystemVersion is the SERVER's one declaration of
// the macOS floor it supports, and client/Info.plist's is the CLIENT's: the two
// apps have different floors (the client moved to macOS 27 on 2026-09-17, the
// server stays on 26), so every other surface that names a minimum is checked
// against the plist of the app it belongs to. The server's floor holds the
// installer's server refusal and the requirement sentence in the README and the
// guide; the client's holds its build script's deployment target, the
// installer's client refusal, and the client README's sentence. A second copy
// of either number cannot drift from its plist unnoticed. Prose that names the
// version some other way is not read here and still needs a human edit when a
// floor moves. Drift is the failure this guard exists to prevent:
//
//   - a bundle minimum below the floor lets an unsupported Mac install the app
//     and discover the problem at runtime, instead of being refused at launch;
//   - a deployment target above the bundle minimum is worse, because then
//     Launch Services admits a supported Mac and dyld kills the app at exec;
//   - a page stating a different number sends the reader to the wrong Mac.
//
// So raising a floor is one edit to that app's Info.plist plus whatever this
// test then reports as out of step. Keep each value a plain "26.0"-style
// string: other tests read them from here as the source of truth.

const (
	minimumSystemVersionKey = "LSMinimumSystemVersion"
	// installerFloorAssignment names the shell variable install.sh gates the
	// server on; installerClientFloorAssignment the one it gates the client on.
	installerFloorAssignment       = "MIN_MACOS_MAJOR"
	installerClientFloorAssignment = "MIN_MACOS_MAJOR_CLIENT"
	// installerKeptClientTag names the release install.sh fetches the client
	// from on a Mac below the client's floor: the one kept 26-floor release.
	installerKeptClientTag = "KEPT_CLIENT_TAG"
)

// TestEverySurfaceDeclaresTheSameMacOSFloor holds every declared minimum to
// the plist of the app it belongs to.
func TestEverySurfaceDeclaresTheSameMacOSFloor(t *testing.T) {
	root := repoRootDir(t)

	serverFloor := plistString(t, filepath.Join(root, "build", "Info.plist"), minimumSystemVersionKey)
	serverMajor, _, found := strings.Cut(serverFloor, ".")
	if !found || serverMajor == "" {
		t.Fatalf("build/Info.plist declares %s %q, which is not a major.minor version", minimumSystemVersionKey, serverFloor)
	}
	clientFloor := plistString(t, filepath.Join(root, "client", "Info.plist"), minimumSystemVersionKey)
	clientMajor, _, found := strings.Cut(clientFloor, ".")
	if !found || clientMajor == "" {
		t.Fatalf("client/Info.plist declares %s %q, which is not a major.minor version", minimumSystemVersionKey, clientFloor)
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
			if m[1] != clientMajor+".0" {
				t.Errorf("client/build.sh compiles against %s; client/Info.plist declares %q", m[0], clientFloor)
			}
		}
	})

	t.Run("installer refusals", func(t *testing.T) {
		raw := readRepoFile(t, root, "install.sh")
		for _, c := range []struct {
			name, want string
		}{
			{installerFloorAssignment, serverMajor},
			{installerClientFloorAssignment, clientMajor},
		} {
			gates := regexp.MustCompile(`(?m)^`+c.name+`=([0-9]+)`).FindAllStringSubmatch(raw, -1)
			if len(gates) == 0 {
				t.Fatalf("install.sh sets no %s; an unsupported Mac is downloaded to before Launch Services refuses the app", c.name)
			}
			for _, m := range gates {
				if m[1] != c.want {
					t.Errorf("install.sh gates on %s; the plist declares major %q", m[0], c.want)
				}
			}
		}
	})

	t.Run("kept client release", func(t *testing.T) {
		// A Mac between the two floors installs the client from the kept
		// release; the tag named in the installer is the one the README names.
		// Whether that release is still published is the forge's state, not
		// the checkout's, and is not read here.
		raw := readRepoFile(t, root, "install.sh")
		m := regexp.MustCompile(`(?m)^` + installerKeptClientTag + `=(v[0-9]+\.[0-9]+\.[0-9]+)`).FindStringSubmatch(raw)
		if m == nil {
			t.Fatalf("install.sh sets no %s; a Mac on macOS %s has no client to install", installerKeptClientTag, serverMajor)
		}
		tag := m[1]
		if !strings.Contains(readRepoFile(t, root, "README.md"), tag) {
			t.Errorf("README.md does not name the kept client release %s that install.sh fetches from", tag)
		}
	})

	t.Run("user-facing prose", func(t *testing.T) {
		// Each page states its app's requirement in the same words, so a reader
		// who meets it twice meets one number. \b keeps "macOS 26" from being
		// satisfied by "macOS 265".
		for _, c := range []struct {
			rel, major string
		}{
			{"README.md", serverMajor},
			{filepath.Join("docs", "getting-started.md"), serverMajor},
			{"README.md", clientMajor},
			{filepath.Join("client", "README.md"), clientMajor},
		} {
			phrase := "Requires macOS " + c.major
			stated := regexp.MustCompile(regexp.QuoteMeta(phrase) + `\b`)
			if !stated.MatchString(readRepoFile(t, root, c.rel)) {
				t.Errorf("%s does not state %q", c.rel, phrase)
			}
		}
	})
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
