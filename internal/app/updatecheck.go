package app

import (
	"context"
	"errors"
	"math/rand/v2"
	"net/http"
	"os"
	"path"
	"path/filepath"
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
// update only to a reviewed commit (itd-2610030656210408). Nil means no model
// is a decision model.
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
	a.updateMu.Lock()
	for _, m := range due {
		a.updateAttempted[dlKey(m.RepoID)] = now
	}
	a.updateMu.Unlock()
	a.CheckForUpdates(ctx, due)
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
		found, err := a.checkOne(ctx, m)
		if ctx.Err() != nil {
			return round
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
	changed, configChanged := a.compareVersions(m, hub.WantedFiles(files))
	if !changed {
		return registry.UpdateCheck{Status: registry.UpdateCurrent}, nil
	}
	if configChanged {
		if !a.Config().UpdateCheck {
			return registry.UpdateCheck{}, errChecksOff
		}
		b, err := a.Hub.SmallFileAt(ctx, up, "config.json", maxCheckedConfig)
		switch {
		case errors.Is(err, hub.ErrCrossOrigin):
			// A config.json the repository keeps in LFS is served from the
			// Hub's content CDN, which a check does not read from. The
			// newer version is offered; the update's own Precheck refuses
			// it if it names a model_file.
		case err != nil:
			return registry.UpdateCheck{}, err
		default:
			names, err := registry.ConfigNamesModelCode(b)
			if err != nil {
				return registry.UpdateCheck{}, err
			}
			if names {
				return registry.UpdateCheck{Status: registry.UpdateRunsOwnCode, Commit: up.Commit}, nil
			}
		}
	}
	if a.reviewed != nil {
		if decision, reviewed := a.reviewed(m.RepoID); decision {
			for _, c := range reviewed {
				if c == up.Commit {
					return registry.UpdateCheck{Status: registry.UpdateAvailable, Commit: up.Commit}, nil
				}
			}
			return registry.UpdateCheck{Status: registry.UpdateAwaitingReview, Commit: up.Commit}, nil
		}
	}
	return registry.UpdateCheck{Status: registry.UpdateAvailable, Commit: up.Commit}, nil
}

// compareVersions says whether the files Dessau uses differ between the
// version on disk and the listing at the newer commit, and whether
// config.json is among them.
//
// Documentation is not compared: a README, a licence, any Markdown. A file
// the record holds a hash for differs when the listing gives another or
// lists it no more. A file the record has no hash for is unknown, not
// changed (iss-2610031239271873): it counts as added only when it is not on
// disk, which is what an added file is. A version recorded as its commit
// alone has no hashes to compare, so any move of the commit is a change.
func (a *App) compareVersions(m registry.Model, files []hub.File) (changed, configChanged bool) {
	if m.FileHashes == nil {
		return true, true
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
	return changed, configChanged
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
