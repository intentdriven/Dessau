package runtime

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// kernelBootSession is the oracle the ledger's stamp is checked against, read
// independently of bootSessionUUID.
func kernelBootSession() (string, error) {
	b, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
	return strings.TrimSpace(string(b)), err
}

// A process names itself, so its name can carry spaces and parentheses that
// would shift every field after it. The start time is counted from the last
// closing parenthesis, or a process could choose the number the reaper
// compares before a SIGKILL.
func TestStatStartTimeIsReadPastAHostileName(t *testing.T) {
	const tail = " S 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 4242 99 100"
	for _, name := range []string{"(sleep)", "(a b) 7 7 7)", "()))"} {
		got, ok := parseStatStartNs("1234 " + name + tail)
		if !ok || got != 4242*(1_000_000_000/userHz) {
			t.Errorf("name %q: start = %d, %v; want 4242 ticks in ns", name, got, ok)
		}
	}
	for _, bad := range []string{"", "1234 sleep S 1 2", "1234 (sleep) S 1 2 3", "1234 (sleep)" + strings.Replace(tail, "4242", "x", 1)} {
		if _, ok := parseStatStartNs(bad); ok {
			t.Errorf("parseStatStartNs(%q) answered; a line it cannot read must read as unknown", bad)
		}
	}
}

// The start time is the identity the reaper checks before a SIGKILL, so it has
// to be a stable, real reading for a live process and no reading at all for a
// gone one.
func TestProcessStartTimeIsStableAndGoneIsUnknown(t *testing.T) {
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid
	first, ok := processStartNs(pid)
	if !ok || first <= 0 {
		t.Fatalf("processStartNs(%d) = %d, %v; want a positive start time", pid, first, ok)
	}
	time.Sleep(20 * time.Millisecond)
	if again, _ := processStartNs(pid); again != first {
		t.Errorf("start time moved from %d to %d for one live process", first, again)
	}
	cmd.Process.Kill()
	cmd.Wait()
	if _, ok := processStartNs(pid); ok {
		t.Errorf("processStartNs(%d) answered for a process that has exited", pid)
	}
}
