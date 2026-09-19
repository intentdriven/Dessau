package lifecycle

import (
	"fmt"
	"os"
	"path/filepath"
)

// `gropius place` puts a staged application bundle where it belongs, with the
// staged swap in swap.go — and it exists because the chat client has no binary
// of its own to do it.
//
// The bootstrap's server half hands over to `gropius install`, which places the
// server bundle in Go. Its client half had no such handover, so GropiusChat was
// placed by the shell's `mv`: it nests into a destination that already exists
// as a directory, follows one that is a symbolic link, and exits 0 in both
// cases (iss-2609081310071028, and iss-2609111454146700 for the half that still
// had it). /Applications is group-writable by the admin group on a stock Mac,
// so on a Mac several people share that window is another account's to win.
//
// A verb rather than a flag on install: `gropius install` is this installation's
// own lifecycle — it places THIS product's bundle at the destination the fixed
// rule chooses, and then grants a firewall entry, provisions a runtime and
// links a command for it. Placing a bundle somebody else names, somewhere
// somebody else names, is a different act with a different contract, and
// hanging it off install's flags would mean install's own destination rule and
// the caller's argument disagreeing about what "the bundle" means.
//
// It is deliberately narrow: it takes the two paths, it checks them, it swaps,
// and it says where the bundle went. It reads no settings, contacts nothing,
// and knows nothing about GropiusChat beyond the name the caller hands it.

// RunPlace is the place verb.
func RunPlace(env Env, args []string) int { return runPlace(env, args, PlaceBundle) }

// runPlace is RunPlace with the swap handed in.
func runPlace(env Env, args []string, place func(src, dest string) error) int {
	fs := flags("place", env.Err)
	bundle := fs.String("bundle", "", "the verified application bundle to place")
	into := fs.String("into", "", "the applications directory it belongs in")
	if err := fs.Parse(args); err != nil {
		return ExitUsage
	}
	if fs.NArg() > 0 {
		writeLine(env.Err, "gropius place: unexpected argument "+Quote(fs.Arg(0)))
		return ExitUsage
	}
	if *bundle == "" || *into == "" {
		writeLine(env.Err, "gropius place: --bundle and --into are both required: "+
			"gropius place --bundle <staged .app> --into <applications directory>")
		return ExitUsage
	}

	dest, err := placeDestination(*bundle, *into)
	if err != nil {
		writeLine(env.Err, "gropius place: "+err.Error())
		return ExitFailed
	}
	if err := place(*bundle, dest); err != nil {
		writeLine(env.Err, "gropius place: "+err.Error())
		return ExitFailed
	}
	// The same sentence the bootstrap used to print, because it is what proves
	// a run reached the end and the release gate reads it as that.
	writeLine(env.Out, "placed "+dest+".")
	return ExitOK
}

// placeDestination checks the two paths the caller named and returns where the
// bundle goes.
//
// Lstat throughout, and that is the whole of what these checks are for. The
// bundle must be the directory the caller verified rather than a link to
// something else, and the destination directory must be a directory rather
// than a link: the swap stages INSIDE it, so a symbolic link there sends both
// the staging copy and the bundle wherever it points, with the destination
// left standing and every launcher still opening what is behind the link.
//
// The destination is not created. The bootstrap chooses between the
// machine-wide applications directory and this account's own and makes the one
// it chose; a directory that is not there when this verb runs means the caller
// named somewhere else, and creating it would place the application where
// nothing looks for it.
func placeDestination(bundle, into string) (string, error) {
	src, err := os.Lstat(bundle)
	if err != nil {
		return "", fmt.Errorf("there is no bundle at %s", bundle)
	}
	if src.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("%s is a symbolic link rather than an application bundle", bundle)
	}
	if !src.IsDir() {
		return "", fmt.Errorf("%s is not an application bundle", bundle)
	}

	dir, err := os.Lstat(into)
	if err != nil {
		return "", fmt.Errorf("there is no directory at %s to place %s in",
			into, filepath.Base(bundle))
	}
	if dir.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("%s is a symbolic link rather than a directory; "+
			"the swap stages inside it, so placing through one would write wherever it points", into)
	}
	if !dir.IsDir() {
		return "", fmt.Errorf("%s is not a directory", into)
	}
	return filepath.Join(into, filepath.Base(bundle)), nil
}
