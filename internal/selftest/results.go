package selftest

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"syscall"
)

// A results file is JSON Lines, one Run a line, appended to until it would
// pass its cap and then started again: a single bounded file, no rotation.
// The repository already has two size-rotating writers (internal/stats and
// internal/applog) and iss-2609091714393599 is about there being two; this
// is deliberately not a third. Thousands of runs fit under the default cap,
// which is more days of a model-a-day cycle than a result stays interesting.
const (
	// DefaultMaxBytes is the cap on the results file.
	DefaultMaxBytes int64 = 4 << 20
	// FileName is the results file's name under config.Paths.SelfTest.
	FileName = "results.jsonl"
)

// KindRun is the only kind of line the file holds today; the field is there
// so a second kind can be added without every reader guessing.
const KindRun = "run"

// The outcomes a run can have.
const (
	// OutcomeOK is every test in the set completed.
	OutcomeOK = "ok"
	// OutcomeYielded is a run abandoned because a real request arrived.
	OutcomeYielded = "yielded"
	// OutcomeStopped is a run abandoned because the switch went off.
	OutcomeStopped = "stopped"
	// OutcomeFailed is a run that could not load its model or whose request
	// failed; Reason says which.
	OutcomeFailed = "failed"
)

// The reasons a failed run carries. A class, not the error's text: the error
// is the model server's or the pool's and may name a path.
const (
	ReasonLoad    = "load"
	ReasonRequest = "request"
)

// Where a test's token counts came from.
const (
	CountedUsage  = "usage"
	CountedChunks = "chunks"
)

// Run is one line of the file: one model, measured once.
type Run struct {
	Kind  string `json:"kind"`
	Model string `json:"model"`
	// At is when the run started, Unix seconds UTC.
	At      int64  `json:"at"`
	Outcome string `json:"outcome"`
	Reason  string `json:"reason,omitempty"`
	// ColdLoad says the model was not resident when the run started, so
	// LoadMs is a load from disk and the run unloaded it afterwards.
	ColdLoad bool `json:"cold_load"`
	// LoadMs is how long Acquire took: the load, or nothing much when the
	// model was already there.
	LoadMs int64 `json:"load_ms"`
	// Tests are the tests that completed, in the order they ran. A yielded or
	// failed run carries the ones that finished before it stopped.
	Tests []Test `json:"tests"`
}

// Test is one test's figures.
type Test struct {
	Name string `json:"name"`
	// Parallel is how many requests were sent at once; 1 for the single
	// tests.
	Parallel int `json:"parallel"`
	// PromptTokens and CompletionTokens are the model server's own counts,
	// summed across a parallel test.
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	// FirstTokenMs is how long the first piece of the answer took; the mean
	// across a parallel test.
	FirstTokenMs int64 `json:"first_token_ms"`
	// TotalMs is how long the whole request took; the wall time of a
	// parallel test.
	TotalMs int64 `json:"total_ms"`
	// PromptTokensPerSec is the prompt's tokens over the time to the first
	// token: how fast the model read. Zero for a parallel test.
	PromptTokensPerSec float64 `json:"prompt_tokens_per_sec"`
	// TokensPerSec is the generation rate: tokens after the first over the
	// time after it, or for a parallel test every token the batch produced
	// over its wall time.
	TokensPerSec float64 `json:"tokens_per_sec"`
	// Counted says where the counts came from: "usage" for the server's own
	// event, "chunks" when it sent none and the chunks were counted instead.
	Counted string `json:"counted"`
}

// RunFields and TestFields are the JSON names of the two records, for the
// docs test that holds the reference page to them.
func RunFields() []string  { return jsonFields(Run{}) }
func TestFields() []string { return jsonFields(Test{}) }

func jsonFields(v any) []string {
	t := reflect.TypeOf(v)
	out := make([]string, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		name, _, _ := strings.Cut(t.Field(i).Tag.Get("json"), ",")
		out = append(out, name)
	}
	return out
}

// file is the bounded writer.
type file struct {
	path     string
	maxBytes int64
	log      *slog.Logger
	mu       sync.Mutex
}

// filePerm is the mode the file carries: this account's own, like the log and
// the statistics store. Its directory is created 0700 for the same reason.
const filePerm os.FileMode = 0o600

// The opens never follow a link and never block: on a shared Mac the
// directory's parent is this account's own, but a planted link or a FIFO in
// the wrong place must be refused rather than followed or waited on — the
// writer runs on the settings path, and a hang there is a hang of every
// save (the same reasoning as internal/stats and internal/applog).
const (
	fileFlags = os.O_CREATE | os.O_WRONLY | os.O_APPEND | syscall.O_NOFOLLOW | syscall.O_NONBLOCK
	readFlags = os.O_RDONLY | syscall.O_NOFOLLOW | syscall.O_NONBLOCK
)

// write appends one run, starting the file again first if the line would
// take it past the cap. A write that fails is logged and dropped: a result
// is not worth stopping the loop for, and the next run will try again.
func (f *file) write(run Run) {
	f.mu.Lock()
	defer f.mu.Unlock()
	line, err := json.Marshal(run)
	if err != nil {
		f.log.Warn("self-test: could not encode a result", "err", err)
		return
	}
	line = append(line, '\n')
	if err := ensureDir(filepath.Dir(f.path)); err != nil {
		f.log.Warn("self-test: the results directory is not safe to write; the result is dropped")
		f.log.Debug("self-test: results directory refused", "err", err)
		return
	}
	h, err := openRegular(f.path, fileFlags, filePerm)
	if err != nil {
		f.log.Warn("self-test: could not open the results file; the result is dropped")
		f.log.Debug("self-test: results file refused", "err", err)
		return
	}
	defer h.Close()
	if st, err := h.Stat(); err == nil && st.Size()+int64(len(line)) > f.maxBytes {
		if err := h.Truncate(0); err != nil {
			f.log.Warn("self-test: could not start the results file again", "err", err)
			return
		}
		f.log.Info("self-test: the results file reached its cap and was started again", "cap_bytes", f.maxBytes)
	}
	if _, err := h.Write(line); err != nil {
		f.log.Warn("self-test: could not write a result", "err", err)
	}
}

// ReadResults reads every run in a results file, oldest first. A file that
// does not exist is no runs; a line that does not decode is skipped, since
// the file is started again at its cap and a reader that refused the whole
// file over one line would show nothing.
func ReadResults(path string) ([]Run, error) {
	h, err := openRegular(path, readFlags, 0)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer h.Close()
	var runs []Run
	sc := bufio.NewScanner(h)
	sc.Buffer(make([]byte, 0, 64<<10), 1<<20)
	for sc.Scan() {
		var run Run
		if err := json.Unmarshal(sc.Bytes(), &run); err != nil || run.Kind != KindRun {
			continue
		}
		runs = append(runs, run)
	}
	if err := sc.Err(); err != nil {
		return runs, fmt.Errorf("read %s: %w", path, err)
	}
	return runs, nil
}

// ensureDir creates the results directory, closed, or checks the one that is
// there: a directory this account owns, that nobody else can write, reached
// without following a link. Its parent is the account's own directory —
// created 0700 under a shared install, and the account-owned root elsewhere
// — so a single Mkdir is enough and nothing above it is walked: what keeps
// the parent honest is its ownership, not its mode.
func ensureDir(dir string) error {
	err := os.Mkdir(dir, 0o700)
	if err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	st, err := os.Lstat(dir)
	if err != nil {
		return err
	}
	if st.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s is a symbolic link", dir)
	}
	if !st.IsDir() {
		return fmt.Errorf("%s is not a directory", dir)
	}
	if st.Mode().Perm()&0o022 != 0 {
		return fmt.Errorf("%s is writable by other accounts", dir)
	}
	if sys, ok := st.Sys().(*syscall.Stat_t); ok && int(sys.Uid) != os.Getuid() {
		return fmt.Errorf("%s is owned by another account", dir)
	}
	return nil
}

// openRegular opens a path and refuses anything but a regular file: a FIFO
// opened non-blocking answers at once, and a device or a socket answers with
// something that is not a file of ours.
func openRegular(path string, flags int, perm os.FileMode) (*os.File, error) {
	h, err := os.OpenFile(path, flags, perm)
	if err != nil {
		return nil, err
	}
	st, err := h.Stat()
	if err != nil {
		h.Close()
		return nil, err
	}
	if !st.Mode().IsRegular() {
		h.Close()
		return nil, fmt.Errorf("%s is not a regular file", path)
	}
	return h, nil
}
