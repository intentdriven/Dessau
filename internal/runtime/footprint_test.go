package runtime

import (
	"context"
	"os/exec"
	"testing"
	"time"
)

const (
	// liveReadBudget is how long the live test goes on asking for a reading
	// before it gives up on this machine. It is not a bound on the reader:
	// one reading is bounded by Footprint itself.
	liveReadBudget = 20 * time.Second
	// hangBound is what tells a reading that is merely slow from one that is
	// never coming. It is deliberately far above anything the reader's own
	// bounds add up to, because the thing being caught is unbounded and the
	// thing being tolerated — a runnable goroutine waiting its turn on a Mac
	// running a dozen suites at once — is seconds wide and not measurable
	// from in here. Its whole job is to turn a park into a line that names
	// the wait, instead of the package's timeout and a goroutine dump.
	hangBound = time.Minute
	// pipeBound is the same idea for a held output pipe, where the broken
	// behaviour is the 30s the grandchild sleeps and the correct one is about
	// a second.
	pipeBound = 15 * time.Second
)

// The live half of the one production reader, and the only part of it that
// needs the real process listing.
//
// A reading is the thing a busy Mac cannot always supply: top takes longer
// than footprintTimeout allows and Footprint answers zero, which is its
// documented answer to a listing that did not arrive rather than a wrong
// figure about a live process (iss-2609190040226948). So the reading is asked
// for again within a budget, and a machine that produced none in that time
// says so and skips — never a false green, and never a failure blamed on the
// reader for the load on the machine.
func TestExecProcessReportsAFootprintWhileAlive(t *testing.T) {
	p := sleeper(t)
	var got int64
	tries := 0
	for deadline := time.Now().Add(liveReadBudget); ; {
		tries++
		got = readWithin(t, p)
		if got > 0 || !time.Now().Before(deadline) {
			break
		}
	}
	if got <= 0 {
		t.Skipf("no reading of a live process in %d tries over %s: every listing took longer than the %s Footprint allows one, so this Mac is too busy to take the sample — the live half is unproven, not disproven",
			tries, liveReadBudget, footprintTimeout)
	}
}

// The dead half, which needs nothing of the machine: a process that is done is
// not read at all, because its pid may already be somebody else's.
func TestExecProcessReportsNoFootprintOnceTheProcessIsDone(t *testing.T) {
	p := sleeper(t)
	_ = p.cmd.Process.Kill()
	_ = p.cmd.Wait()
	close(p.done)
	if got := readWithin(t, p); got != 0 {
		t.Errorf("a dead process reports %d", got)
	}
}

// A listing that leaves its output pipe held open is abandoned rather than
// waited on. Killing the listing does not close the pipe — a grandchild here,
// a listing the kernel has not finished with on a Mac under memory pressure —
// and without the second bound the caller waits for whatever holds it. The
// pool's close waits on this reader (iss-2609190135528563 saw the wait as a
// package-wide timeout, not a failure).
func TestAProcessSampleDoesNotWaitOnAnOutputPipeItCannotClose(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = sampleProcess(ctx, "/bin/sh", "-c", "sleep 30 & sleep 30")
	}()
	// Generous against a loaded scheduler and still nowhere near the 30s the
	// held pipe would cost.
	select {
	case <-done:
	case <-time.After(pipeBound):
		t.Fatalf("the sample had not returned %s after a context that fired in 200ms, so it is waiting on the output pipe of a listing it already gave up on", pipeBound)
	}
}

// sleeper is a live process wrapped the way the launcher wraps one, with its
// own cleanup so no /bin/sleep outlives the test.
func sleeper(t *testing.T) *execProcess {
	t.Helper()
	cmd := exec.Command("/bin/sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})
	return &execProcess{cmd: cmd, done: make(chan struct{})}
}

// readWithin takes one reading off this goroutine and holds it to Footprint's
// own bound. A reader that does not answer inside its bound ends the test with
// a line naming the wait, rather than parking until the package's timeout
// prints a goroutine dump somebody has to read (iss-2609190135528563).
func readWithin(t *testing.T, p *execProcess) int64 {
	t.Helper()
	res := make(chan int64, 1)
	go func() { res <- p.Footprint() }()
	select {
	case got := <-res:
		return got
	case <-time.After(hangBound):
		t.Fatalf("Footprint did not answer within %s, though it bounds one listing at %s and the wait for its output at %s: it is parked, not slow",
			hangBound, footprintTimeout, footprintWaitDelay)
		return 0
	}
}

func TestParseTopMem(t *testing.T) {
	for in, want := range map[string]int64{
		"\nMEM  \n1728K\n": 1728 << 10, "MEM\n2560M+": 2560 << 20, "MEM\n3G-": 3 << 30, "MEM\n512B": 512, "": 0, "MEM\n": 0, "x\nnonsense": 0,
	} {
		if got := parseTopMem(in); got != want {
			t.Errorf("parseTopMem(%q) = %d, want %d", in, got, want)
		}
	}
}
