package app

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/intentdriven/Dessau/internal/config"
	"github.com/intentdriven/Dessau/internal/hub"
	"github.com/intentdriven/Dessau/internal/registry"
	"github.com/intentdriven/Dessau/internal/runtime"
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

func (a *App) stagingRoot() string { return filepath.Join(a.Paths.Models, stagingDirName) }

// stagingDir is where repoID's new version is fetched.
func (a *App) stagingDir(repoID string) string {
	return config.ModelDirIn(a.stagingRoot(), repoID)
}

// asideDir is where repoID's old version waits while the new one moves in.
func (a *App) asideDir(repoID string) string {
	return a.stagingDir(repoID) + asideSuffix
}

// Update fetches the newer version a check found for a model, beside the one
// being served, and swaps it in only when every file has checked out. A model
// whose version Dessau never recorded, or that no check has marked, is brought
// to the repository's current version. A model whose newer version ships its
// own code, or waits for review, is refused: there is nothing Dessau would
// run to update it to.
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
		case registry.UpdateRunsOwnCode, registry.UpdateAwaitingReview:
			return fmt.Errorf("%s: %w", m.RepoID, ErrUpdateNotOffered)
		}
	}
	return a.startDownload(m.RepoID, commit)
}

// stagedDownload fetches repoID at commit (the current one when empty) into
// the staging folder, checks it, and swaps it in for the version being
// served. It returns the snapshot of what is now on disk. On any failure the
// staging folder is removed and the version being served is untouched.
func (a *App) stagedDownload(ctx context.Context, repoID, commit string, prior registry.Model, onProgress func(hub.Progress)) (hub.Snapshot, error) {
	dest := a.Paths.ModelDir(repoID)
	staging := a.stagingDir(repoID)
	if err := os.RemoveAll(staging); err != nil {
		return hub.Snapshot{}, fmt.Errorf("clear the staging folder: %w", err)
	}
	fail := func(err error) (hub.Snapshot, error) {
		if rerr := os.RemoveAll(staging); rerr != nil {
			a.Log.Warn("could not remove a staged version", "model", repoID, "err", rerr)
		}
		return hub.Snapshot{}, err
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
	linked := a.linkUnchanged(repoID, prior, files)
	var need int64
	for _, f := range files {
		if !linked[f.Path] {
			need += f.Size
		}
	}
	if free, ok := a.freeSpace(a.Paths.Models); ok && free < need {
		return fail(fmt.Errorf("%s needs %d bytes and the models folder has %d free: %w", repoID, need, free, ErrNoSpace))
	}

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
		return fail(fmt.Errorf("the new version is not a usable MLX model: %w", err))
	}
	if err := a.launcher.Precheck(runtime.Spec{RepoID: repoID, ModelPath: staging}); err != nil {
		return fail(fmt.Errorf("the new version would not be started: %w", err))
	}
	if err := a.swapIn(ctx, repoID, staging, dest); err != nil {
		return fail(err)
	}
	return snap, nil
}

// linkUnchanged hard-links into the staging folder each file of the version
// being served that the new listing gives the same hash the record holds for
// it, so an update that changes a config does not fetch every weight again.
// The download that follows holds each linked file to its hash like any file
// already on disk, and fetches one that fails. A file that cannot be linked
// is simply fetched. Everything happens inside a root opened at the models
// folder, so a link planted in either tree is not followed out of it.
func (a *App) linkUnchanged(repoID string, prior registry.Model, files []hub.File) map[string]bool {
	linked := map[string]bool{}
	if prior.FileHashes == nil {
		return linked
	}
	root, err := os.OpenRoot(a.Paths.Models)
	if err != nil {
		return linked
	}
	defer root.Close()
	from, err := filepath.Rel(a.Paths.Models, a.Paths.ModelDir(repoID))
	if err != nil {
		return linked
	}
	to, err := filepath.Rel(a.Paths.Models, a.stagingDir(repoID))
	if err != nil {
		return linked
	}
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

// freeBytes is how much space the volume holding dir has for this account.
func freeBytes(dir string) (int64, bool) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(dir, &st); err != nil {
		return 0, false
	}
	return int64(st.Bavail) * int64(st.Bsize), true
}

// swapIn moves the staged version in for the one being served. Loads of the
// model are refused from the start (the swapping mark, which modelSource
// reads), the model is drained from the pool — requests in flight finish,
// for up to drainWait — and then the old directory is moved aside, the
// staged one moved in, and the new one checked once more before the old one
// is removed. A failure at any point puts the old directory back.
func (a *App) swapIn(ctx context.Context, repoID, staging, dest string) error {
	a.dlMu.Lock()
	a.swapping[dlKey(repoID)] = true
	a.dlMu.Unlock()
	defer func() {
		a.dlMu.Lock()
		delete(a.swapping, dlKey(repoID))
		a.dlMu.Unlock()
	}()

	deadline := time.Now().Add(a.drainWait)
	for {
		err := a.Pool.Remove(repoID)
		if err == nil || errors.Is(err, runtime.ErrNotLoaded) {
			break
		}
		if !errors.Is(err, runtime.ErrBusy) {
			return fmt.Errorf("stop the old version: %w", err)
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("the old version was still answering requests after %v, so it keeps serving: %w", a.drainWait, err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}

	aside := a.asideDir(repoID)
	if err := os.RemoveAll(aside); err != nil {
		return fmt.Errorf("clear the folder the old version moves to: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(aside), 0o755); err != nil {
		return err
	}
	if err := os.Rename(dest, aside); err != nil {
		return fmt.Errorf("move the old version aside: %w", err)
	}
	if err := os.Rename(staging, dest); err != nil {
		if rerr := os.Rename(aside, dest); rerr != nil {
			a.Log.Error("could not put the old version back after a failed swap", "model", repoID, "err", rerr)
		}
		return fmt.Errorf("move the new version in: %w", err)
	}
	if err := validateModelDir(dest); err != nil {
		// Not expected — the staged copy was checked — but a check that
		// fails here puts the old version back rather than serving neither.
		os.Rename(dest, staging)
		if rerr := os.Rename(aside, dest); rerr != nil {
			a.Log.Error("could not put the old version back after a failed swap", "model", repoID, "err", rerr)
		}
		return fmt.Errorf("the new version did not check out in place: %w", err)
	}
	if err := os.RemoveAll(aside); err != nil {
		a.Log.Warn("could not remove the old version after an update", "model", repoID, "err", err)
	}
	return nil
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
// a download is never resumed into the staging folder.
func recoverStaging(models string, log interface {
	Warn(string, ...any)
}) {
	root := filepath.Join(models, stagingDirName)
	orgs, err := os.ReadDir(root)
	if errors.Is(err, fs.ErrNotExist) {
		return
	}
	for _, org := range orgs {
		if !org.IsDir() {
			continue
		}
		entries, err := os.ReadDir(filepath.Join(root, org.Name()))
		if err != nil {
			continue
		}
		for _, e := range entries {
			name, ok := strings.CutSuffix(e.Name(), asideSuffix)
			if !ok || !e.IsDir() || !config.ValidRepoID(org.Name()+"/"+name) {
				continue
			}
			dest := config.ModelDirIn(models, org.Name()+"/"+name)
			if _, err := os.Lstat(dest); !errors.Is(err, fs.ErrNotExist) {
				continue
			}
			if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
				continue
			}
			if err := os.Rename(filepath.Join(root, org.Name(), e.Name()), dest); err != nil {
				log.Warn("could not put back a model left aside by an interrupted update", "model", org.Name()+"/"+name, "err", err)
			}
		}
	}
	if err := os.RemoveAll(root); err != nil {
		log.Warn("could not clear the staging folder", "err", err)
	}
}
