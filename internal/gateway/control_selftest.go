package gateway

import (
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/intentdriven/Gropius/internal/config"
	"github.com/intentdriven/Gropius/internal/selftest"
)

// selfTestCache holds the last view served, keyed on the results file's size
// and modification time. The panel asks every two seconds and any local
// process may ask as often as it likes; the file changes at most once a run.
// Without this every ask is a parse of up to the file's cap. It is a leaf
// lock: nothing is taken while it is held.
type selfTestCache struct {
	mu    sync.Mutex
	size  int64
	mtime time.Time
	runs  []selftest.Run
	valid bool
}

// selfTestRecent is how many runs, newest first, the endpoint carries beside
// the latest run per model. The file is bounded, not small; the panel shows
// what happened lately, and the file is there for the rest.
const selfTestRecent = 50

// SelfTestView is what GET /api/selftest answers: whether the loop is on,
// the latest run for each model that has one, and the most recent runs. It
// carries no path — the file's location is the reference page's to state,
// not the endpoint's — and a run holds no prompt, answer, key or address
// (internal/selftest).
type SelfTestView struct {
	Enabled bool           `json:"enabled"`
	Latest  []selftest.Run `json:"latest"`
	Recent  []selftest.Run `json:"recent"`
}

// handleSelfTest serves the self-test's results to the panel. Loopback only,
// like every control-plane route: the figures are the operator's.
func (c *Control) handleSelfTest(w http.ResponseWriter, r *http.Request) {
	runs, err := c.selfTestRuns(filepath.Join(c.App.Paths.SelfTest, selftest.FileName))
	if err != nil {
		// The error names the path; the panel gets the fact, the log the
		// detail, and only at the level that carries paths.
		c.App.Log.Debug("the self-test results could not be read", "err", err)
		writeError(w, http.StatusInternalServerError, "the self-test results could not be read")
		return
	}
	writeJSON(w, http.StatusOK, selfTestView(c.App.SelfTest.Enabled(), runs))
}

// selfTestRuns reads the results file, or returns the last reading when the
// file has not changed since. A missing file is no runs, and is not cached:
// the first run to be written must be seen.
func (c *Control) selfTestRuns(path string) ([]selftest.Run, error) {
	st, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	c.selfTest.mu.Lock()
	defer c.selfTest.mu.Unlock()
	if c.selfTest.valid && st.Size() == c.selfTest.size && st.ModTime().Equal(c.selfTest.mtime) {
		return c.selfTest.runs, nil
	}
	runs, err := selftest.ReadResults(path)
	if err != nil {
		return nil, err
	}
	c.selfTest.size, c.selfTest.mtime, c.selfTest.runs, c.selfTest.valid = st.Size(), st.ModTime(), runs, true
	return runs, nil
}

// selfTestView folds the file's runs, oldest first as read, into the view.
func selfTestView(enabled bool, runs []selftest.Run) SelfTestView {
	view := SelfTestView{Enabled: enabled, Latest: []selftest.Run{}, Recent: []selftest.Run{}}
	latest := map[string]selftest.Run{}
	for _, run := range runs {
		key := config.FoldRepoID(run.Model)
		if prev, ok := latest[key]; !ok || run.At >= prev.At {
			latest[key] = run
		}
	}
	for _, run := range latest {
		view.Latest = append(view.Latest, run)
	}
	sort.Slice(view.Latest, func(i, j int) bool {
		return config.FoldRepoID(view.Latest[i].Model) < config.FoldRepoID(view.Latest[j].Model)
	})
	for i := len(runs) - 1; i >= 0 && len(view.Recent) < selfTestRecent; i-- {
		view.Recent = append(view.Recent, runs[i])
	}
	return view
}
