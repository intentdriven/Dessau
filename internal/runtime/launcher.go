package runtime

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/intentdriven/Dessau/internal/config"
)

// Spec describes one model server process to launch.
type Spec struct {
	RepoID string
	// ModelPath is the directory handed to `mlx_lm.server --model`. It is also
	// the exact string clients must put in the request's "model" field, which is
	// why the pool hands it back to the gateway to rewrite with.
	ModelPath string
	Port      int
	// DecodeConcurrency maps to --decode-concurrency: how many requests are
	// batched together during generation.
	DecodeConcurrency int
	// Sampling is the set of sampling defaults this server starts with. The
	// server applies them to any request that omits the parameter, and a
	// request's own value replaces them for that request alone — which is why
	// they belong on the command line rather than in the relayed body.
	Sampling config.Sampling
	// DebugLog says this one launch runs at the model server's debug level, at
	// which it writes every request body and every response to its log —
	// prompts and completions, whoever sent them. It is derived from the
	// pool's per-model mark and from nothing else: not from the statistics
	// switch, not from log_level. The pool spends the mark at the launch that
	// carries it, so the launch after this one is back at INFO with nothing to
	// remember (adr-2609201008477513).
	DebugLog bool
}

// DebugLogMaxBytes bounds what one armed run of a model server writes to its
// log. At DEBUG the server writes every request body and every response, so
// the bytes in the file are chosen by whoever is sending requests; the bound
// is what keeps a looping client from filling the disk. It is large enough to
// hold thousands of ordinary request-and-answer pairs and many times the
// largest single body the server ever sees — the context probe's generated
// filler — which is why the bound holds per write rather than per file. With
// the previous run's file kept beside the current one, one model's worst case
// on disk is two of these.
const DebugLogMaxBytes = 64 << 20

// The model server's debug level, spelled here and nowhere else. Its one use
// is inside launchArgs, in the block that reads Spec.DebugLog;
// internal/archtest holds it to that.
const debugLogLevel = "DEBUG"

// samplingFlagsVerifiedAgainst is the mlx-lm release whose source the flag
// spellings below, and the ranges in internal/config, were read from. Nothing
// else connects them to it: a renamed flag makes argparse exit on every model
// launch, which no test with a stand-in interpreter can see. A version bump
// therefore has to come past
// TestSamplingFlagsWereVerifiedAgainstThePinnedServer.
const samplingFlagsVerifiedAgainst = "0.31.3"

// samplingFlags is the model server's own spelling of each sampling
// parameter's launch flag, read from its argument parser (see
// .abcd/development/research/notes/2026-09-06-mlx-lm-sampling-launch-flags.md).
var samplingFlags = map[string]string{
	"temperature": "--temp",
	"top_p":       "--top-p",
	"top_k":       "--top-k",
	"min_p":       "--min-p",
	"max_tokens":  "--max-tokens",
}

// samplingArgs renders the sampling defaults as command-line flags.
//
// It walks the same table config validates against, so a parameter added
// there without a flag here is caught by a test rather than accepted, stored
// and never applied. Every value is re-rendered from a number, so nothing a
// client sent and nothing a file contained reaches the argument vector as a
// string, and each flag and its value are separate elements — there is no
// shell here to split them.
//
// Out-of-range values are dropped rather than passed. The settings endpoint
// already refuses them, but this is the last point at which one could still
// do harm, and the harm is large: the model server takes the flag as its
// default and checks the effective value of every request against it, and its
// check raises uncaught — the connection closes with no response and the
// gateway answers 502. A value it will not accept therefore breaks every
// request that omits that parameter, precisely the traffic these defaults
// exist to serve.
func samplingArgs(s config.Sampling) []string {
	sane, _ := s.Sanitized()
	var args []string
	for _, v := range sane.Values() {
		flag, ok := samplingFlags[v.Field]
		if !ok {
			continue
		}
		args = append(args, flag, formatSamplingValue(v))
	}
	return args
}

// formatSamplingValue renders one value for the command line.
//
// Negative zero is the one number that passes a "must be at least zero" check
// and still renders with a leading dash, which would read as another flag.
func formatSamplingValue(v config.SamplingValue) string {
	if v.Integer {
		// From the integer field, never from the float64 the bounds are
		// compared in: narrowing a float64 that is out of integer range is
		// implementation-defined, and on one of Go's architectures it wraps to
		// a negative — which argparse would accept as a token budget.
		return strconv.Itoa(v.Int)
	}
	if v.Number == 0 {
		return "0"
	}
	return strconv.FormatFloat(v.Number, 'f', -1, 64)
}

// Process is a running model server.
type Process interface {
	// Stop terminates the process, gracefully if it can, forcefully if it must.
	Stop(ctx context.Context) error
	// Done is closed when the process exits.
	Done() <-chan struct{}
	// Err reports why the process exited, if it failed.
	Err() error
	// Pid is the OS process id, for diagnostics.
	Pid() int
}

// Launcher starts model server processes. The pool is written against this
// interface so it can be tested without Python or a GPU.
type Launcher interface {
	Launch(ctx context.Context, spec Spec) (Process, error)
	// Precheck reports whether Launch is likely to succeed for spec, without
	// starting a process. The pool calls it before evicting another model to
	// make room: an eviction is not reversible, so a launch failure caught
	// only after the victim is gone destroys a healthy, unrelated model for
	// nothing.
	Precheck(spec Spec) error
}

// LaunchError wraps a Launcher.Launch failure — the process failing to start
// at all. Its message can embed absolute local filesystem paths (the venv
// interpreter, the model directory, the log file) rooted under the serving
// account's home directory, so callers that relay pool errors to the network
// must not forward it verbatim — the gateway matches on this type to log the
// detail server-side and return a generic message instead. It does not cover
// a readiness failure once the process has started (Pool.waitReady's
// readyErr): that error currently carries no local-path detail, since it
// comes from the process's own exit status or a readiness-probe timeout, not
// from Launch.
type LaunchError struct {
	Err error
}

func (e *LaunchError) Error() string { return e.Err.Error() }
func (e *LaunchError) Unwrap() error { return e.Err }

// NotReadyError wraps the other half of a load going wrong: the process
// started and then never answered a completion — it exited during startup, or
// it was still loading when the readiness timeout ran out.
//
// It carries the same message it always did; the type is what lets a caller
// tell "could not be started" from "started and never answered". The two are
// different failures: the first is a broken installation or a vanished model
// directory, the second is usually a model too large for this Mac or weights
// that will not load. Its message is safe to relay, unlike a LaunchError's:
// it comes from the process's own exit status, from the probe's timeout, or
// from the terminal line of a traceback in the child's log with anything
// path-shaped stripped out (fatalLoadLine), never from a path on this machine.
type NotReadyError struct {
	Err error
	// Reason is the failure without the model's name — "did not become
	// ready within 10m0s", "could not load: ValueError: …" — for a record
	// kept on the model itself (registry.LoadFailure), where the name is
	// the entry's own.
	Reason string
	// Transient says the verdict is the pool's own bound rather than the
	// child's assertion — the readiness timeout ran out, or the process was
	// ended by a signal — so the same load may well go differently on a
	// quieter machine. A record kept of it does not outlive this process.
	Transient bool
	// Interrupted says the load did not fail on its own: another path took
	// the entry out of the pool while it was loading — a client that hung
	// up, an unload, an eviction — and the process was stopped for it. It
	// is not the model's failure and leaves no record on the model.
	Interrupted bool
}

func (e *NotReadyError) Error() string { return e.Err.Error() }
func (e *NotReadyError) Unwrap() error { return e.Err }

// ExecLauncher runs the real `mlx_lm.server` out of the managed virtualenv.
type ExecLauncher struct {
	Paths config.Paths
	// LogDir receives one log file per model process.
	LogDir string
	// Owner is the uid the interpreter must be owned by (root is always
	// accepted); zero means the current effective uid. See trustedExecutable.
	Owner int

	// debugLogMaxBytes is the bound on an armed run's log; zero means
	// DebugLogMaxBytes. A field so a test can reach the bound with a stub,
	// not a setting: the figure the product ships is the constant.
	debugLogMaxBytes int64

	ledgerOnce sync.Once
	ledger     *pidLedger
}

func (l *ExecLauncher) pidLedger() *pidLedger {
	// This account's own directory, not the data root: the ledger records
	// process groups only the uid that started them can signal, so it is no use
	// to another account — and in a shared root the second account's write over
	// the first account's ledger is refused by the sticky bit and swallowed,
	// which ends orphan reaping for it without a word.
	// Account, falling back to Root for a Paths built by hand without it — the
	// same fallback config.Paths applies to the state directory. With neither,
	// newPIDLedger returns an inert ledger rather than a relative path in
	// whatever directory the process was started from.
	l.ledgerOnce.Do(func() {
		dir := l.Paths.Account
		if dir == "" {
			dir = l.Paths.Root
		}
		l.ledger = newPIDLedger(dir)
	})
	return l.ledger
}

// ReapOrphans kills any model servers left running by a previous, crashed run.
// Call once at startup before launching anything.
func (l *ExecLauncher) ReapOrphans() int {
	return l.pidLedger().reapOrphans()
}

// Precheck confirms the venv interpreter is present and trustworthy (see
// trustedExecutable) and the model directory exists.
func (l *ExecLauncher) Precheck(spec Spec) error {
	python := l.Paths.VenvPython()
	if err := trustedExecutable(python, ownerOrSelf(l.Owner)); err != nil {
		return fmt.Errorf("python runtime is not installed (%s): %w", python, err)
	}
	if _, err := os.Stat(spec.ModelPath); err != nil {
		return fmt.Errorf("model directory is missing (%s): %w", spec.ModelPath, err)
	}
	return nil
}

// launchArgs is the model server's whole command line, spec by spec.
//
// It is a function of the Spec alone, and the one thing the Spec says about
// logging is the per-model debug mark. That is the point. At DEBUG the model
// server writes every request body and every response to its log, prompts and
// completions included, so the level is never something another feature can
// reach — recording request statistics leaves this vector byte for byte as it
// was. Raising it is a separate, deliberate action per model that the panel
// describes in plain words, spent at the one launch that carries it
// (adr-2609201008477513, which narrows adr-2609061503319212 to exactly this).
func launchArgs(spec Spec) []string {
	level := "INFO"
	if spec.DebugLog {
		level = debugLogLevel
	}
	// `python -m mlx_lm.server` is deprecated in 0.31; `python -m mlx_lm server`
	// is the supported spelling.
	args := []string{
		"-m", "mlx_lm", "server",
		"--model", spec.ModelPath,
		// Model servers are strictly loopback. Only the Go gateway faces the LAN,
		// so it alone enforces auth and rewrites requests.
		"--host", "127.0.0.1",
		"--port", strconv.Itoa(spec.Port),
		"--log-level", level,
	}
	args = append(args, samplingArgs(spec.Sampling)...)
	if spec.DecodeConcurrency > 1 {
		args = append(args, "--decode-concurrency", strconv.Itoa(spec.DecodeConcurrency))
	}
	return args
}

// Launch spawns mlx_lm.server for one model.
func (l *ExecLauncher) Launch(ctx context.Context, spec Spec) (Process, error) {
	if err := l.Precheck(spec); err != nil {
		return nil, err
	}
	python := l.Paths.VenvPython()
	cmd := exec.Command(python, launchArgs(spec)...)
	cmd.Env = append(os.Environ(),
		// Without an existing HF_HUB_CACHE directory, mlx_lm.server raises
		// CacheNotFound while serving /v1/models and returns an empty 200.
		"HF_HOME="+filepath.Dir(l.Paths.HFCache),
		"HF_HUB_CACHE="+l.Paths.HFCache,
		// Inference must never reach the network: everything it needs is already
		// in ModelPath, and a stray download would stall a request for minutes.
		"HF_HUB_OFFLINE=1",
		"PYTHONUNBUFFERED=1",
	)
	// Put the child in its own process group so we can signal the whole group;
	// mlx_lm can spawn helpers that would otherwise outlive it.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	logName := logFileName(spec.RepoID)
	logPath := filepath.Join(l.LogDir, logName)
	// EnsureDirs created LogDir at startup as a real directory. Re-check rather
	// than MkdirAll: a path-based MkdirAll would follow a link left under that
	// name and put this account's log file inside a directory it did not choose.
	if fi, err := os.Lstat(l.LogDir); err != nil {
		return nil, fmt.Errorf("log directory %s: %w", l.LogDir, err)
	} else if !fi.IsDir() {
		return nil, fmt.Errorf("log directory %s is not a directory", l.LogDir)
	}
	if err := keepPreviousLog(l.LogDir, spec.RepoID); err != nil {
		return nil, fmt.Errorf("keep previous log %s: %w", logPath, err)
	}
	// 0600, not the 0644 os.Create would give: the model server logs at INFO —
	// request-level detail nobody else has business reading — and at DEBUG,
	// when armed, every prompt and every answer. O_TRUNC keeps the per-model
	// log from growing without bound across restarts; the previous run's file
	// was renamed aside just above, so the truncation destroys nothing.
	//
	// LogDir is this account's own directory (config.Paths.Logs resolves through
	// accountDir), which is what makes the open reachable at all: while the logs
	// sat in the shared root, one account's 0600 log under a name derived from
	// the repo id meant the NEXT account's O_CREATE|O_TRUNC returned EACCES and
	// the model would not start for it.
	//
	// The hardening stays. The name is predictable, and a link or a FIFO left
	// under it — by anything that can write this directory, or by an older
	// install that kept logs elsewhere — would let a truncating open empty, then
	// stream logs into, any file this account can write. O_NOFOLLOW refuses the
	// link; O_NONBLOCK keeps a planted FIFO from blocking the open forever (and
	// is inert on the regular file the fstat below guarantees); the fstat on the
	// opened handle refuses anything else that is not a regular file.
	logFile, err := os.OpenFile(logPath,
		os.O_CREATE|os.O_WRONLY|os.O_TRUNC|syscall.O_NONBLOCK|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		return nil, fmt.Errorf("create log %s: %w", logPath, err)
	}
	if info, err := logFile.Stat(); err != nil || !info.Mode().IsRegular() {
		logFile.Close()
		if err == nil {
			err = fmt.Errorf("not a regular file")
		}
		return nil, fmt.Errorf("create log %s: %w", logPath, err)
	}
	// An unarmed launch hands the file to the child directly, as it always
	// has: no pipe, no goroutine, nothing changed for anyone who does not arm.
	// An armed launch writes every prompt and every answer, so its bytes are
	// chosen by whoever is sending requests, and they pass through the bound.
	// The one value on both streams makes os/exec open one pipe and one
	// copying goroutine, so stdout and stderr stay interleaved as they are
	// with the file.
	if spec.DebugLog {
		max := l.debugLogMaxBytes
		if max <= 0 {
			max = DebugLogMaxBytes
		}
		bounded := newBoundedWriter(logFile, max)
		cmd.Stdout = bounded
		cmd.Stderr = bounded
	} else {
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	}

	if err := cmd.Start(); err != nil {
		logFile.Close()
		return nil, fmt.Errorf("start mlx_lm.server: %w", err)
	}

	// Record the child's process group so a future run can reap it if we crash
	// before Stop runs. Setpgid makes the child lead its own group (pgid == pid).
	pgid := cmd.Process.Pid
	l.pidLedger().add(pgid)

	p := &execProcess{
		cmd:     cmd,
		log:     logFile,
		logPath: logPath,
		done:    make(chan struct{}),
		ledger:  l.pidLedger(),
		pgid:    pgid,
	}
	go func() {
		err := cmd.Wait()
		p.mu.Lock()
		p.err = err
		p.mu.Unlock()
		logFile.Close()
		// The process is gone; drop it from the crash-recovery ledger.
		p.ledger.remove(p.pgid)
		close(p.done)
	}()
	return p, nil
}

// logFileName turns a repo id into a safe filename. The separator must be a
// character ValidRepoID rejects: with one the id itself can contain (an
// underscore, say), "a/b_c" and "a_b/c" would share a file, and launching the
// second model would truncate the first one's live log.
func logFileName(repoID string) string {
	safe := make([]rune, 0, len(repoID))
	for _, r := range repoID {
		if r == '/' || r == ' ' {
			r = '@'
		}
		safe = append(safe, r)
	}
	return string(safe) + ".log"
}

// keepPreviousLog renames the previous run's log aside before the truncating
// open that follows it, on every launch and not only an armed one. The open
// is what would otherwise destroy the run the operator armed: an unattended
// reload overnight would leave an empty file where the evidence was. One
// generation only — the rename replaces what was under the previous name, so
// consecutive launches do not accumulate.
//
// The rename goes through an os.Root on the logs directory, the discipline the
// rest of the tree applies to these files (internal/applog's OpenIn): both
// names are in the one directory, so the rename cannot cross a filesystem or
// the account boundary the shared-cache install creates, and a name that
// leaves the directory is refused rather than followed. A link or a FIFO left
// under the log's name is refused here, before the open would refuse it, so
// that renaming it aside never turns a planted name into a kept one. No
// previous file is normal; any other failure refuses the launch, in the same
// class as the open failing — silently truncating the run the operator armed
// is the failure this exists to prevent.
func keepPreviousLog(dir, repoID string) error {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return err
	}
	defer root.Close()
	name := logFileName(repoID)
	info, err := root.Lstat(name)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", name)
	}
	return root.Rename(name, previousLogFileName(repoID))
}

// previousLogFileName is where the previous run's log is kept: logFileName's
// name with ".previous" before the extension. One generation only — a launch
// renames the current file over this name, so what was here before is gone.
func previousLogFileName(repoID string) string {
	name := logFileName(repoID)
	return strings.TrimSuffix(name, ".log") + ".previous.log"
}

type execProcess struct {
	cmd     *exec.Cmd
	log     *os.File
	logPath string
	done    chan struct{}
	ledger  *pidLedger
	pgid    int

	mu  sync.Mutex
	err error
}

func (p *execProcess) Done() <-chan struct{} { return p.done }

// Footprint reads the server's memory footprint the way Activity Monitor and
// top report it — the physical footprint, which is what the 2026-09-06
// campaign sampled and what a unified-memory Mac actually spends — through
// top itself, since no unprivileged pure-Go reader of that figure exists. A
// launched server runs under this account, so the listing can see it. Zero
// when the process is gone or the listing fails or stalls.
func (p *execProcess) Footprint() int64 {
	if p.cmd == nil || p.cmd.Process == nil {
		return 0
	}
	// A process that has gone is not read: its pid may already be somebody
	// else's, and the listing is not scoped to this account.
	select {
	case <-p.done:
		return 0
	default:
	}
	// Bounded: the sampler runs off the pool's lock but the pool's close
	// waits for it, and a process listing on a Mac under memory pressure can
	// stall. A listing that does not answer in time is no reading.
	ctx, cancel := context.WithTimeout(context.Background(), footprintTimeout)
	defer cancel()
	out, err := sampleProcess(ctx, "/usr/bin/top", "-l", "1", "-stats", "mem", "-pid", strconv.Itoa(p.cmd.Process.Pid))
	if err != nil {
		return 0
	}
	select {
	case <-p.done:
		return 0
	default:
	}
	return parseTopMem(string(out))
}

// sampleProcess runs one process listing under ctx and returns what it
// printed. It is the whole of Footprint's boundedness, and it takes two bounds
// rather than one because a context is only half of it: the context kills the
// listing, but Wait goes on waiting for the listing's output pipe to close,
// and the pipe is not this process's to close. A listing wedged in the kernel
// — which is the state a Mac under memory pressure puts one in, and the state
// this reader exists to survive — does not die when it is signalled, so its
// pipe stays open and the caller stays parked for as long as that lasts. The
// pool's close waits on this reader, so an unbounded wait here is a Dessau
// that will not quit. WaitDelay is the second bound: it gives up on the output
// rather than on the answer, and an abandoned listing is simply no reading.
func sampleProcess(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.WaitDelay = footprintWaitDelay
	return cmd.Output()
}

// parseTopMem reads the last figure top printed: a count with a K, M or G
// suffix, sometimes followed by a + or - that marks a change since the last
// sample. Zero when there is no such line.
func parseTopMem(out string) int64 {
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) == 0 {
		return 0
	}
	field := strings.TrimRight(strings.TrimSpace(lines[len(lines)-1]), "+-")
	if field == "" {
		return 0
	}
	unit := int64(1)
	switch field[len(field)-1] {
	case 'K':
		unit, field = 1<<10, field[:len(field)-1]
	case 'M':
		unit, field = 1<<20, field[:len(field)-1]
	case 'G':
		unit, field = 1<<30, field[:len(field)-1]
	case 'B':
		field = field[:len(field)-1]
	}
	n, err := strconv.ParseInt(field, 10, 64)
	if err != nil || n <= 0 {
		return 0
	}
	return n * unit
}

// footprintTimeout bounds one process listing, and footprintWaitDelay bounds
// the wait for its output once the listing has been given up on.
const (
	footprintTimeout   = 3 * time.Second
	footprintWaitDelay = time.Second
)

func (p *execProcess) Pid() int { return p.cmd.Process.Pid }

func (p *execProcess) Err() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.err
}

// The one bound on stopping a model server, and the two intervals it is made
// of. Every caller that has to know how long a server may take to go reads
// stopBound rather than a figure of its own: the pool's stop context, and the
// wait for the memory to come back, were three different numbers before this,
// none of which was what Stop actually did.
const (
	// stopTermGrace is how long a model server has to honor SIGTERM.
	stopTermGrace = 10 * time.Second
	// stopKillGrace is how long SIGKILL is then given to land.
	stopKillGrace = 5 * time.Second
	// stopBound is the longest Stop can take: after it, either the process is
	// gone or the kernel is not letting go of it.
	stopBound = stopTermGrace + stopKillGrace
)

// Stop asks the process group to exit, escalating to SIGKILL if it will not.
//
// It returns an error only in the case that matters to a caller accounting for
// the process's memory: the group was still there after SIGKILL, so its memory
// is still held.
func (p *execProcess) Stop(ctx context.Context) error {
	select {
	case <-p.done:
		return nil // already gone
	default:
	}

	pgid := -p.cmd.Process.Pid // negative pid signals the whole group
	_ = syscall.Kill(pgid, syscall.SIGTERM)

	deadline := stopTermGrace
	if dl, ok := ctx.Deadline(); ok {
		if d := time.Until(dl); d < deadline {
			deadline = d
		}
	}

	select {
	case <-p.done:
		return nil
	case <-time.After(deadline):
		// A model server wedged mid-generation will not honor SIGTERM. Do not
		// leave it holding gigabytes of GPU memory.
		_ = syscall.Kill(pgid, syscall.SIGKILL)
		select {
		case <-p.done:
			return nil
		case <-time.After(stopKillGrace):
			return errors.New("model server would not die, even after SIGKILL")
		}
	}
}

// LogPath is where this process's output is being written.
func (p *execProcess) LogPath() string { return p.logPath }

// freePort asks the kernel for an unused loopback TCP port.
//
// There is an unavoidable race between closing the listener and the child
// binding the port. It is tolerable here because the ports are loopback-only and
// handed out one at a time, and because a collision surfaces immediately as a
// failed readiness probe rather than as silent corruption.
func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
