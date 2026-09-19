package lifecycle

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// `gropius place` is the client's half of the staged swap.
//
// The server bundle has been placed by this package since
// iss-2609081310071028; the chat client was still placed by install.sh's own
// `mv`, which nests into a destination that already exists as a directory and
// follows one that is a symbolic link, exiting 0 in both cases
// (iss-2609111454146700). GropiusChat carries no binary of its own, so the verb
// that places it is a verb on this binary, and the bootstrap calls it.
//
// Every case below runs the verb the way a command line reaches it, because
// what is under test is the pair — the checks the verb makes on its two
// arguments, and the swap underneath them.

// placeArgs is the command line the bootstrap writes.
func placeArgs(bundle, into string) []string {
	return []string{"--bundle", bundle, "--into", into}
}

// The ordinary case: nothing at the destination yet.
func TestPlacePutsTheBundleInTheDestinationDirectory(t *testing.T) {
	dir := t.TempDir()
	src := bundleAt(t, filepath.Join(dir, "extract", "GropiusChat.app"), "new")
	into := filepath.Join(dir, "Applications")
	if err := os.MkdirAll(into, 0o755); err != nil {
		t.Fatal(err)
	}

	env, out, errOut := testEnv()
	if code := RunPlace(env, placeArgs(src, into)); code != ExitOK {
		t.Fatalf("exit = %d, want %d (%s)", code, ExitOK, errOut)
	}
	dest := filepath.Join(into, "GropiusChat.app")
	if got := markerAt(t, dest); got != "new" {
		t.Errorf("the destination holds %q, want the new bundle", got)
	}
	// The line the release gate greps for, and the line a person reads.
	if want := "placed " + dest + "."; !strings.Contains(out.String(), want) {
		t.Errorf("the verb said %q, which does not carry %q", out, want)
	}
	assertNoStagingLeft(t, into)
}

// An installed bundle is REPLACED. `mv` puts the staged bundle inside it and
// exits 0, which leaves every launcher opening the old one.
func TestPlaceReplacesAnInstalledBundleRatherThanNestingInsideIt(t *testing.T) {
	dir := t.TempDir()
	src := bundleAt(t, filepath.Join(dir, "extract", "GropiusChat.app"), "new")
	into := filepath.Join(dir, "Applications")
	dest := bundleAt(t, filepath.Join(into, "GropiusChat.app"), "old")

	env, _, errOut := testEnv()
	if code := RunPlace(env, placeArgs(src, into)); code != ExitOK {
		t.Fatalf("exit = %d, want %d (%s)", code, ExitOK, errOut)
	}
	if got := markerAt(t, dest); got != "new" {
		t.Errorf("the destination holds %q, want the new bundle", got)
	}
	if _, err := os.Lstat(filepath.Join(dest, "GropiusChat.app")); err == nil {
		t.Error("the new bundle was nested inside the installed one, which is what `mv` does")
	}
	assertNoStagingLeft(t, into)
}

// A symbolic link at the destination NAME is replaced rather than written
// through: what it points at is left exactly as it was.
func TestPlaceReplacesASymlinkAtTheDestinationRatherThanFollowingIt(t *testing.T) {
	dir := t.TempDir()
	src := bundleAt(t, filepath.Join(dir, "extract", "GropiusChat.app"), "new")
	into := filepath.Join(dir, "Applications")
	if err := os.MkdirAll(into, 0o755); err != nil {
		t.Fatal(err)
	}
	elsewhere := bundleAt(t, filepath.Join(dir, "elsewhere", "GropiusChat.app"), "planted")
	dest := filepath.Join(into, "GropiusChat.app")
	if err := os.Symlink(elsewhere, dest); err != nil {
		t.Fatal(err)
	}

	env, _, errOut := testEnv()
	if code := RunPlace(env, placeArgs(src, into)); code != ExitOK {
		t.Fatalf("exit = %d, want %d (%s)", code, ExitOK, errOut)
	}
	fi, err := os.Lstat(dest)
	if err != nil {
		t.Fatalf("no bundle at the destination: %v", err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		t.Error("the destination is still a symbolic link, so the bundle was written through it")
	}
	if got := markerAt(t, dest); got != "new" {
		t.Errorf("the destination holds %q, want the new bundle", got)
	}
	if got := markerAt(t, elsewhere); got != "planted" {
		t.Errorf("what the link pointed at now holds %q; the swap followed the link", got)
	}
	assertNoStagingLeft(t, into)
}

// The destination DIRECTORY is a different matter: the swap stages inside it,
// so a symbolic link there means the staging and the bundle both land wherever
// it points. It is refused, and nothing is written.
func TestPlaceRefusesADestinationDirectoryThatIsASymlink(t *testing.T) {
	dir := t.TempDir()
	src := bundleAt(t, filepath.Join(dir, "extract", "GropiusChat.app"), "new")
	real := filepath.Join(dir, "elsewhere")
	if err := os.MkdirAll(real, 0o755); err != nil {
		t.Fatal(err)
	}
	into := filepath.Join(dir, "Applications")
	if err := os.Symlink(real, into); err != nil {
		t.Fatal(err)
	}

	env, out, errOut := testEnv()
	if code := RunPlace(env, placeArgs(src, into)); code != ExitFailed {
		t.Fatalf("exit = %d, want %d for a symlinked destination directory", code, ExitFailed)
	}
	if !strings.Contains(errOut.String(), "symbolic link") {
		t.Errorf("the refusal %q does not say what was wrong with the destination", errOut)
	}
	if entries, err := os.ReadDir(real); err != nil || len(entries) != 0 {
		t.Errorf("the refusal still wrote %v through the link", entries)
	}
	if out.Len() != 0 {
		t.Errorf("a refused placement still reported %q as done", out)
	}
}

// A bundle that is a symbolic link is refused too: the verb copies what it is
// pointed at, and following one would copy something outside the directory the
// bootstrap verified.
func TestPlaceRefusesABundleThatIsASymlink(t *testing.T) {
	dir := t.TempDir()
	real := bundleAt(t, filepath.Join(dir, "elsewhere", "GropiusChat.app"), "planted")
	src := filepath.Join(dir, "extract", "GropiusChat.app")
	if err := os.MkdirAll(filepath.Dir(src), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, src); err != nil {
		t.Fatal(err)
	}
	into := filepath.Join(dir, "Applications")
	if err := os.MkdirAll(into, 0o755); err != nil {
		t.Fatal(err)
	}

	env, _, errOut := testEnv()
	if code := RunPlace(env, placeArgs(src, into)); code != ExitFailed {
		t.Fatalf("exit = %d, want %d for a bundle that is a symbolic link", code, ExitFailed)
	}
	if !strings.Contains(errOut.String(), "symbolic link") {
		t.Errorf("the refusal %q does not say what was wrong with the bundle", errOut)
	}
}

// A command line that does not name both halves is a usage error rather than a
// run: a missing destination is not a reason to guess one.
func TestPlaceRefusesAnIncompleteCommandLine(t *testing.T) {
	dir := t.TempDir()
	src := bundleAt(t, filepath.Join(dir, "extract", "GropiusChat.app"), "new")
	into := filepath.Join(dir, "Applications")
	if err := os.MkdirAll(into, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"nothing at all", nil},
		{"no destination", []string{"--bundle", src}},
		{"no bundle", []string{"--into", into}},
		{"a flag it does not have", append(placeArgs(src, into), "-wibble")},
		{"a stray argument", append(placeArgs(src, into), "everything")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env, out, errOut := testEnv()
			if code := RunPlace(env, tc.args); code != ExitUsage {
				t.Fatalf("exit = %d, want %d", code, ExitUsage)
			}
			if errOut.Len() == 0 {
				t.Error("the refusal said nothing about why")
			}
			if out.Len() != 0 {
				t.Errorf("a refused command line still wrote %q to standard output", out)
			}
			if _, err := os.Lstat(filepath.Join(into, "GropiusChat.app")); err == nil {
				t.Error("a refused command line placed the bundle anyway")
			}
		})
	}
}

// A destination directory that is not there is not created: the bootstrap
// chooses between /Applications and ~/Applications and makes the one it
// chooses, so an absent one here means the caller named the wrong place.
func TestPlaceRefusesADestinationDirectoryThatIsNotThere(t *testing.T) {
	dir := t.TempDir()
	src := bundleAt(t, filepath.Join(dir, "extract", "GropiusChat.app"), "new")
	into := filepath.Join(dir, "Applications")

	env, _, errOut := testEnv()
	if code := RunPlace(env, placeArgs(src, into)); code != ExitFailed {
		t.Fatalf("exit = %d, want %d for a destination that is not there", code, ExitFailed)
	}
	if !strings.Contains(errOut.String(), into) {
		t.Errorf("the refusal %q does not name the destination", errOut)
	}
	if _, err := os.Lstat(into); err == nil {
		t.Error("the refusal created the destination directory")
	}
}
