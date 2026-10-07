package app

import (
	"context"
	"errors"
	"math/rand/v2"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/intentdriven/Dessau/internal/hub"
	"github.com/intentdriven/Dessau/internal/registry"
)

// The update check's pacing (spc-2610030929021692). One repository at a time
// with a pause between, as the category job paces itself; each request
// bounded, as the category request is; and a round that stops early once the
// Hub says few requests are left in its window, so a Mac with many models
// never spends the operator's budget on checks.
const (
	updateCheckPause          = time.Second
	updateCheckRequestTimeout = 15 * time.Second
	updateCheckLowBudget      = 10
	// updateCheckTick is how often the schedule compares the clock with the
	// recorded check times. It is a comparison of wall-clock times, not a
	// timer for the interval, so a Mac that slept through a due check runs it
	// at the first tick after it wakes.
	updateCheckTick = time.Minute
	// maxCheckedConfig bounds the one file a check reads, a model's
	// config.json, as the registry bounds reading one from disk.
	maxCheckedConfig = 1 << 20
)

// errChecksOff ends a check the operator turned checks off in the middle of.
var errChecksOff = errors.New("update checks were turned off")

// ReviewedBuilds says whether a model is a decision model, and which upstream
// commits of it a Dessau release has reviewed. A decision model is offered an
// update only to a reviewed commit (itd-2610030656210408), and a newer one
// that is not reviewed is never marked (iss-2610042101436891). Nil means no
// model is a decision model.
type ReviewedBuilds func(repoID string) (decision bool, reviewed []string)

// UpdateRound is what one round of update checks did.
type UpdateRound struct {
	// Checked is how many models the Hub answered for; Marked how many of
	// those now carry a mark; Unreachable how many it did not answer for.
	Checked, Marked, Unreachable int
	// StoppedEarly says the Hub's remaining request budget ran low, or it
	// refused for rate, and the rest of the round was left for the next.
	StoppedEarly bool
}

// updateCheckDue lists the models a check would ask about now: ready, with a
// recorded version, and neither checked nor attempted by this process within
// the interval. Each model is due on its own clock, so one downloaded between
// two rounds is checked an interval after its download, not at the round
// after that. A model whose version Dessau never recorded is never checked —
// there is nothing to compare — and a download records a check, since
// resolving the commit it fetched answered the same question. The attempt
// time is what keeps a Mac that is offline from asking every minute: a check
// the Hub did not answer records nothing on the model, which stays due.
func (a *App) updateCheckDue(now time.Time) []registry.Model {
	interval := a.Config().EffectiveUpdateCheckInterval()
	a.updateMu.Lock()
	defer a.updateMu.Unlock()
	var due []registry.Model
	for _, m := range a.Registry.Ready() {
		if !m.VersionKnown() {
			continue
		}
		if m.Update != nil && now.Sub(m.Update.CheckedAt) < interval {
			continue
		}
		if at, ok := a.updateAttempted[dlKey(m.RepoID)]; ok && now.Sub(at) < interval {
			continue
		}
		due = append(due, m)
	}
	return due
}

// updateCheckTickAt is one tick of the schedule at the given wall-clock time:
// a round, when checks are on, for the models that are due.
func (a *App) updateCheckTickAt(ctx context.Context, now time.Time) {
	if !a.Config().UpdateCheck {
		return
	}
	due := a.updateCheckDue(now)
	if len(due) == 0 {
		return
	}
	a.checkForUpdatesAt(ctx, due, now)
}

// StartCheckingForUpdates runs the update check's schedule in the background
// under the app's own job context, beside the category job: the caller is
// runServer, after New has adopted whatever is on disk. It never takes the
// pool's lock or the download lock, so nothing it waits for — a Hub that is
// slow, or not there — can slow a request. The first tick comes after a short
// delay with jitter, so a start is not also a burst of requests; Close cancels
// the job and waits for it.
func (a *App) StartCheckingForUpdates() {
	a.updateStarted.Do(a.startCheckingForUpdates)
}

func (a *App) startCheckingForUpdates() {
	delay := 30*time.Second + rand.N(30*time.Second)
	a.jobsWG.Add(1)
	go func() {
		defer a.jobsWG.Done()
		select {
		case <-a.jobsCtx.Done():
			return
		case <-time.After(delay):
		}
		a.updateCheckTickAt(a.jobsCtx, time.Now())
		t := time.NewTicker(updateCheckTick)
		defer t.Stop()
		for {
			select {
			case <-a.jobsCtx.Done():
				return
			case <-t.C:
				a.updateCheckTickAt(a.jobsCtx, time.Now())
			}
		}
	}()
}

// CheckForUpdates asks the Hub about each model in turn and records what it
// finds, then logs one line for the round. A model the Hub does not answer
// for keeps the mark it had. The requests are the Hub's: no statistic moves.
//
// The setting is read again before every model, so turning checks off stops
// a round in flight: nothing more leaves the Mac once the switch is off. The
// Hub's remaining budget is forgotten at the start, so a low figure some
// other request heard does not end the round; only this round's answers do.
func (a *App) CheckForUpdates(ctx context.Context, models []registry.Model) UpdateRound {
	return a.checkForUpdatesAt(ctx, models, time.Now())
}

// checkForUpdatesAt is CheckForUpdates on the schedule's clock: each model is
// stamped as attempted at now when it is asked about, so a model a round did
// not reach — the round cut short by the rate limit or the switch — is due
// again at the next tick, not an interval later.
func (a *App) checkForUpdatesAt(ctx context.Context, models []registry.Model, now time.Time) UpdateRound {
	var round UpdateRound
	a.Hub.ForgetRateLimit()
	for i, m := range models {
		if i > 0 {
			select {
			case <-ctx.Done():
				return round
			case <-time.After(a.updatePause):
			}
		}
		if !a.Config().UpdateCheck {
			break
		}
		a.updateMu.Lock()
		a.updateAttempted[dlKey(m.RepoID)] = now
		a.updateMu.Unlock()
		found, err := a.checkOne(ctx, m)
		if ctx.Err() != nil {
			return round
		}
		if errors.Is(err, errChecksOff) {
			// Not a Hub that did not answer: the operator turned checks
			// off, and the round says nothing about the model it left.
			break
		}
		if err != nil {
			round.Unreachable++
			var ae *hub.APIError
			if errors.As(err, &ae) && ae.StatusCode == http.StatusTooManyRequests {
				round.StoppedEarly = true
				break
			}
		} else {
			found.CheckedAt = time.Now()
			if err := a.Registry.SetUpdate(m.RepoID, m.Commit, found); err == nil {
				round.Checked++
				if found.Status != registry.UpdateCurrent {
					round.Marked++
				}
			}
		}
		if n, ok := a.Hub.RateRemaining(); ok && n < updateCheckLowBudget && i < len(models)-1 {
			round.StoppedEarly = true
			break
		}
	}
	switch {
	case round.Checked == 0 && round.Unreachable > 0:
		a.Log.Info("could not reach HuggingFace to check models for newer versions; marks stay as they were",
			"models", round.Unreachable)
	case round.Checked > 0 || round.StoppedEarly:
		a.Log.Info("checked downloaded models for newer versions",
			"checked", round.Checked, "marked", round.Marked, "unreachable", round.Unreachable,
			"stopped_early", round.StoppedEarly)
	}
	return round
}

// checkOne asks the Hub about one model and says what it found, or why it
// could not tell. Only when the commit has moved is the listing at the new
// commit asked for, and only when config.json changed is that file read.
func (a *App) checkOne(ctx context.Context, m registry.Model) (registry.UpdateCheck, error) {
	ctx, cancel := context.WithTimeout(ctx, a.updateTimeout)
	defer cancel()
	up, err := a.Hub.Latest(ctx, m.RepoID)
	if err != nil {
		return registry.UpdateCheck{}, err
	}
	if up.Commit == m.Commit {
		return registry.UpdateCheck{Status: registry.UpdateCurrent}, nil
	}
	if !a.Config().UpdateCheck {
		return registry.UpdateCheck{}, errChecksOff
	}
	files, err := a.Hub.FilesAt(ctx, up)
	if err != nil {
		return registry.UpdateCheck{}, err
	}
	changed, configChanged, paths := a.compareVersions(m, hub.WantedFiles(files))
	if !changed {
		return registry.UpdateCheck{Status: registry.UpdateCurrent}, nil
	}
	// What a newer version is found to be, naming the files it changes.
	found := func(status string) (registry.UpdateCheck, error) {
		return newerVersion(status, up.Commit, paths), nil
	}
	if configChanged {
		if !a.Config().UpdateCheck {
			return registry.UpdateCheck{}, errChecksOff
		}
		listed := hub.File{Path: "config.json"}
		for _, f := range files {
			if f.Path == "config.json" {
				listed = f
				break
			}
		}
		b, err := a.Hub.SmallFileAt(ctx, up, listed, maxCheckedConfig)
		switch {
		case errors.Is(err, hub.ErrCrossOrigin), errors.Is(err, hub.ErrContentMismatch), errors.Is(err, hub.ErrOversizedBody):
			// The Hub answered, but with a config.json the check cannot
			// hold to the hash it lists — handed off its origin with no
			// sha256 to check, not the listed bytes, or past the bound —
			// so whether the newer version names a model_file is unknown,
			// and it is not offered (iss-2610042101439623).
			return found(registry.UpdateCannotCheck)
		case err != nil:
			return registry.UpdateCheck{}, err
		default:
			names, err := registry.ConfigNamesModelCode(b)
			if err != nil {
				return registry.UpdateCheck{}, err
			}
			if names {
				return found(registry.UpdateRunsOwnCode)
			}
		}
	}
	if a.reviewed != nil {
		if decision, reviewed := a.reviewed(m.RepoID); decision {
			for _, c := range reviewed {
				if c == up.Commit {
					return found(registry.UpdateAvailable)
				}
			}
			// Nothing to offer, and a decision model is never marked: the
			// 'awaiting review' mark is withdrawn (iss-2610042101436891), so
			// the check records what it records for a version with nothing
			// newer, which carries no mark and no Update.
			return registry.UpdateCheck{Status: registry.UpdateCurrent}, nil
		}
	}
	return found(registry.UpdateAvailable)
}

// newerVersion is what a check records for a newer version: its status, its
// commit, and the files it changes, the first registry.MaxUpdateFiles of them
// in order with the count of all (iss-2610042101436222).
func newerVersion(status, commit string, paths []string) registry.UpdateCheck {
	slices.Sort(paths)
	u := registry.UpdateCheck{Status: status, Commit: commit, FilesChanged: len(paths)}
	if len(paths) > 0 {
		u.Files = slices.Clone(paths[:min(len(paths), registry.MaxUpdateFiles)])
	}
	return u
}

// compareVersions says whether the files Dessau uses differ between the
// version on disk and the listing at the newer commit, whether config.json is
// among them, and which they are: a file changed, removed or added.
//
// Documentation is not compared: a README, a licence, any Markdown. A file
// the record holds a hash for differs when the listing gives another or
// lists it no more. A file the record has no hash for is unknown, not
// changed (iss-2610031239271873): it counts as added only when it is not on
// disk, which is what an added file is. A version recorded as its commit
// alone has no hashes to compare, so any move of the commit is a change, and
// which files it changes is not known: no path is named.
func (a *App) compareVersions(m registry.Model, files []hub.File) (changed, configChanged bool, paths []string) {
	if m.FileHashes == nil {
		return true, true, nil
	}
	// A config.json the record has no hash for is read whenever anything
	// changed: whether the newer one names a model_file cannot otherwise be
	// told.
	defer func() {
		if _, ok := m.FileHashes["config.json"]; !ok && changed {
			configChanged = true
		}
	}()
	dir := a.Paths.ModelDir(m.RepoID)
	listed := make(map[string]string, len(files))
	for _, f := range files {
		if isDocumentation(f.Path) {
			continue
		}
		h := f.OID
		if f.LFS != nil && f.LFS.OID != "" {
			h = f.LFS.OID
		}
		listed[f.Path] = strings.ToLower(h)
	}
	mark := func(p string) {
		changed = true
		paths = append(paths, p)
		if p == "config.json" {
			configChanged = true
		}
	}
	for p, had := range m.FileHashes {
		if isDocumentation(p) {
			continue
		}
		if now, ok := listed[p]; !ok || now != had {
			mark(p)
		}
	}
	for p := range listed {
		if _, recorded := m.FileHashes[p]; recorded {
			continue
		}
		if !filepath.IsLocal(filepath.FromSlash(p)) {
			continue
		}
		if _, err := os.Lstat(filepath.Join(dir, filepath.FromSlash(p))); errors.Is(err, os.ErrNotExist) {
			mark(p)
		}
	}
	return changed, configChanged, paths
}

// isDocumentation reports whether a repository file is one no model server
// reads: Markdown, and the licence and notice files a repository carries.
func isDocumentation(p string) bool {
	base := strings.ToLower(path.Base(p))
	if strings.HasSuffix(base, ".md") || strings.HasSuffix(base, ".markdown") {
		return true
	}
	for _, prefix := range []string{"license", "licence", "notice", "copying"} {
		if strings.HasPrefix(base, prefix) {
			return true
		}
	}
	return false
}
