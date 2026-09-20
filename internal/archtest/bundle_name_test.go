package archtest_test

import (
	"path/filepath"
	"regexp"
	"testing"
)

// The server's BUNDLE NAME is declared once, in build/Info.plist, and three
// other places have to spell it the same way or they stop working silently.
//
// `quit app "<name>"` is the one that bites. AppleScript resolves an
// application by the name Launch Services knows it under — CFBundleName — and a
// name nothing matches is not an error the caller can see: osascript answers,
// the old process keeps running, the new bundle is placed over it, and the
// update reports success while the previous build goes on serving. That is
// precisely the failure `quitRunningCopy` and the Makefile's install target
// exist to prevent, and a rename is how it comes back: the family rename moved
// the bundle to DessauServer.app and left the quit calls aimed at a name no
// bundle carries any more.
//
// The installer's `$APP` is the same value under another name: it is what the
// archive unpacks to, what `pgrep -f` looks for and what is placed in
// /Applications, so it must be the bundle's name and not the readable one.
//
// This test is deliberately about the BUNDLE name and says nothing about the
// display name: "Dessau Server" is what a person reads, "DessauServer" is what
// the filesystem and Launch Services hold, and they differ on purpose.
func TestTheServersBundleNameIsSpelledTheSameOnEverySurface(t *testing.T) {
	root := repoRootDir(t)

	want := plistString(t, filepath.Join(root, "build", "Info.plist"), "CFBundleName")
	if want == "" {
		t.Fatal("build/Info.plist declares no CFBundleName; the bundle has no name to hold anything to")
	}

	for _, surface := range []struct {
		file, why string
		pattern   *regexp.Regexp
	}{
		{
			file:    "Makefile",
			why:     "the bundle the build produces and `make install` quits, copies and opens",
			pattern: regexp.MustCompile(`(?m)^APP\s*:=\s*(\S+)\s*$`),
		},
		{
			file:    "install.sh",
			why:     "the bundle the bootstrap unpacks, quits and places",
			pattern: regexp.MustCompile(`(?m)^\tAPP="([^"]+)"\n\tNAME="Dessau Server"`),
		},
		{
			file:    "internal/lifecycle/elevate.go",
			why:     "the application `quit app` asks to quit before the bundle is replaced",
			pattern: regexp.MustCompile("`quit app \"([^\"]+)\"`"),
		},
		{
			file:    "internal/lifecycle/install.go",
			why:     "the bundle `dessau install` looks for and places",
			pattern: regexp.MustCompile(`(?m)bundleName\s*=\s*"([^"]+)\.app"`),
		},
		{
			file:    "internal/lifecycle/updatefetch.go",
			why:     "the release asset `dessau update` downloads; a name that misses is a 404 on every update",
			pattern: regexp.MustCompile(`(?m)updateArchiveName\s*=\s*"([^"]+)\.app\.zip"`),
		},
		{
			file:    "install.sh",
			why:     "the release asset the bootstrap downloads for the server (the first ASSET= line; the client's follows it)",
			pattern: regexp.MustCompile(`(?m)^\tASSET="([^"]+)\.app\.zip"$`),
		},
		{
			file:    "install.sh",
			why:     "the release asset the bootstrap downloads to place the client with",
			pattern: regexp.MustCompile(`(?m)^\tPLACER_ASSET="([^"]+)\.app\.zip"`),
		},
	} {
		m := surface.pattern.FindStringSubmatch(readRepoFile(t, root, surface.file))
		if m == nil {
			t.Errorf("%s no longer names the server bundle where this test looks (%s); "+
				"it names %s, and a rename that misses it fails silently",
				surface.file, surface.pattern, surface.why)
			continue
		}
		if m[1] != want {
			t.Errorf("%s names the server bundle %q while build/Info.plist declares CFBundleName %q — %s, "+
				"and a name that matches nothing is answered without an error",
				surface.file, m[1], want, surface.why)
		}
	}
}
