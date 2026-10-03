package runtime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/intentdriven/Dessau/internal/config"
)

// Distinct repo ids must map to distinct log files. ValidRepoID admits
// underscores in both components, so a separator the id itself can contain
// collapses ids like these into one path — and Launch opens the log O_TRUNC,
// so the collision truncates a live log, not just a stale one.
func TestLogFileNamesDoNotCollideAcrossDistinctRepoIDs(t *testing.T) {
	a, b := logFileName("a/b_c"), logFileName("a_b/c")
	if a == b {
		t.Fatalf("logFileName maps distinct repo ids to one file %q — launching the second model truncates the first model's live log", a)
	}
}

// The launcher writes the per-model log into the logs directory it is given
// and nowhere else. The log is opened O_CREATE|O_TRUNC at 0600 under a name
// derived from the repo id, so a same-named log in another installation's
// logs directory — one this process cannot even write — must neither stop the
// launch nor be touched by it.
//
// A single-uid test cannot own a file as another account, so the other
// installation's log is made unwritable instead, which fails an open the same
// way. Where the logs directory comes from is pinned in internal/config.
func TestTheLauncherWritesOnlyIntoItsOwnLogDirectory(t *testing.T) {
	elsewhere := t.TempDir()
	model := filepath.Join(elsewhere, "models", "org", "name")
	if err := os.MkdirAll(model, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(model, "config.json"), []byte(plainConfig), 0o644); err != nil {
		t.Fatal(err)
	}

	account := func(t *testing.T) config.Paths {
		t.Helper()
		p := config.NewPaths(t.TempDir())
		if err := os.MkdirAll(filepath.Dir(p.VenvPython()), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p.VenvPython(), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(p.Logs, 0o755); err != nil {
			t.Fatal(err)
		}
		return p
	}
	first, second := account(t), account(t)

	// The other installation has served this model: its log exists and this
	// process cannot write it.
	firstLog := filepath.Join(first.Logs, logFileName("org/name"))
	if err := os.WriteFile(firstLog, []byte("the first account's log\n"), 0o400); err != nil {
		t.Fatal(err)
	}
	if _, err := os.OpenFile(firstLog, os.O_WRONLY|os.O_TRUNC, 0); err == nil {
		t.Skip("this filesystem does not enforce write permission on the owner")
	}

	l := &ExecLauncher{Paths: second, LogDir: second.Logs}
	p, err := l.Launch(context.Background(), Spec{RepoID: "org/name", ModelPath: model, Socket: privateSocket(t)})
	if p != nil {
		<-p.Done()
	}
	if err != nil {
		t.Fatalf("the launch failed over another installation's log: %v", err)
	}
	if _, err := os.Stat(filepath.Join(second.Logs, logFileName("org/name"))); err != nil {
		t.Errorf("the launcher's own log was not written: %v", err)
	}
	if b, _ := os.ReadFile(firstLog); string(b) != "the first account's log\n" {
		t.Errorf("the other installation's log was touched: %q", b)
	}
}

// The per-model log has a predictable name in the logs directory, so a
// symlink planted there would make the truncating open land on any file this
// account can write. Launch must refuse to open anything but a regular file — and must
// not block on a planted FIFO either.
func TestLaunchRefusesSymlinkedLogFile(t *testing.T) {
	root := t.TempDir()
	paths := config.NewPaths(root)
	if err := os.MkdirAll(filepath.Dir(paths.VenvPython()), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.VenvPython(), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(paths.Logs, 0o755); err != nil {
		t.Fatal(err)
	}
	victim := filepath.Join(t.TempDir(), "victim")
	if err := os.WriteFile(victim, []byte("precious"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(victim, filepath.Join(paths.Logs, logFileName("org/name"))); err != nil {
		t.Fatal(err)
	}

	l := &ExecLauncher{Paths: paths, LogDir: paths.Logs}
	p, err := l.Launch(context.Background(), Spec{RepoID: "org/name", ModelPath: plainModelDir(t), Socket: privateSocket(t)})
	if p != nil {
		<-p.Done()
	}
	if err == nil {
		t.Fatal("Launch opened a symlinked log file")
	}
	if b, _ := os.ReadFile(victim); string(b) != "precious" {
		t.Fatalf("victim file was truncated through the symlink: %q", b)
	}
}

func TestLaunchDoesNotBlockOnFIFOLogFile(t *testing.T) {
	root := t.TempDir()
	paths := config.NewPaths(root)
	if err := os.MkdirAll(filepath.Dir(paths.VenvPython()), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.VenvPython(), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(paths.Logs, 0o755); err != nil {
		t.Fatal(err)
	}
	fifo := filepath.Join(paths.Logs, logFileName("org/name"))
	if err := syscall.Mkfifo(fifo, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if fd, err := syscall.Open(fifo, syscall.O_RDONLY|syscall.O_NONBLOCK, 0); err == nil {
			syscall.Close(fd)
		}
	})
	done := make(chan error, 1)
	go func() {
		p, err := (&ExecLauncher{Paths: paths, LogDir: paths.Logs}).Launch(context.Background(),
			Spec{RepoID: "org/name", ModelPath: plainModelDir(t), Socket: privateSocket(t)})
		if p != nil {
			<-p.Done()
		}
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("Launch accepted a FIFO as its log file")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Launch blocked opening a FIFO planted as the log file")
	}
}

// An unarmed model server is started at INFO, whatever else its Spec carries,
// and an armed one at DEBUG — named once, and only then.
//
// At DEBUG the pinned model server writes every request body and every
// response it produces to its log — prompts and completions. So this is not a
// formatting detail: it is the line between the content-free record the
// operator opted into and a full transcript on disk. The level comes from
// Spec.DebugLog and from nothing else (adr-2609201008477513 narrows
// adr-2609061503319212 to exactly that). The architecture tests keep the
// statistics switch out of this package; this one asserts what a model server
// is actually launched with, off the argument vector rather than off the
// source.
func TestAnUnarmedModelServerIsLaunchedAtInfoAndAnArmedOneAtDebug(t *testing.T) {
	temp := 0.7
	specs := map[string]Spec{
		"a plain spec":            {RepoID: "org/a", ModelPath: "/models/org/a"},
		"with sampling defaults":  {RepoID: "org/b", ModelPath: "/models/org/b", Sampling: config.Sampling{Temperature: &temp}},
		"with decode concurrency": {RepoID: "org/c", ModelPath: "/models/org/c", DecodeConcurrency: 4},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			argv := launchArgs(spec)
			got, ok := flagValue(argv, "--log-level")
			if !ok {
				t.Fatalf("the model server is launched with no log level at all: %v", argv)
			}
			if got != "INFO" {
				t.Errorf("the model server is launched at %q, want INFO — above it the server writes every prompt and every answer to its log", got)
			}
			for _, arg := range argv {
				if arg == "DEBUG" {
					t.Errorf("the argument vector of an unarmed launch names DEBUG: %v", argv)
				}
			}
		})
		t.Run(name+", armed", func(t *testing.T) {
			spec.DebugLog = true
			argv := launchArgs(spec)
			got, ok := flagValue(argv, "--log-level")
			if !ok {
				t.Fatalf("the armed model server is launched with no log level at all: %v", argv)
			}
			if got != "DEBUG" {
				t.Errorf("the armed model server is launched at %q, want DEBUG", got)
			}
			if n := countArg(argv, "--log-level"); n != 1 {
				t.Errorf("the armed argument vector names --log-level %d times, want exactly once: %v", n, argv)
			}
		})
	}
}

// countArg is how many times an argument appears in argv.
func countArg(argv []string, arg string) int {
	n := 0
	for _, a := range argv {
		if a == arg {
			n++
		}
	}
	return n
}

// stubbedLauncher is an ExecLauncher whose interpreter is a shell script, so
// Launch can be exercised end to end — the log's rename, open and write —
// without Python or a model.
func stubbedLauncher(t *testing.T, script string) *ExecLauncher {
	t.Helper()
	paths := config.NewPaths(t.TempDir())
	if err := os.MkdirAll(filepath.Dir(paths.VenvPython()), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.VenvPython(), []byte("#!/bin/sh\n"+script+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(paths.Logs, 0o755); err != nil {
		t.Fatal(err)
	}
	return &ExecLauncher{Paths: paths, LogDir: paths.Logs}
}

// plainConfig is the config.json of a model that ships no code of its own.
const plainConfig = `{"model_type":"llama"}`

// plainModelDir is a model directory the launcher will start: this account's
// own, holding a config.json that names no model_file.
func plainModelDir(t *testing.T) string {
	t.Helper()
	return modelDirWithConfig(t, plainConfig)
}

// launchAndWait launches spec and waits for the stub to exit.
func launchAndWait(t *testing.T, l *ExecLauncher, spec Spec) {
	t.Helper()
	if spec.ModelPath == "" {
		spec.ModelPath = plainModelDir(t)
	}
	p, err := l.Launch(context.Background(), spec)
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	<-p.Done()
}

// The restart that ends a debug run must leave Alice the file. Every launch
// opens the per-model log O_TRUNC, so before that open the previous run's log
// is renamed to <name>.previous.log — on every launch, not only an armed one —
// and the content the run wrote survives the launch that follows it.
func TestALaunchKeepsThePreviousRunsLog(t *testing.T) {
	l := stubbedLauncher(t, "echo the new run")
	current := filepath.Join(l.LogDir, logFileName("org/name"))
	previous := filepath.Join(l.LogDir, previousLogFileName("org/name"))
	if err := os.WriteFile(current, []byte("the run Alice armed\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	launchAndWait(t, l, Spec{RepoID: "org/name", Socket: privateSocket(t)})

	if b, err := os.ReadFile(previous); err != nil || string(b) != "the run Alice armed\n" {
		t.Errorf("the previous run's log was not kept under %s: %q, %v", filepath.Base(previous), b, err)
	}
	if b, err := os.ReadFile(current); err != nil || string(b) != "the new run\n" {
		t.Errorf("the new run's log is not a fresh file: %q, %v", b, err)
	}
	if info, err := os.Lstat(previous); err == nil && info.Mode().Perm() != 0o600 {
		t.Errorf("the kept file is mode %04o, want 0600", info.Mode().Perm())
	}
}

// One generation only: the rename replaces any previous file of that name, so
// consecutive launches leave exactly one previous log behind rather than
// accumulating without bound.
func TestTwoConsecutiveLaunchesKeepOnlyOnePreviousLog(t *testing.T) {
	l := stubbedLauncher(t, "echo run $DESSAU_TEST_RUN")
	current := filepath.Join(l.LogDir, logFileName("org/name"))
	previous := filepath.Join(l.LogDir, previousLogFileName("org/name"))
	if err := os.WriteFile(current, []byte("run 0\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("DESSAU_TEST_RUN", "1")
	launchAndWait(t, l, Spec{RepoID: "org/name", Socket: privateSocket(t)})
	t.Setenv("DESSAU_TEST_RUN", "2")
	launchAndWait(t, l, Spec{RepoID: "org/name", Socket: privateSocket(t)})

	if b, _ := os.ReadFile(previous); string(b) != "run 1\n" {
		t.Errorf("the previous log holds %q, want the run before this one", b)
	}
	if b, _ := os.ReadFile(current); string(b) != "run 2\n" {
		t.Errorf("the current log holds %q, want this run", b)
	}
	entries, err := os.ReadDir(l.LogDir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	if len(names) != 2 {
		t.Errorf("the logs folder holds %v, want exactly the current and one previous file", names)
	}
}

// A first launch has no previous file, and that is not an error.
func TestAFirstLaunchHasNoPreviousLogToKeep(t *testing.T) {
	l := stubbedLauncher(t, "exit 0")
	launchAndWait(t, l, Spec{RepoID: "org/name", Socket: privateSocket(t)})
	if _, err := os.Lstat(filepath.Join(l.LogDir, previousLogFileName("org/name"))); err == nil {
		t.Error("a first launch left a previous file behind")
	}
}

// An armed launch writes through the bounded writer and an unarmed one does
// not: the unarmed path hands the file to the child directly, as it always
// has, so nothing changes for anyone who does not arm. The bound is exercised
// through the real Launch, pipe and all, with the bound made small enough for
// a shell stub to reach.
func TestAnArmedLaunchStopsItsLogAtTheBound(t *testing.T) {
	// The stub prints more than the bound: 500 short lines.
	const line = "a line of nineteen\n"
	script := "i=0; while [ $i -lt 500 ]; do echo 'a line of nineteen'; i=$((i+1)); done"

	t.Run("armed", func(t *testing.T) {
		l := stubbedLauncher(t, script)
		l.debugLogMaxBytes = 1000
		launchAndWait(t, l, Spec{RepoID: "org/name", Socket: privateSocket(t), DebugLog: true})
		b, err := os.ReadFile(filepath.Join(l.LogDir, logFileName("org/name")))
		if err != nil {
			t.Fatal(err)
		}
		body, after, ok := strings.Cut(string(b), debugLogBoundLine)
		if !ok {
			t.Fatalf("the armed log has no final line; it holds %d bytes", len(b))
		}
		if len(body) != 1000 {
			t.Errorf("the armed log holds %d bytes before the final line, want the bound of 1000", len(body))
		}
		if after != "" {
			t.Errorf("bytes reached the file after the bound: %q", after)
		}
	})

	t.Run("unarmed", func(t *testing.T) {
		l := stubbedLauncher(t, script)
		l.debugLogMaxBytes = 1000
		launchAndWait(t, l, Spec{RepoID: "org/name", Socket: privateSocket(t)})
		b, err := os.ReadFile(filepath.Join(l.LogDir, logFileName("org/name")))
		if err != nil {
			t.Fatal(err)
		}
		if len(b) != 500*len(line) {
			t.Errorf("the unarmed log holds %d bytes, want all %d the server wrote: the bound applies to an armed run only", len(b), 500*len(line))
		}
		if strings.Contains(string(b), debugLogBoundLine) {
			t.Error("an unarmed run's log carries the bound's final line")
		}
	})
}
