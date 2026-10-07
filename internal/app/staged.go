package app

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	goruntime "runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/intentdriven/Dessau/internal/config"
	"github.com/intentdriven/Dessau/internal/hub"
	"github.com/intentdriven/Dessau/internal/registry"
	"github.com/intentdriven/Dessau/internal/runtime"
	"golang.org/x/sys/unix"
)

// stagingDirName is the folder under the models directory where a new version
// of a model is fetched beside the one being served (spc-2610030929021692
// step 3). The '+' is a character no repo id can hold, so the rescan, which
// adopts only directories whose names are valid repo ids, never takes it or
// anything in it for a model, and no download can be pointed at it.
const stagingDirName = ".dessau+staging"

// asideSuffix marks the old copy of a model moved aside during a swap. It is
// a name no repo id can hold either.
const asideSuffix = "+old"

// drainWait bounds how long a swap waits for requests in flight on the model
// to finish. Past it the update is abandoned and the old version keeps
// serving: an update is never worth cutting off someone's answer for.
const drainWait = 2 * time.Minute

// ErrUpdateNotOffered is Update's refusal for a model whose newer version is
// one Dessau will not run, or one that waits for review.
var ErrUpdateNotOffered = errors.New("no update is offered for this model")

// ErrNoSpace is a staged download's refusal when the models folder cannot
// hold the new version beside the old one.
var ErrNoSpace = errors.New("not enough free disk space for the new version beside the old one")

// Every path below is relative to the models folder, and every operation on
// one goes through an os.Root opened there (modelsRoot): the staging folder
// sits inside a tree a third party's file names reach, and a link planted at
// any name in it must never carry a removal, a rename or a write out of the
// models folder. Within the folder a link is still a link: the staging
// folder's own components are made real directories before anything is done
// under them (realStagingDirs), and a model's org folder must already be one.
func stagingRel(repoID string) string {
	return filepath.Join(stagingDirName, filepath.FromSlash(repoID))
}

// asideRel names where this attempt moves repoID's old version: one name per
// attempt, so a later update's swap and an earlier one's clean-up never act on
// the same folder.
func asideRel(repoID, attempt string) string { return stagingRel(repoID) + asideSuffix + attempt }
func destRel(repoID string) string           { return filepath.FromSlash(repoID) }

func (a *App) stagingRoot() string { return filepath.Join(a.Paths.Models, stagingDirName) }

// stagingDir is where repoID's new version is fetched.
func (a *App) stagingDir(repoID string) string {
	return filepath.Join(a.Paths.Models, stagingRel(repoID))
}

// modelsRoot opens the models folder as a root every staging operation is
// confined to.
func (a *App) modelsRoot() (*os.Root, error) { return os.OpenRoot(a.Paths.Models) }

// realStagingDirs makes the staging folder and repoID's org folder inside it
// real directories: whatever else stands at either name — a link, a file — is
// removed (the link itself, never what it names) and a directory made in its
// place. The staging folder is Dessau's own, so nothing there is anyone's to
// keep.
func realStagingDirs(root *os.Root, repoID string) error {
	org, _, _ := strings.Cut(repoID, "/")
	for _, rel := range []string{stagingDirName, filepath.Join(stagingDirName, org)} {
		fi, err := root.Lstat(rel)
		switch {
		case err == nil && fi.IsDir():
			continue
		case err == nil:
			if err := root.Remove(rel); err != nil {
				return fmt.Errorf("clear %s from the staging folder: %w", rel, err)
			}
		case !errors.Is(err, fs.ErrNotExist):
			return err
		}
		if err := root.Mkdir(rel, 0o755); err != nil && !errors.Is(err, fs.ErrExist) {
			return err
		}
		if fi, err := root.Lstat(rel); err != nil || !fi.IsDir() {
			return fmt.Errorf("%s is not a directory in the staging folder", rel)
		}
	}
	return nil
}

// refusedError marks a new version that the checks a launch makes refused,
// so a failed update can say so (updateFailureClass) without the error's
// text changing.
type refusedError struct{ err error }

func (e refusedError) Error() string { return e.err.Error() }
func (e refusedError) Unwrap() error { return e.err }

// updateFailureClass is why an update failed, as the class the model's
// record carries and the card has words for.
func updateFailureClass(err error) string {
	var refused refusedError
	switch {
	case errors.Is(err, ErrNoSpace):
		return registry.UpdateFailedNoSpace
	case errors.Is(err, ErrUpdateNotOffered):
		return registry.UpdateFailedNotOffered
	case errors.Is(err, runtime.ErrBusy):
		return registry.UpdateFailedBusy
	case errors.As(err, &refused):
		return registry.UpdateFailedRefused
	default:
		return registry.UpdateFailedDownload
	}
}

// stagingOrg is repoID's org folder in the staging folder, opened as a root
// once it is checked to be a real directory. Every removal under it goes
// through this root, which holds the directory itself rather than its name:
// a link planted at the name afterwards — even one that stays inside the
// models folder, which the models root would follow — is never what a
// removal reaches (iss-2610031324593822).
type stagingOrg struct {
	*os.Root
	rel string
	fi  fs.FileInfo
}

// openStagingOrg opens repoID's org folder in the staging folder, refusing
// unless what it opened is the real directory the name held when it was
// checked. It makes nothing: realStagingDirs does that first, where it is
// wanted.
func openStagingOrg(root *os.Root, repoID string) (*stagingOrg, error) {
	org, _, _ := strings.Cut(repoID, "/")
	return openStagingOrgNamed(root, org)
}

// openStagingOrgNamed is openStagingOrg for an org folder named directly.
func openStagingOrgNamed(root *os.Root, org string) (*stagingOrg, error) {
	rel := filepath.Join(stagingDirName, org)
	fi, err := root.Lstat(rel)
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("%s is not a directory in the staging folder", rel)
	}
	r, err := root.OpenRoot(rel)
	if err != nil {
		return nil, err
	}
	if got, err := r.Stat("."); err != nil || !os.SameFile(fi, got) {
		r.Close()
		return nil, fmt.Errorf("%s changed while it was opened", rel)
	}
	return &stagingOrg{Root: r, rel: rel, fi: fi}, nil
}

// still refuses unless the name the staging org folder was opened at is
// still that folder. The hard links an update makes before its download
// (linkUnchanged) are made by name from the models root, so it is asked
// just before them; that narrows the window rather than closing it, and a
// link misdirected there only adds a name for a file, removing nothing. The
// swap's renames need no such check: they are made relative to the opened
// folders themselves (renameAt).
func (s *stagingOrg) still(root *os.Root) error {
	fi, err := root.Lstat(s.rel)
	if err != nil {
		return err
	}
	if !fi.IsDir() || !os.SameFile(fi, s.fi) {
		return fmt.Errorf("%s was replaced during the update", s.rel)
	}
	return nil
}

// modelOrg is repoID's org folder in the models folder, held open as a
// root and as a directory a rename can be made relative to.
type modelOrg struct {
	*os.Root
	dir *os.File
}

func (m *modelOrg) Close() error {
	m.dir.Close()
	return m.Root.Close()
}

// openModelOrg opens repoID's org folder in the models folder, refusing
// unless it is a real directory and what was opened is that directory: a
// swap renames the model's folder within it.
func openModelOrg(root *os.Root, repoID string) (*modelOrg, error) {
	org, _, _ := strings.Cut(repoID, "/")
	fi, err := root.Lstat(org)
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("the models folder's %s is not a directory, so a new version is not moved into it", org)
	}
	r, err := root.OpenRoot(org)
	if err != nil {
		return nil, err
	}
	dir, err := r.Open(".")
	if err != nil {
		r.Close()
		return nil, err
	}
	if got, err := dir.Stat(); err != nil || !os.SameFile(fi, got) {
		dir.Close()
		r.Close()
		return nil, fmt.Errorf("the models folder's %s changed while it was opened", org)
	}
	return &modelOrg{Root: r, dir: dir}, nil
}

// sameDir reports whether name in the folder r holds is a directory, not a
// link, and the directory want describes.
func sameDir(r *os.Root, name string, want fs.FileInfo) bool {
	got, err := r.Lstat(name)
	return err == nil && got.IsDir() && os.SameFile(got, want)
}

// renameAt renames one entry of an open directory to one of another, by the
// directories themselves rather than by any path to them.
func renameAt(from *os.File, fromName string, to *os.File, toName string) error {
	err := unix.Renameat(int(from.Fd()), fromName, int(to.Fd()), toName)
	goruntime.KeepAlive(from)
	goruntime.KeepAlive(to)
	return err
}

// Update fetches the newer version a check found for a model, beside the one
// being served, and swaps it in only when every file has checked out. A model
// whose version Dessau never recorded, or that no check has marked, is brought
// to the repository's current version. A model whose newer version ships its
// own code, waits for review, or could not be checked for code, is refused:
// there is nothing Dessau knows it would run to update it to.
func (a *App) Update(repoID string) error {
	m, err := a.Registry.Get(repoID)
	if err != nil {
		return fmt.Errorf("%s is not downloaded: %w", repoID, err)
	}
	if !m.Ready() {
		return fmt.Errorf("%s is not ready (%s)", m.RepoID, m.State)
	}
	commit := ""
	if u := m.Update; u != nil && m.VersionKnown() {
		switch u.Status {
		case registry.UpdateAvailable:
			commit = u.Commit
		case registry.UpdateRunsOwnCode, registry.UpdateAwaitingReview, registry.UpdateCannotCheck:
			return fmt.Errorf("%s: %w", m.RepoID, ErrUpdateNotOffered)
		}
	}
	return a.startDownload(m.RepoID, commit)
}

// stagedVersion is what a staged download hands its caller once the new
// version is in place: the version, and the facts about it read from its
// files before it was swapped in, so the record that describes it can be
// written the moment the swap is done rather than after a walk of the
// directory and a request to the Hub.
type stagedVersion struct {
	snap        hub.Snapshot
	bytes       int64
	facts       registry.ModelFacts
	pipelineTag string
	tags        []string
	answered    bool
	// aside is where the swap moved the old version, for the caller to
	// remove once the new record is published.
	aside string
}

// stagedDownload fetches repoID at commit (the current one when empty) into
// the staging folder, checks it, reads its facts, and swaps it in for the
// version being served. On success the model is still marked swapping —
// loads refused — until the caller has published the new record and lifted
// the mark, and the old version waits aside for the caller to remove
// (removeAside). On any failure the staging folder is removed and the
// version being served is untouched.
func (a *App) stagedDownload(ctx context.Context, repoID, commit string, prior registry.Model, onProgress func(hub.Progress)) (stagedVersion, error) {
	root, err := a.modelsRoot()
	if err != nil {
		return stagedVersion{}, err
	}
	defer root.Close()
	if err := realStagingDirs(root, repoID); err != nil {
		return stagedVersion{}, err
	}
	sorg, err := openStagingOrg(root, repoID)
	if err != nil {
		return stagedVersion{}, err
	}
	defer sorg.Close()
	_, name, _ := strings.Cut(repoID, "/")
	if err := sorg.RemoveAll(name); err != nil {
		return stagedVersion{}, fmt.Errorf("clear the staging folder: %w", err)
	}
	fail := func(err error) (stagedVersion, error) {
		if rerr := sorg.RemoveAll(name); rerr != nil {
			a.Log.Warn("could not remove a staged version", "model", repoID, "err", rerr)
		}
		return stagedVersion{}, err
	}

	if commit == "" {
		c, err := a.Hub.Commit(ctx, repoID)
		if err != nil {
			return fail(err)
		}
		commit = c
	}
	// A decision model is fetched only at a version a Dessau release has
	// reviewed, however the commit was arrived at: the registry's mark is a
	// record anyone who can write the index can edit, so it is asked again
	// here rather than trusted.
	if a.reviewed != nil {
		if decision, reviewed := a.reviewed(repoID); decision && !slices.Contains(reviewed, commit) {
			return fail(fmt.Errorf("%s at %s has not been reviewed by a Dessau release: %w", repoID, commit, ErrUpdateNotOffered))
		}
	}
	listing, err := a.Hub.Files(ctx, repoID, commit)
	if err != nil {
		return fail(err)
	}
	files := hub.WantedFiles(listing)
	if err := sorg.still(root); err != nil {
		return fail(err)
	}
	linked := a.linkUnchanged(root, repoID, prior, files)
	var need int64
	for _, f := range files {
		if !linked[f.Path] {
			need += f.Size
		}
	}
	if free, ok := a.freeSpace(a.Paths.Models); ok && free < need {
		return fail(fmt.Errorf("%s needs %d bytes and the models folder has %d free: %w", repoID, need, free, ErrNoSpace))
	}

	staging := a.stagingDir(repoID)
	snap, err := a.Hub.Download(ctx, hub.DownloadRequest{
		RepoID:      repoID,
		Revision:    commit,
		ModelsDir:   a.Paths.Models,
		Dest:        staging,
		Concurrency: 4,
		OnProgress:  onProgress,
	})
	if err != nil {
		return fail(err)
	}
	if err := validateModelDir(staging); err != nil {
		return fail(refusedError{fmt.Errorf("the new version is not a usable MLX model: %w", err)})
	}
	if err := a.prechecked(runtime.Spec{RepoID: repoID, ModelPath: staging}); err != nil {
		return fail(refusedError{fmt.Errorf("the new version would not be started: %w", err)})
	}
	// Read before the swap, so the moment the new files are in place is the
	// moment their record can be written.
	out := stagedVersion{snap: snap, bytes: a.measureDir(staging), facts: registry.ReadModelFacts(staging)}
	out.pipelineTag, out.tags, out.answered = a.repoCategory(ctx, repoID)
	aside, err := a.swapIn(ctx, root, sorg, repoID)
	if err != nil {
		return fail(err)
	}
	out.aside = aside
	return out, nil
}

// linkUnchanged hard-links into the staging folder each file of the version
// being served that the new listing gives the same hash the record holds for
// it, so an update that changes a config does not fetch every weight again.
// The download that follows holds each linked file to its hash like any file
// already on disk, and fetches one that fails. A file that cannot be linked
// is simply fetched. Everything happens inside root, the models folder, so a
// link planted in either tree is not followed out of it.
func (a *App) linkUnchanged(root *os.Root, repoID string, prior registry.Model, files []hub.File) map[string]bool {
	linked := map[string]bool{}
	if prior.FileHashes == nil {
		return linked
	}
	from, to := destRel(repoID), stagingRel(repoID)
	for _, f := range files {
		want := strings.ToLower(f.OID)
		if f.LFS != nil && f.LFS.OID != "" {
			want = strings.ToLower(f.LFS.OID)
		}
		if want == "" || prior.FileHashes[f.Path] != want {
			continue
		}
		rel := filepath.FromSlash(f.Path)
		if !filepath.IsLocal(rel) {
			continue
		}
		src, dst := filepath.Join(from, rel), filepath.Join(to, rel)
		fi, err := root.Lstat(src)
		if err != nil || !fi.Mode().IsRegular() {
			continue
		}
		if err := root.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			continue
		}
		if err := root.Link(src, dst); err == nil {
			linked[f.Path] = true
		}
	}
	return linked
}

// swapIn moves the staged version in for the one being served. Loads of the
// model are refused from the start (the swapping mark, which modelSource
// reads), the model is drained from the pool — requests in flight finish,
// for up to drainWait — and then the old directory is moved aside and the
// staged one moved in and checked once more. A failure at any point puts the
// old directory back and lifts the mark; on success the mark stays for the
// caller to lift with the new record.
func (a *App) swapIn(ctx context.Context, root *os.Root, sorg *stagingOrg, repoID string) (aside string, err error) {
	a.dlMu.Lock()
	a.swapping[dlKey(repoID)] = true
	a.dlMu.Unlock()
	defer func() {
		if err != nil {
			a.endSwap(repoID)
		}
	}()

	// New requests are refused from here until the old version is stopped,
	// or the swap gives up: a model under steady traffic never falls idle
	// otherwise (iss-2610031317470004). Once it is stopped, the swapping
	// mark keeps a load away until the new version is in.
	undrain := a.Pool.Drain(repoID)
	defer undrain()
	deadline := time.Now().Add(a.drainWait)
	for {
		err := a.Pool.Remove(repoID)
		if err == nil || errors.Is(err, runtime.ErrNotLoaded) {
			break
		}
		if !errors.Is(err, runtime.ErrBusy) {
			return "", fmt.Errorf("stop the old version: %w", err)
		}
		if time.Now().After(deadline) {
			return "", fmt.Errorf("the old version was still answering requests after %v, so it keeps serving: %w", a.drainWait, err)
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}

	org, err := openModelOrg(root, repoID)
	if err != nil {
		return "", err
	}
	defer org.Close()
	stagingDir, err := sorg.Open(".")
	if err != nil {
		return "", err
	}
	defer stagingDir.Close()
	// Every rename below names one component in a directory held open: the
	// model's org folder and the staging org folder, each opened once it was
	// checked real, so a link planted at either folder's name afterwards is
	// never followed. A rename moves whatever stands at the one name it
	// moves, a link included, so each is checked after it runs, through the
	// held folders, to have moved the directory it was meant to; one that
	// did not is undone and the update abandoned, before anything is removed.
	_, name, _ := strings.Cut(repoID, "/")
	attempt := strconv.FormatInt(time.Now().UnixNano(), 36)
	asideName := name + asideSuffix + attempt
	aside = asideRel(repoID, attempt)
	old, err := org.Lstat(name)
	if err != nil {
		// The model's folder is not there to move aside. If an earlier swap
		// left the only copy aside, it goes back rather than being cleared
		// to make way: an update never removes the last copy of a model.
		if left, ok := asideLeft(sorg, repoID); ok {
			leftName := filepath.Base(left)
			was, lerr := sorg.Lstat(leftName)
			if lerr != nil || !was.IsDir() {
				return "", errors.New("the model's folder is missing and the copy left aside is not a directory; this update is abandoned")
			}
			if rerr := renameAt(stagingDir, leftName, org.dir, name); rerr != nil {
				return "", fmt.Errorf("the model's folder is missing and the copy left aside could not be put back: %w", rerr)
			}
			if !sameDir(org.Root, name, was) {
				if rerr := renameAt(org.dir, name, stagingDir, leftName); rerr != nil {
					a.Log.Error("could not undo a swap's rename", "model", repoID, "err", rerr)
				}
				return "", errors.New("the copy left aside changed as it was put back; this update is abandoned")
			}
			return "", errors.New("the model's folder was missing; the copy an earlier update left aside is back in place, and this update is abandoned")
		}
		return "", fmt.Errorf("the model's folder is missing: %w", err)
	}
	if !old.IsDir() {
		return "", errors.New("the model's folder is not a directory, so it is not moved aside")
	}
	undo := func(from *os.File, fromName string, to *os.File, toName string) {
		if rerr := renameAt(from, fromName, to, toName); rerr != nil {
			a.Log.Error("could not undo a swap's rename", "model", repoID, "err", rerr)
		}
	}
	if a.beforeSwapRename != nil {
		a.beforeSwapRename("aside")
	}
	if err := renameAt(org.dir, name, stagingDir, asideName); err != nil {
		return "", fmt.Errorf("move the old version aside: %w", err)
	}
	if !sameDir(sorg.Root, asideName, old) {
		undo(stagingDir, asideName, org.dir, name)
		return "", errors.New("the model's folder changed as it was moved aside; the update is abandoned")
	}
	putBack := func() {
		if rerr := renameAt(stagingDir, asideName, org.dir, name); rerr != nil {
			a.Log.Error("could not put the old version back after a failed swap; the next start puts it back, or keeps it aside if something else stands in its place",
				"model", repoID, "err", rerr)
		}
	}
	staged, err := sorg.Lstat(name)
	if err != nil || !staged.IsDir() {
		putBack()
		return "", errors.New("the staged version is not a directory, so it is not moved in")
	}
	if a.beforeSwapRename != nil {
		a.beforeSwapRename("in")
	}
	if err := renameAt(stagingDir, name, org.dir, name); err != nil {
		putBack()
		return "", fmt.Errorf("move the new version in: %w", err)
	}
	// The check below reads the model's folder by its path, so the path must
	// name the folder just moved in, as well as the held folder holding it.
	if fi, err := os.Stat(a.Paths.ModelDir(repoID)); !sameDir(org.Root, name, staged) || err != nil || !os.SameFile(fi, staged) {
		undo(org.dir, name, stagingDir, name)
		putBack()
		return "", errors.New("the model's folder changed as the new version was moved in; the update is abandoned")
	}
	if err := validateModelDir(a.Paths.ModelDir(repoID)); err != nil {
		// Not expected — the staged copy was checked — but a check that
		// fails here puts the old version back rather than serving neither.
		if rerr := renameAt(org.dir, name, stagingDir, name); rerr != nil {
			a.Log.Error("could not move a failed new version out of the way", "model", repoID, "err", rerr)
		} else {
			putBack()
		}
		return "", refusedError{fmt.Errorf("the new version did not check out in place: %w", err)}
	}
	return aside, nil
}

// endSwap lifts the swapping mark.
func (a *App) endSwap(repoID string) {
	a.dlMu.Lock()
	defer a.dlMu.Unlock()
	delete(a.swapping, dlKey(repoID))
}

// asideLeft finds a copy of repoID an earlier swap left aside, if one is,
// the newest when there are several.
// The caller renames it by its base name in the staging org folder.
func asideLeft(sorg *stagingOrg, repoID string) (string, bool) {
	org, name, _ := strings.Cut(repoID, "/")
	entries, err := readDirIn(sorg.Root, ".")
	if err != nil {
		return "", false
	}
	// The newest, when more than one is there: each attempt names its own by
	// the time it started, and the latest is the version served last.
	best, bestAt := "", int64(-1)
	for _, e := range entries {
		attempt, ok := strings.CutPrefix(e.Name(), name+asideSuffix)
		if !ok || !e.IsDir() || e.Type()&fs.ModeSymlink != 0 {
			continue
		}
		if at := asideAttempt(attempt); at > bestAt {
			best, bestAt = e.Name(), at
		}
	}
	if best == "" {
		return "", false
	}
	return filepath.Join(stagingDirName, org, best), true
}

// asideAttempt is when the attempt an aside copy's name ends in started, so
// copies can be ordered newest first; a name that is not one reads as the
// oldest.
func asideAttempt(attempt string) int64 {
	at, err := strconv.ParseInt(attempt, 36, 64)
	if err != nil || at < 0 {
		return 0
	}
	return at
}

// removeAside removes the old version this attempt's swap left aside. Called
// after the mark is lifted, so loads are not refused for as long as the
// removal takes; the folder is this attempt's own, so a later update's swap
// is never what it removes.
func (a *App) removeAside(repoID, aside string) {
	if aside == "" {
		return
	}
	err := func() error {
		root, err := a.modelsRoot()
		if err != nil {
			return err
		}
		defer root.Close()
		sorg, err := openStagingOrg(root, repoID)
		if err != nil {
			return err
		}
		defer sorg.Close()
		return sorg.RemoveAll(filepath.Base(aside))
	}()
	if err != nil {
		a.Log.Warn("could not remove the old version after an update", "model", repoID, "err", err)
	}
}

// pruneUnlisted removes from repoID's folder every file the version just
// fetched into it does not name: what an earlier attempt at another commit
// left there (iss-2610030913179523). A model server loads every weight
// shard it finds, so a leftover shard is not harmless. Only the folder's own
// files are touched, inside a root at the models folder, and a link is
// removed as a link.
func (a *App) pruneUnlisted(repoID string, keep []string) {
	if len(keep) == 0 {
		return
	}
	root, err := a.modelsRoot()
	if err != nil {
		return
	}
	defer root.Close()
	// Compared case-folded: on the case-insensitive volume a Mac uses by
	// default, a file the listing names in another case is that file, and
	// removing it would remove what was just fetched.
	wanted := make(map[string]bool, len(keep))
	for _, p := range keep {
		wanted[strings.ToLower(filepath.ToSlash(filepath.Clean(filepath.FromSlash(p))))] = true
	}
	dir := filepath.ToSlash(destRel(repoID))
	var stale []string
	fs.WalkDir(root.FS(), dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel := strings.TrimPrefix(p, dir+"/")
		if !wanted[strings.ToLower(rel)] {
			stale = append(stale, p)
		}
		return nil
	})
	for _, p := range stale {
		if err := root.Remove(filepath.FromSlash(p)); err != nil {
			a.Log.Warn("could not remove a file the downloaded version does not name", "model", repoID, "err", err)
		}
	}
}

// UpdateProgress is how far the newer version of a ready model has come, in
// percent, while one is being fetched beside it, and whether one is.
func (a *App) UpdateProgress(repoID string) (float64, bool) {
	a.dlMu.Lock()
	dl, ok := a.downloads[dlKey(repoID)]
	a.dlMu.Unlock()
	if !ok || !dl.staged.Load() {
		return 0, false
	}
	return float64(dl.progress.Load()) / 100, true
}

// isSwapping reports whether this model's directory is being swapped for a new
// version right now. Like isDeleting it takes dlMu and calls nothing while
// holding it, so the pool may ask it under p.mu.
func (a *App) isSwapping(repoID string) bool {
	a.dlMu.Lock()
	defer a.dlMu.Unlock()
	return a.swapping[dlKey(repoID)]
}

// recoverStaging runs once at start, before the rescan: an old version left
// aside by a swap the previous process died in the middle of goes back where
// it was when nothing took its place, and every staged version is removed —
// a download is never resumed into the staging folder. One that cannot go
// back is kept (clearStagingKeepingAside). All of it inside a
// root at the models folder; a link at the staging folder's name, or at an
// org's inside it, is removed as a link and nothing it names is touched.
func recoverStaging(models string, log interface {
	Warn(string, ...any)
}) {
	root, err := os.OpenRoot(models)
	if err != nil {
		return
	}
	defer root.Close()
	fi, err := root.Lstat(stagingDirName)
	if errors.Is(err, fs.ErrNotExist) {
		return
	}
	if err != nil || !fi.IsDir() {
		if err := root.RemoveAll(stagingDirName); err != nil {
			log.Warn("could not clear the staging folder", "err", err)
		}
		return
	}
	putBackAside(root, log)
	clearStagingKeepingAside(root, models, log)
}

// clearStagingKeepingAside removes every staged version and everything else
// in the staging folder but an old version left aside that could not be put
// back while nothing the rescan would adopt stands in its model's folder — the
// rescan's own check, so the two never disagree — that may
// be the only copy of the model, and an update never removes the last copy.
// It is kept and named in the log for the person to look at. An aside copy
// beside a model folder that checks out is what a swap that finished but
// died before removing it leaves, and goes like everything else. Each org
// folder is opened and held before anything in it is removed, so a link
// planted at its name meanwhile is never followed.
func clearStagingKeepingAside(root *os.Root, models string, log interface{ Warn(string, ...any) }) {
	orgs, err := readDirIn(root, stagingDirName)
	if err != nil {
		log.Warn("could not read the staging folder", "err", err)
		return
	}
	for _, org := range orgs {
		orgRel := filepath.Join(stagingDirName, org.Name())
		sorg, err := openStagingOrgNamed(root, org.Name())
		if err != nil {
			// Not a real directory, or not the one its name held: the name
			// itself goes, as a link or a file, and nothing it names.
			if fi, lerr := root.Lstat(orgRel); lerr == nil && !fi.IsDir() {
				if err := root.Remove(orgRel); err != nil {
					log.Warn("could not clear the staging folder", "err", err)
				}
			}
			continue
		}
		entries, err := readDirIn(sorg.Root, ".")
		if err != nil {
			sorg.Close()
			continue
		}
		for _, e := range entries {
			if name, _, isAside := strings.Cut(e.Name(), asideSuffix); isAside && e.IsDir() && e.Type()&fs.ModeSymlink == 0 {
				repoID := org.Name() + "/" + name
				// One whose name gives no model can never go back, and goes.
				if config.ValidRepoID(repoID) && !registry.ModelDirComplete(filepath.Join(models, filepath.FromSlash(repoID))) {
					log.Warn("an old version an interrupted update left aside could not be put back, because something else stands in its model's folder; it is kept",
						"path", filepath.ToSlash(filepath.Join(orgRel, e.Name())))
					continue
				}
			}
			if err := sorg.RemoveAll(e.Name()); err != nil {
				log.Warn("could not clear the staging folder", "err", err)
			}
		}
		sorg.Close()
		_ = root.Remove(orgRel) // only when nothing was kept in it
	}
	_ = root.Remove(stagingDirName)
}

// putBackAside moves each old version an interrupted swap left aside back to
// its model's folder, when nothing stands there.
func putBackAside(root *os.Root, log interface{ Warn(string, ...any) }) {
	orgs, err := readDirIn(root, stagingDirName)
	if err != nil {
		return
	}
	for _, org := range orgs {
		orgRel := filepath.Join(stagingDirName, org.Name())
		if fi, err := root.Lstat(orgRel); err != nil || !fi.IsDir() {
			continue
		}
		entries, err := readDirIn(root, orgRel)
		if err != nil {
			continue
		}
		// Newest first: the first copy of a model put back is the one served
		// last, and the rest then find its folder taken.
		slices.SortStableFunc(entries, func(x, y os.DirEntry) int {
			_, ax, _ := strings.Cut(x.Name(), asideSuffix)
			_, ay, _ := strings.Cut(y.Name(), asideSuffix)
			return cmp.Compare(asideAttempt(ay), asideAttempt(ax))
		})
		for _, e := range entries {
			name, _, ok := strings.Cut(e.Name(), asideSuffix)
			repoID := org.Name() + "/" + name
			if !ok || !config.ValidRepoID(repoID) {
				continue
			}
			aside := filepath.Join(orgRel, e.Name())
			if fi, err := root.Lstat(aside); err != nil || !fi.IsDir() {
				continue
			}
			if _, err := root.Lstat(destRel(repoID)); !errors.Is(err, fs.ErrNotExist) {
				continue
			}
			if fi, err := root.Lstat(org.Name()); err == nil && !fi.IsDir() {
				continue
			}
			if err := root.MkdirAll(org.Name(), 0o755); err != nil {
				continue
			}
			if err := root.Rename(aside, destRel(repoID)); err != nil {
				log.Warn("could not put back a model left aside by an interrupted update", "model", repoID, "err", err)
			}
		}
	}
}

// readDirIn lists a directory inside root.
func readDirIn(root *os.Root, rel string) ([]os.DirEntry, error) {
	f, err := root.Open(rel)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return f.ReadDir(-1)
}

// prechecked holds a staged version to the launcher's Precheck, or — when the
// runtime is not installed yet, and only then — to the model half of it,
// which needs no runtime (iss-2610031317475284, at the maintainer's answer).
// An interpreter that is there but not trusted is still a refusal, and so is
// a launcher that cannot check a model on its own.
func (a *App) prechecked(spec runtime.Spec) error {
	err := a.launcher.Precheck(spec)
	if !errors.Is(err, runtime.ErrRuntimeNotInstalled) {
		return err
	}
	if mc, ok := a.launcher.(runtime.ModelPrechecker); ok {
		return mc.PrecheckModel(spec)
	}
	return err
}
