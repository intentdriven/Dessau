// Package applog builds the log Gropius keeps about itself.
//
// It exists because the app had nowhere to say why. Everything the server knew
// about a refusal, a failed launch or a model leaving memory went to standard
// error, and a Finder-launched .app has no standard error — it goes to the
// bit bucket the moment launchd hands the process over. So the one place an
// operator could read the reason behind a refusal their client saw was a
// terminal they were not using.
//
// What this package adds is a file beside the model servers' own logs, in the
// account's own directory, written at the same time as stderr and from the
// same handler, so `make run` is unchanged and a Finder launch is no longer
// mute. The level is a slog.LevelVar the operator moves from Settings, which
// is why it is exported rather than baked into the handler: the handler is
// built once and the level is read on every line.
//
// Nothing here decides what goes on a line. That is the caller's business, and
// the rule the whole feature rests on — no prompt, no answer, no key, no
// token, no client address, at any level — is held where the lines are written
// and by the scans in internal/archtest, not here.
//
// The package has a second job, in rotate.go: it is the canonical home of the
// size-rotating file writer and of the guarded open underneath it, which
// internal/stats and internal/selftest write their own bounded files through.
// The log was the plain case of the three, so the primitive was extracted here
// rather than out of the statistics store's specialisation of it
// (iss-2609091714393599). This file is the log; that one is the primitive, and
// it names nothing of the log's.
package applog

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

// DefaultName is the current log's file name. It is fixed rather than dated so
// that "the log" is one path a person can tail, which is what an operator
// chasing a refusal actually does; the rotated files carry the numbers.
const DefaultName = "gropius.log"

// DefaultRotateBytes is how large the current log grows before the next one is
// started, and DefaultKeep is how many files survive — the current one and its
// predecessors together. Five megabytes is what internal/stats already rotates
// a file at, so the two logs on this Mac grow in the same unit; five files
// bound the whole log under 25 MB, which is small enough to leave alone on any
// Mac that can run a model and long enough to hold a session at the detailed
// level.
//
// They are constants because docs/logging.md prints them and an archtest holds
// the page to these names.
const (
	DefaultRotateBytes = 5 << 20
	DefaultKeep        = 5
)

// Options build a Log.
type Options struct {
	// Dir is the directory the log file lives in: config.Paths.Logs, which
	// resolves through the account directory, so one account's log is never
	// written into the shared root.
	Dir string
	// Name is the current log's file name. Empty means DefaultName.
	Name string
	// RotateBytes and Keep bound the log. Zero or negative means the defaults.
	RotateBytes int64
	Keep        int
	// Stderr is the second destination, and the only one when the file cannot
	// be opened. Nil means os.Stderr; a caller that genuinely wants no console
	// output passes io.Discard.
	Stderr io.Writer
	// Level is the level the logger starts at. The zero value is slog.LevelInfo,
	// which is the sparse level — so a caller that forgets it gets the default
	// the setting's own default names.
	Level slog.Level
}

// Log is the process's logger, the level it reads, and where its file is.
type Log struct {
	// Logger is what everything else in the process is handed.
	Logger *slog.Logger
	// Level is the one variable the handler reads on every line. Setting it
	// takes effect on the next line, which is what lets Settings change the
	// level without a restart.
	Level *slog.LevelVar
	// Path is the current log file, or "" when there is no file — a directory
	// that could not be used leaves the logger writing to stderr alone.
	Path string

	file *Rotator
}

// Open builds the log.
//
// It returns a usable *Log even when it returns an error, and the two are
// independent: the error describes the file, and the Log is at worst the
// stderr-only logger the process had before this package existed. Nothing in
// the start-up path may exit because a log file could not be created — a
// server that will not start because it cannot write about starting is a
// worse outcome than one that starts quietly.
func Open(opts Options) (*Log, error) {
	if opts.Name == "" {
		opts.Name = DefaultName
	}
	if opts.RotateBytes <= 0 {
		opts.RotateBytes = DefaultRotateBytes
	}
	if opts.Keep <= 0 {
		opts.Keep = DefaultKeep
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}

	level := new(slog.LevelVar)
	level.Set(opts.Level)

	l := &Log{Level: level}
	out := opts.Stderr
	file, err := OpenRotator(RotateOptions{
		Dir:      opts.Dir,
		Name:     opts.Name,
		MaxBytes: opts.RotateBytes,
		Keep:     opts.Keep,
	})
	if err == nil {
		// stderr first, deliberately. io.MultiWriter stops at the first error,
		// so a file that fails mid-run costs the file and not the console.
		out = io.MultiWriter(opts.Stderr, file)
		l.file = file
		l.Path = filepath.Join(opts.Dir, opts.Name)
	}
	l.Logger = slog.New(slog.NewTextHandler(out, &slog.HandlerOptions{Level: level}))
	return l, err
}

// Close flushes and closes the file. It is safe to call more than once, and a
// logger that has been closed goes on writing to stderr — the app closes this
// from a defer that a second path can reach.
func (l *Log) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	return l.file.Close()
}
