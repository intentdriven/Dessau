package app

import (
	"context"
	"time"
)

// categoryPause is the gap between two models' requests to the Hub in
// CompleteCategories. The job runs at every start, for every model with no
// word, and a Mac that adopted a whole shared cache at once has several; one
// request a second is a courtesy to the Hub and costs nobody anything, since
// nothing waits on the job.
const categoryPause = time.Second

// CompleteCategories asks the Hub what each ready model with no word is — a
// model adopted from the shared cache by an account that never downloaded
// it, or downloaded while the Hub was unreachable — and records the answer,
// so that a model judged by its chat template until now is judged by the
// operator's rule from then on (iss-2609202237468921).
//
// It is started once per process, in the background, after the startup
// rescan has adopted whatever is on disk (StartCompletingCategories); it is
// never triggered by a client and nothing waits on it — not the start, not a
// load, not a request — except Close, which joins it like every other job. One
// model at a time with a pause between, through repoCategory, which bounds
// each request and logs the one line a Hub that does not answer earns. A Hub
// that does not answer for a model leaves it as it was, to be asked again at
// the next start; a Hub that answers with no words marks the model HubSilent
// so it is not asked again. The requests are the Hub's, not a model's: they
// touch no statistic. With nothing to do it logs nothing; having completed
// models it logs one line naming how many.
func (a *App) CompleteCategories(ctx context.Context) {
	var todo []string
	for _, m := range a.Registry.Ready() {
		if m.HasHubWord() || m.HubSilent {
			continue
		}
		todo = append(todo, m.RepoID)
	}
	completed := 0
	for i, repoID := range todo {
		if i > 0 {
			select {
			case <-ctx.Done():
				// Cut short, not abandoned: what was completed is still
				// counted below.
				todo = nil
			case <-time.After(categoryPause):
			}
			if todo == nil {
				break
			}
		}
		pipelineTag, tags, answered := a.repoCategory(ctx, repoID)
		if !answered {
			continue
		}
		if err := a.Registry.SetCategory(repoID, pipelineTag, tags); err != nil {
			// Removed between the walk and the answer, or the index would
			// not write: nothing to record, and nothing to retry.
			a.Log.Warn("could not record what the hub says this model is", "model", repoID, "err", err)
			continue
		}
		completed++
	}
	if completed > 0 {
		a.Log.Info("recorded what the hub says about models that had no word", "models", completed)
	}
}

// StartCompletingCategories runs CompleteCategories in the background under
// the app's own job context, once per process: the caller is runServer, after
// New has adopted whatever is on disk. Close cancels the job and waits for it.
func (a *App) StartCompletingCategories() {
	a.jobsWG.Add(1)
	go func() {
		defer a.jobsWG.Done()
		a.CompleteCategories(a.jobsCtx)
	}()
}
