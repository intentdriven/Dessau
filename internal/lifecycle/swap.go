package lifecycle

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"syscall"

	"github.com/intentdriven/Gropius/internal/config"
)

// The staged swap, in Go, because the shell cannot express it.
//
// The shipped script removes the installed bundle and only then renames the
// staged one into its place. A destination that exists as a directory ends up
// with the staged bundle nested inside it; a destination that is a symlink is
// written through and left standing; and both exit 0, so an upgrade reports
// success while every launcher goes on opening the old binary
// (iss-2609081310071028). `mv` has no dependable "fail if the destination
// exists" mode, and any test-then-move in shell is a race by construction.
//
// os.Rename is rename(2): it replaces a symlink rather than following it, and
// it refuses a destination that is a non-empty directory. The order below is
// what makes the failure safe — the installed bundle is renamed ASIDE, the new
// one is renamed in, and the set-aside copy is removed only once the new one is
// in place.
//
// WHAT THIS GUARANTEES, AND AGAINST WHOM.
//
// Against everything that is not another account on this Mac — a failing
// rename, a full disk, a permission revoked between two calls, a process
// killed part way through — no path leaves the Mac with no application. That
// sentence rests on the last rule below: where the new bundle did not go in
// AND the set-aside one could not be put back, the directory holding the
// set-aside copy is NOT cleaned up, and the failure says where the only
// remaining copy is. An earlier version removed it and destroyed the
// installation on a path it claimed to protect — rename(2) refuses a non-empty
// directory, so anything that creates one at the destination between the two
// renames is enough to reach it.
//
// The wait that rule opens is unbounded: it ends when a person moves the copy
// back. So WHERE the copy waits is what decides who can take it away, and a
// directory entry is removed by write permission on the PARENT, not by the
// mode of the entry itself. /Applications is drwxrwxr-x root:admin with no
// sticky bit, so while the copy waited there, any other admin account on the
// Mac could delete it and leave no application at all — 0700 on the staging
// directory governs what is inside it, never who may unlink it
// (iss-2609111755330533). The set-aside copy therefore waits in THIS ACCOUNT'S
// OWN directory (config.AccountHome, under ~/Library, which macOS creates
// 0700), and the new bundle alone is staged in the destination directory.
//
// Putting the copy back has to be a rename, and rename(2) does not cross
// filesystems. Where this account's own directory and the destination are on
// different volumes — or where that directory cannot be used, because it is
// not a directory or something else can write to it — the copy is staged in
// the destination directory as before and a warning says so. That is a loud
// degrade to the earlier exposure, never a silent one.
//
// THE ONE RESIDUAL. An admin account on this Mac that has been compromised is
// not held off by any of this: it can write /Applications directly, so it
// needs no window and no swap. What the account's own home removes is the
// window this file used to open for it. It does not remove the actor, and root
// is outside all of it.

// stagingPrefix names a staging directory. It is a dot name so it does not
// appear in a Finder listing of the destination while the swap runs.
const stagingPrefix = ".gropius-incoming-"

// retiredPrefix names the directory the set-aside bundle waits in, inside this
// account's own directory. A name of its own rather than stagingPrefix: the
// two directories hold opposite things — one the copy that can be thrown away,
// one the copy that must not be — and a person looking at either wants to know
// which they have.
const retiredPrefix = ".gropius-retired-"

// renameFunc is os.Rename, handed in so a test can fail one call of it. The
// failure that produced the defect is an ordinary one — a full disk, a locked
// file, a permission revoked between two calls — and it cannot be provoked on a
// real filesystem on demand.
type renameFunc func(oldpath, newpath string) error

// stagingDir creates a staging directory inside dir, under an unguessable name.
//
// os.MkdirTemp draws the suffix from the same source as any other random name
// in this repository and creates the directory 0700. The name matters: the
// shipped script used the process id before this, and on a Mac where the
// destination is group-writable by the admin group — /Applications is, on stock
// macOS — a predictable name hands another account a reliable signal for when
// to act on the directory.
func stagingDir(dir string) (string, error) { return os.MkdirTemp(dir, stagingPrefix) }

// setAsideDir is the directory the retired bundle is set aside in: one of this
// account's own, reported as own=true, or the staging directory in the
// destination as a fallback.
//
// The fallback is announced every time it is taken, because it is the earlier
// exposure — a copy waiting where a co-resident admin account can unlink it —
// and a degrade nobody is told about is one nobody acts on.
func setAsideDir(destDir, staging string) (dir string, own bool) {
	home, err := accountSwapHome(destDir)
	if err == nil {
		if dir, err = os.MkdirTemp(home, retiredPrefix); err == nil {
			return dir, true
		}
	}
	// Redacted the way every other line an operator may paste in public is:
	// which directory could not be used is the point, whose account it belongs
	// to is not (see redact).
	account, _ := os.UserHomeDir()
	slog.Warn("the retired copy of the application is being staged in the destination directory rather than in "+
		"this account's own, so another administrator account on this Mac could remove it if it has to wait there",
		"destination", redact(destDir, account), "reason", redact(err.Error(), account))
	return staging, false
}

// accountSwapHome is this account's own directory, checked for the one
// property the set-aside copy is put there for and for the one the restore
// needs.
//
// The property: an entry inside it can be unlinked only by this account,
// because ~/Library is 0700 and nothing but this account (or root, which is
// outside all of this) can write a component of the path. The check is of the
// leaf, and it is the same check config.EnsureDirs makes for the same
// directory — a home that something else can write to has lost the property,
// and is refused rather than used as though it still had it.
//
// The restore needs the directory to be on the destination's filesystem:
// putting the copy back is rename(2), which does not cross one. Comparing the
// device number answers that before the copy is moved anywhere, so a wrong
// answer costs a warning rather than an application.
func accountSwapHome(destDir string) (string, error) {
	home, err := config.AccountHome()
	if err != nil {
		return "", err
	}
	// Lstat FIRST, and Lstat rather than Stat: a symbolic link standing where
	// this directory should be is not this directory, whoever planted it — and
	// creating the directory before looking would create it through the link,
	// somewhere nobody asked for, before the refusal.
	//
	// Ordinarily it is already there: a swap that retires a bundle is a swap
	// over an installation, and an installation has this directory. Created
	// closed where it is not, which is the mode config.EnsureDirs holds the
	// same directory to.
	fi, err := os.Lstat(home)
	if errors.Is(err, fs.ErrNotExist) {
		if err := os.MkdirAll(home, 0o700); err != nil {
			return "", err
		}
		fi, err = os.Lstat(home)
	}
	if err != nil {
		return "", err
	}
	if !fi.IsDir() {
		return "", fmt.Errorf("%s is not a directory", home)
	}
	if perm := fi.Mode().Perm(); perm&0o022 != 0 {
		return "", fmt.Errorf("%s is mode %04o, so it is not this account's alone", home, perm)
	}
	// Owned by this account, and not only closed to everyone else. The mode
	// bits are not the whole story on macOS: a directory another account owns
	// can read 0700 in a listing and still grant this one write through an
	// ACL, and a copy waiting in a directory its owner can unlink is the
	// exposure this whole home was chosen to remove.
	if st, ok := fi.Sys().(*syscall.Stat_t); !ok || st.Uid != uint32(os.Getuid()) {
		return "", fmt.Errorf("%s is not owned by this account", home)
	}
	if !sameVolume(home, destDir) {
		return "", fmt.Errorf("%s and %s are on different filesystems, and putting the bundle back is a rename",
			home, destDir)
	}
	return home, nil
}

// sameVolume reports whether two existing directories are on one filesystem.
//
// It is a variable so a test can answer it without a second volume: the
// fallback it guards is a path that a machine with one volume — every Mac this
// ships to, ordinarily — can otherwise never reach.
var sameVolume = func(a, b string) bool {
	fa, err := os.Stat(a)
	if err != nil {
		return false
	}
	fb, err := os.Stat(b)
	if err != nil {
		return false
	}
	sa, ok := fa.Sys().(*syscall.Stat_t)
	sb, ok2 := fb.Sys().(*syscall.Stat_t)
	if !ok || !ok2 {
		return false
	}
	return sa.Dev == sb.Dev
}

// PlaceBundle installs the bundle at src as dest, replacing whatever is there.
func PlaceBundle(src, dest string) error { return placeBundle(src, dest, os.Rename) }

// placeBundle is PlaceBundle with the rename handed in.
func placeBundle(src, dest string, rename renameFunc) error {
	destDir := filepath.Dir(dest)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("prepare %s: %w", destDir, err)
	}

	// Stage the NEW bundle inside the destination directory, so the rename
	// that puts it in place is within one filesystem and cannot fail part way
	// through. A copy is what can run out of space, and it runs first. This
	// directory is the one that can be thrown away: every failure below either
	// puts the installed bundle back or says where it is, and neither answer
	// is in here.
	staging, err := stagingDir(destDir)
	if err != nil {
		return fmt.Errorf("stage the new bundle in %s: %w", destDir, err)
	}
	// Where the set-aside copy waits, decided when there is one to set aside
	// and not before: a swap with nothing at the destination retires nothing,
	// and must not warn about where it would have put it.
	aside, own := "", false
	// keep is cleared only where the set-aside copy may safely go: while it is
	// the only copy of the application, the directory holding it stays. The
	// staging directory in the destination never holds the only copy of
	// anything once the two homes are separate, so it goes on every path —
	// except under the fallback, where it IS the set-aside directory.
	keep := false
	defer func() {
		if own || !keep {
			os.RemoveAll(staging)
		}
		if own && !keep {
			os.RemoveAll(aside)
		}
	}()

	staged := filepath.Join(staging, filepath.Base(dest))
	if err := copyTree(src, staged); err != nil {
		return fmt.Errorf("copy the verified bundle into %s: %w", destDir, err)
	}

	// Set the installed bundle aside rather than deleting it. Lstat, not Stat:
	// a destination that is a symlink to a directory that no longer exists is
	// still a name that has to be got out of the way, and Stat would report it
	// absent and leave the rename below to fail.
	retired := ""
	if _, err := os.Lstat(dest); err == nil {
		aside, own = setAsideDir(destDir, staging)
		retired = filepath.Join(aside, filepath.Base(dest)+".retired")
		if err := rename(dest, retired); err != nil {
			return fmt.Errorf("set the installed bundle aside: %w (it is untouched)", err)
		}
	}

	// Nothing may be at the destination now. Something that is, is a name
	// another process created in the window since the rename above, and it is
	// not this installer's to replace — with the new bundle OR with the old
	// one.
	//
	// So the set-aside copy is NOT put back here, which is the difference
	// between refusing a clobber and performing it with a different bundle: a
	// restore is a rename onto that same name, and whether it lands is the
	// filesystem's business rather than this file's (macOS answers EEXIST for
	// an empty directory and ENOTDIR for a symbolic link; POSIX permits
	// replacing an empty directory, and Linux does). The installed bundle stays
	// where it was set aside, the directory holding it is kept, and the failure
	// names both it and what appeared, because a person now has two things to
	// look at and one of them is their application.
	if fi, err := os.Lstat(dest); err == nil {
		keep = retired != ""
		return keptWhen(keep, retired, fmt.Errorf(
			"%s was created by something else while the new bundle was being staged (%s); "+
				"refusing to replace it%s", dest, describe(fi), retiredNote(retired, dest)))
	}

	if err := rename(staged, dest); err != nil {
		put, note := restore(rename, retired, dest)
		keep = !put && retired != ""
		return keptWhen(keep, retired, fmt.Errorf("move the new bundle into place: %w%s", err, note))
	}

	// And only now is the set-aside copy removed, by the deferred RemoveAll of
	// the directory it sits in — in this account's own directory, so a swap
	// that succeeds leaves nothing of itself anywhere.
	return nil
}

// describe says what kind of thing took a name, for a message a person has to
// act on: a symbolic link, a directory and a file each mean something different
// about who put it there.
func describe(fi os.FileInfo) string {
	switch {
	case fi.Mode()&os.ModeSymlink != 0:
		return "a symbolic link"
	case fi.IsDir():
		return "a directory"
	default:
		return "a file"
	}
}

// retiredNote says where the installed bundle is when it has been set aside and
// is not going back on its own.
func retiredNote(retired, dest string) string {
	if retired == "" {
		return " (nothing was installed there before, and nothing was removed)"
	}
	return ". The installed bundle was already set aside and is intact at " + retired +
		"; move it back to " + dest + " once you have dealt with what is there."
}

// restore puts a set-aside bundle back, so a failure is a no-op rather than an
// uninstall. It reports whether the bundle is back, and the sentence the caller
// should append to the failure it is already returning.
//
// A restore that fails is the one case where a person has to do something: the
// application is not where it belongs, and the only copy of it is sitting under
// a name they would never think to look for. So the failure names that path,
// and the caller keeps the directory it is in.
func restore(rename renameFunc, retired, dest string) (put bool, note string) {
	if retired == "" {
		return true, " (nothing was installed there before, and nothing was removed)"
	}
	if err := rename(retired, dest); err != nil {
		return false, " — and the installed bundle could NOT be put back (" + err.Error() + "). " +
			"It is intact at " + retired + "; move it to " + dest + " to restore the application."
	}
	return true, " (the previous copy is left as it was)"
}

// copyTree copies a directory recursively: regular files, directories and
// symbolic links, with their permission bits.
//
// In-process rather than through /usr/bin/ditto, because a copy that is one
// function is a copy whose failure modes are visible in this file. What it does
// not carry is extended attributes, which is correct here: the bootstrap clears
// the quarantine attribute on the verified bundle before handing over, and an
// ad-hoc signature lives inside the Mach-O and in _CodeSignature, both ordinary
// files.
func copyTree(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	switch {
	case info.Mode()&os.ModeSymlink != 0:
		target, err := os.Readlink(src)
		if err != nil {
			return err
		}
		return os.Symlink(target, dst)
	case info.IsDir():
		if err := os.Mkdir(dst, info.Mode().Perm()); err != nil {
			return err
		}
		entries, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		for _, e := range entries {
			if err := copyTree(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())); err != nil {
				return err
			}
		}
		return nil
	case info.Mode().IsRegular():
		return copyFile(src, dst, info.Mode().Perm())
	default:
		// A device node, a socket or a fifo inside an application bundle is
		// not something to reproduce silently.
		return fmt.Errorf("%s is not a file, a directory or a symbolic link", src)
	}
}

func copyFile(src, dst string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// keptStagingError is the ONE swap failure a person has to act on: the
// application is not where it belongs, and the only copy of it is sitting under
// a name nobody would think to look for.
//
// It carries that path as a VALUE rather than only inside its sentence, and
// that is the whole reason it exists. The first version of the update report
// searched the failure text for the staging name and printed from there on —
// which drops the directory the staging name sits in, so the report named a
// relative path nobody could go to, and fired on the failures where the placer
// had already REMOVED the staging directory. A caller that has to scrape a path
// out of prose gets it wrong; this is the placer stating it.
type keptStagingError struct {
	// Path is the set-aside bundle: the only remaining copy of the application.
	Path string
	err  error
}

func (e *keptStagingError) Error() string { return e.err.Error() }
func (e *keptStagingError) Unwrap() error { return e.err }

// keptWhen wraps a failure as one that kept the only copy, and leaves every
// other failure exactly as it was: the failures that keep nothing must not say
// they kept something.
func keptWhen(keep bool, retired string, err error) error {
	if !keep || retired == "" {
		return err
	}
	return &keptStagingError{Path: retired, err: err}
}
