package runtime

import (
	"bytes"
	"context"
	"io"
	"os"
	"regexp"
	"strings"
	"syscall"
	"time"
	"unicode"
)

// LoadLogger is a Process that can say where its output is being written.
// It is optional, like Footprinter: the real launcher's process reports its
// per-model log, and the pool reads that log while it waits for the model to
// become ready, so a child that has already said it cannot load the model
// ends the wait at once instead of at the readiness timeout. A process
// without it, or one that reports no path, is waited for as it always was.
type LoadLogger interface {
	LogPath() string
}

// The real launcher's process is one.
var _ LoadLogger = (*execProcess)(nil)

// FatalLoadError is the cause a readiness wait is cancelled with when the
// child's log says the load cannot succeed. Line is the traceback's terminal
// line, bounded and stripped of anything path-shaped (fatalLoadLine), so it
// is safe to carry on a NotReadyError and out to an entitled client.
type FatalLoadError struct {
	Line string
}

func (e *FatalLoadError) Error() string { return e.Line }

// The bounds on reading the child's log: how much of its tail is read on
// each look, how often it is looked at, and how long a line is carried.
const (
	loadLogTailBytes  = 64 << 10
	loadLogPoll       = 500 * time.Millisecond
	maxFatalLineBytes = 300
)

// tracebackHeader opens a Python traceback; fatalLine is a terminal line of
// one that means the model did not load. The set is deliberately short —
// the kinds the 2026-09-21 child raised (a model type the runtime has no
// module for, raised as ModuleNotFoundError and re-raised as ValueError) and
// their import-time sibling — and not "any exception": a request handler's
// BrokenPipeError is a traceback the child survives, and reading it as a
// load failure would fail a model that was about to become ready.
var (
	tracebackHeader = "Traceback (most recent call last):"
	fatalLine       = regexp.MustCompile(`^(ValueError|ModuleNotFoundError|ImportError)(: .*)?$`)
)

// fatalLoadLine reads the tail of a child's log and reports the terminal
// line of a traceback that means the load cannot succeed, or "" and false
// when there is none yet. A log that does not exist yet, or cannot be read,
// is no verdict: the wait goes on to the readiness timeout as before.
//
// The line is what goes on the wire to an entitled client, so it is bounded
// and any whitespace-delimited token containing a path separator is replaced
// — a traceback's own frames name the venv under this account's home, and a
// message can name a weights file.
func fatalLoadLine(path string) (string, bool) {
	if path == "" {
		return "", false
	}
	// Opened the way the launcher opened it for writing: a link left under
	// the name is refused rather than followed, a FIFO cannot block the
	// open, and the handle is checked to be a regular file before it is
	// read. The directory is this account's own; this is the same
	// hardening on the reading side.
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return "", false
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return "", false
	}
	if size := info.Size(); size > loadLogTailBytes {
		if _, err := f.Seek(size-loadLogTailBytes, io.SeekStart); err != nil {
			return "", false
		}
	}
	tail, err := io.ReadAll(io.LimitReader(f, loadLogTailBytes))
	if err != nil {
		return "", false
	}
	// Only whole lines: the child may be mid-write, and a terminal line cut
	// short would be recorded as the reason.
	if i := bytes.LastIndexByte(tail, '\n'); i < 0 {
		return "", false
	} else {
		tail = tail[:i]
	}
	// The last such line, not the first: a chained exception ends in the
	// one the child actually raised — "Model type … not supported" after
	// the ModuleNotFoundError it was handling — and that is the line a
	// person reading the log's end would quote. A traceback is its header
	// and the indented frames after it; the terminal line is read only
	// straight after a frame, so a matching line the child writes on its
	// own later — a logged message, a traceback it survived and went on
	// from — is not taken for the verdict.
	inTraceback, found := false, ""
	for _, raw := range bytes.Split(tail, []byte("\n")) {
		line := strings.TrimRight(string(raw), "\r")
		switch {
		case line == tracebackHeader:
			inTraceback = true
		case line == "":
			// Blank lines separate a chained exception's parts.
		case strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t"):
			// A frame, its source line, or a caret marker.
		case inTraceback && fatalLine.MatchString(line):
			found = sanitizeFatalLine(line)
			inTraceback = false
		case strings.HasPrefix(line, "During handling of the above exception"),
			strings.HasPrefix(line, "The above exception was the direct cause"):
			// The chain's own joins; the next header follows.
		default:
			inTraceback = false
		}
	}
	return found, found != ""
}

// sanitizeFatalLine bounds the line, drops anything unprintable, and
// blanks everything from the first path-shaped token to the last as one
// path: a path with spaces in it — a directory name of the person's own —
// would otherwise leak its inner words one token at a time.
func sanitizeFatalLine(line string) string {
	line = strings.Map(func(r rune) rune {
		if r == ' ' || r == '\t' || (unicode.IsPrint(r) && !unicode.IsControl(r)) {
			return r
		}
		return -1
	}, line)
	fields := strings.Fields(line)
	first, last := -1, -1
	for i, w := range fields {
		if strings.Contains(w, "/") {
			if first < 0 {
				first = i
			}
			last = i
		}
	}
	if first >= 0 {
		fields = append(append(fields[:first:first], "<path>"), fields[last+1:]...)
	}
	out := strings.Join(fields, " ")
	if len(out) > maxFatalLineBytes {
		out = out[:maxFatalLineBytes]
	}
	return out
}

// watchLoadLog looks at the child's log until the wait ends, and cancels
// the wait with the child's own reason the moment the log says the load
// cannot succeed. It runs beside probeReady, whose one completion request
// can block for the whole readiness timeout against a child that has died
// in its generate thread while its httpd goes on answering.
func (p *Pool) watchLoadLog(ctx context.Context, path string, cancel context.CancelCauseFunc) {
	ticker := time.NewTicker(loadLogPoll)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		if line, ok := fatalLoadLine(path); ok {
			cancel(&FatalLoadError{Line: line})
			return
		}
	}
}
