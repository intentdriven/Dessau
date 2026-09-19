package applog

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
)

// This file is the repository's one size-rotating file writer, and the one
// guarded open underneath it.
//
// Gropius writes three bounded line files on this Mac: this package's own log,
// internal/stats's request statistics store, and internal/selftest's results.
// Each of them appends lines to a file whose name is predictable, in a
// directory anything running as this account can reach, and each of them used
// to carry its own copy of the same four rules — open through an os.Root on the
// directory, follow no link, wait on nothing, and refuse a handle that is not
// this account's own regular file. iss-2609091714393599 recorded what that
// costs: "the two drifting on the discipline they share — one gaining a check
// on the opened handle, or a mode, or a symlink refusal that the other does
// not", which is exactly what had happened by the time it was extracted.
//
// So the discipline lives here once, in OpenIn, and the plain rotation that
// two of the three want lives here with it as Rotator. The statistics store
// keeps its own writer on top of OpenIn rather than on Rotator: its names carry
// a UTC day and a counter, its retention holds a months horizon as well as a
// byte ceiling, and what it drops is folded into a summary that counts toward
// that ceiling. That is a specialisation of the rotation, not a use of it, and
// the store's on-disk format is this repository's durable data.
//
// What is NOT here is the directory policy, and deliberately. The three
// directories are held to three different rules for three recorded reasons —
// the store walks every ancestor for group-write and foreign ownership, the
// self-test's results checks the one directory it makes, and the log creates
// its own and falls back to stderr rather than refusing to start. Folding those
// into one would either weaken the store's rule or make a log directory able to
// stop the log. The caller checks its directory and hands this the handle.

// FilePerm is the mode every file these writers create carries: this account's
// own record of what its own Mac did, and nobody else's business.
const FilePerm os.FileMode = 0o600

// AppendFlags and ReadFlags are how every one of these files is opened.
//
// O_NOFOLLOW refuses a link left under the name, O_NONBLOCK refuses to wait on
// a named pipe left there instead, and neither of them is optional: the names
// are predictable and the directories are reachable. It is the discipline
// internal/runtime/launcher.go applies to a model server's log, with O_APPEND
// where that has O_TRUNC — these files outlive one run, and rotation rather
// than truncation is what bounds them.
const (
	AppendFlags = os.O_CREATE | os.O_WRONLY | os.O_APPEND | syscall.O_NONBLOCK | syscall.O_NOFOLLOW
	ReadFlags   = os.O_RDONLY | syscall.O_NONBLOCK | syscall.O_NOFOLLOW
)

// OpenIn opens name inside root and refuses anything that is not a file of
// ours. It returns the open file and the stat of the handle itself, which is
// what a caller continuing an existing file reads its size from.
//
// The flags stop a link and a pipe being followed or waited on; the stat is
// what stops the rest. A handle that is not a regular file is refused however
// it was opened — O_NOFOLLOW says nothing about a device or a directory. And
// when perm says the caller is writing, a file carrying permission bits outside
// perm is refused as well: these writers create at FilePerm and a umask can
// only take bits away, so a file in one of these directories that any other
// account can read or write is not one of ours, whatever its name says. A read
// passes perm 0 and gets the regular-file check alone.
func OpenIn(root *os.Root, name string, flags int, perm os.FileMode) (*os.File, os.FileInfo, error) {
	f, err := root.OpenFile(name, flags, perm)
	if err != nil {
		return nil, nil, err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, nil, err
	}
	if !info.Mode().IsRegular() {
		f.Close()
		return nil, nil, fmt.Errorf("%s is not a regular file", name)
	}
	if perm != 0 && info.Mode().Perm()&^perm != 0 {
		f.Close()
		return nil, nil, fmt.Errorf("%s is mode %04o, which this account's own writers never create", name, info.Mode().Perm())
	}
	return f, info, nil
}

// RotateOptions build a Rotator.
type RotateOptions struct {
	// Dir is the directory the file lives in. It is created 0700 when it is
	// missing, and refused when what stands there is not a directory. Anything
	// stricter than that is the caller's to check first: see the note at the
	// top of this file.
	Dir string
	// Name is the current file's name. The rotated ones take its stem and
	// extension with a number between them.
	Name string
	// MaxBytes is how large the current file grows before the next one is
	// started. Zero or negative means DefaultRotateBytes.
	MaxBytes int64
	// Keep is how many files survive, the current one among them. Zero or
	// negative means DefaultKeep; 1 means no history at all, so the file is
	// simply started again when it fills.
	Keep int
	// Rotated, when set, is called after a file has been started again. It
	// runs on the writing goroutine with the Rotator's lock held, so it must
	// not write to this Rotator — the log is not allowed to call it for that
	// reason, and the self-test uses it to say in the log that its results
	// file reached its cap.
	Rotated func()
}

// Rotator is the plain size-rotating file: append until the next line would
// take the file past its cap, then shift the names along and start again.
//
// It is an io.Writer and it is safe for concurrent use.
type Rotator struct {
	root *os.Root
	name string
	base string
	ext  string
	max  int64
	keep int
	// rotated is RotateOptions.Rotated.
	rotated func()
	mu      sync.Mutex
	f       *os.File
	n       int64
	// dead marks a writer that has failed to rotate. See Write.
	dead bool
}

// OpenRotator opens the current file for appending, continuing it rather than
// emptying it when it is already there.
func OpenRotator(opts RotateOptions) (*Rotator, error) {
	if opts.MaxBytes <= 0 {
		opts.MaxBytes = DefaultRotateBytes
	}
	if opts.Keep <= 0 {
		opts.Keep = DefaultKeep
	}
	// Created when it is missing, and only then. On a first run — and on a
	// shared-cache install where this account has never run Gropius per-user —
	// nothing has made this directory yet, and the run whose log is most worth
	// having is the first one. Owner-only, following internal/stats's own rule
	// for the directory it creates: config.Paths.EnsureDirs leaves an existing
	// non-shared directory's mode alone, so a 0700 made here survives.
	if _, err := os.Lstat(opts.Dir); errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(opts.Dir, 0o700); err != nil {
			return nil, fmt.Errorf("create directory %s: %w", opts.Dir, err)
		}
	}
	// Lstat rather than Stat, exactly as the launcher does: a link standing
	// where the directory should be would otherwise let anything that can
	// write the parent choose where this account's file is written, and a
	// path-based MkdirAll would have followed it rather than reporting it.
	fi, err := os.Lstat(opts.Dir)
	if err != nil {
		return nil, fmt.Errorf("directory %s: %w", opts.Dir, err)
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", opts.Dir)
	}
	root, err := os.OpenRoot(opts.Dir)
	if err != nil {
		return nil, fmt.Errorf("open directory %s: %w", opts.Dir, err)
	}
	ext := filepath.Ext(opts.Name)
	r := &Rotator{
		root:    root,
		name:    opts.Name,
		base:    strings.TrimSuffix(opts.Name, ext),
		ext:     ext,
		max:     opts.MaxBytes,
		keep:    opts.Keep,
		rotated: opts.Rotated,
	}
	if err := r.open(); err != nil {
		root.Close()
		return nil, err
	}
	return r, nil
}

// open opens the current file for appending and records how large it already
// is, so a restart continues the file rather than emptying it.
func (r *Rotator) open() error {
	f, info, err := OpenIn(r.root, r.name, AppendFlags, FilePerm)
	if err != nil {
		return fmt.Errorf("create %s: %w", r.name, err)
	}
	r.f, r.n = f, info.Size()
	return nil
}

// Write appends one record, rotating first if it would not fit.
//
// A write that fails is reported to the caller, which for the log is an
// io.MultiWriter with stderr written first: the line the operator can still see
// is never lost to a full disk. A rotation that fails puts the writer to sleep
// rather than retrying per line — a directory that has stopped working will not
// start again inside this process, and a writer that tries every line turns one
// broken directory into a syscall storm on the request path.
func (r *Rotator) Write(b []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.dead || r.f == nil {
		return 0, os.ErrClosed
	}
	if r.n > 0 && r.n+int64(len(b)) > r.max {
		if err := r.rotate(); err != nil {
			r.dead = true
			return 0, err
		}
		if r.rotated != nil {
			r.rotated()
		}
	}
	n, err := r.f.Write(b)
	r.n += int64(n)
	return n, err
}

// rotate closes the current file, shifts the numbered names along, removes
// what falls past keep, and opens a fresh current file.
//
// The shift runs from the oldest name down so that no rename overwrites a file
// that has not been moved yet, and every one of them goes through the os.Root:
// a link or a directory left under a rotated name cannot become the place the
// previous file ends up, because Rename inside a root replaces the name rather
// than following it.
func (r *Rotator) rotate() error {
	if err := r.f.Close(); err != nil {
		return err
	}
	r.f, r.n = nil, 0
	// keep counts the current file as one of them, so the highest number that
	// survives is keep-1 and anything at or above it goes.
	for i := r.keep - 1; i >= 1; i-- {
		from := r.name
		if i > 1 {
			from = r.numbered(i - 1)
		}
		to := r.numbered(i)
		if i == r.keep-1 {
			// The file this rename is about to land on is the oldest kept one,
			// and it is what falls out of the window.
			if err := r.root.Remove(to); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
		if err := r.root.Rename(from, to); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	// keep == 1 means no history at all: the current file is simply removed
	// and started again.
	if r.keep == 1 {
		if err := r.root.Remove(r.name); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return r.open()
}

// numbered is the name of the nth previous file: gropius.1.log, gropius.2.log…
func (r *Rotator) numbered(n int) string {
	return fmt.Sprintf("%s.%d%s", r.base, n, r.ext)
}

// Close closes the file and the directory handle. Calling it twice is not an
// error, and neither is writing afterwards: a closed writer reports
// os.ErrClosed to its caller, which for the log is a multi-writer that has
// already written the line to stderr.
func (r *Rotator) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.f == nil {
		return nil
	}
	err := r.f.Close()
	r.f = nil
	r.root.Close()
	return err
}
