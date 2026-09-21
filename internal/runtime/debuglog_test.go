package runtime

import (
	"context"
	"reflect"
	"testing"
)

// Arming debug logging for a model that is running changes nothing about the
// run going on now: the level is a launch flag, and a running model server has
// no level switch. So no signal, no restart, nothing written to the child, no
// new argument — the mark waits for the next launch, and the panel says so.
func TestArmingTouchesNoRunningProcess(t *testing.T) {
	l := newFakeLauncher()
	src := &fakeSource{models: map[string]int64{"org/m": 1 << 20}}
	p := newTestPool(t, l, src, PoolOptions{MaxResidentBytes: 1 << 30})

	_, release, err := p.Acquire(context.Background(), "org/m")
	if err != nil {
		t.Fatal(err)
	}
	release()
	proc := l.procFor("org/m")
	before := l.specFor("org/m")
	seen := l.serverFor("org/m").Completions()

	if err := p.ArmDebugLog("org/m"); err != nil {
		t.Fatalf("ArmDebugLog: %v", err)
	}

	if l.launchCount() != 1 {
		t.Errorf("arming launched a process: %d launches, want 1", l.launchCount())
	}
	if l.procFor("org/m") != proc {
		t.Error("arming replaced the running process")
	}
	select {
	case <-proc.stopped:
		t.Error("arming stopped the running process")
	default:
	}
	if got := l.serverFor("org/m").Completions(); got != seen {
		t.Errorf("arming sent the running process %d request(s)", got-seen)
	}
	if got := l.specFor("org/m"); !reflect.DeepEqual(got, before) || got.DebugLog {
		t.Errorf("arming changed the running process's spec: %+v, want %+v", got, before)
	}
	// The mark is visible as armed, and the running process is not at debug.
	if got := p.DebugArmed(); !reflect.DeepEqual(got, []string{"org/m"}) {
		t.Errorf("DebugArmed = %v, want [org/m]", got)
	}
	for _, r := range p.Resident() {
		if r.DebugLog {
			t.Errorf("the running process %s reports itself at debug, but it was launched at INFO", r.RepoID)
		}
	}
}

// The next launch of an armed model carries --log-level DEBUG exactly once,
// and the mark is spent in the same step, so the launch after that is back at
// INFO with nothing to remember.
func TestAnArmedLaunchIsAtDebugAndTheNextIsAtInfo(t *testing.T) {
	l := newFakeLauncher()
	src := &fakeSource{models: map[string]int64{"org/m": 1 << 20}}
	p := newTestPool(t, l, src, PoolOptions{MaxResidentBytes: 1 << 30})

	load := func() {
		t.Helper()
		_, release, err := p.Acquire(context.Background(), "org/m")
		if err != nil {
			t.Fatal(err)
		}
		release()
	}
	unload := func() {
		t.Helper()
		if err := p.Unload("org/m"); err != nil {
			t.Fatal(err)
		}
	}

	load()
	if err := p.ArmDebugLog("org/m"); err != nil {
		t.Fatal(err)
	}
	unload()
	load() // the armed run
	unload()
	load() // the run after

	specs := l.specHistory()
	if len(specs) != 3 {
		t.Fatalf("%d launches, want 3", len(specs))
	}
	for i, want := range []string{"INFO", "DEBUG", "INFO"} {
		argv := launchArgs(specs[i])
		got, _ := flagValue(argv, "--log-level")
		if got != want {
			t.Errorf("launch %d is at %q, want %s: %v", i+1, got, want, argv)
		}
		if n := countArg(argv, "--log-level"); n != 1 {
			t.Errorf("launch %d names --log-level %d times, want once: %v", i+1, n, argv)
		}
	}
	if got := p.DebugArmed(); len(got) != 0 {
		t.Errorf("the mark survived the launch that consumed it: DebugArmed = %v", got)
	}
}

// The running process reports the level it was launched at — a fact about the
// process, not about the mark — so the panel can tell an armed model from one
// whose current run is the logged one.
func TestTheResidentEntrySaysWhetherItsRunIsAtDebug(t *testing.T) {
	l := newFakeLauncher()
	src := &fakeSource{models: map[string]int64{"org/m": 1 << 20}}
	p := newTestPool(t, l, src, PoolOptions{MaxResidentBytes: 1 << 30})

	if err := p.ArmDebugLog("org/m"); err != nil {
		t.Fatal(err)
	}
	_, release, err := p.Acquire(context.Background(), "org/m")
	if err != nil {
		t.Fatal(err)
	}
	release()

	res := p.Resident()
	if len(res) != 1 || !res[0].DebugLog {
		t.Fatalf("Resident = %+v, want org/m with DebugLog true", res)
	}
	if got := p.DebugArmed(); len(got) != 0 {
		t.Errorf("a model whose run is at debug is still reported as armed: %v", got)
	}
}

// A launch that fails to spawn leaves the mark armed rather than spent: the
// run Alice wanted never started, so telling her it did would be a lie in
// both directions — armed once and then not, or armed twice.
func TestALaunchThatFailsToSpawnLeavesTheMarkArmed(t *testing.T) {
	l := newFakeLauncher()
	l.failFor = "org/m"
	src := &fakeSource{models: map[string]int64{"org/m": 1 << 20}}
	p := newTestPool(t, l, src, PoolOptions{MaxResidentBytes: 1 << 30})

	if err := p.ArmDebugLog("org/m"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := p.Acquire(context.Background(), "org/m"); err == nil {
		t.Fatal("the launch that was made to fail succeeded")
	}
	if got := p.DebugArmed(); !reflect.DeepEqual(got, []string{"org/m"}) {
		t.Fatalf("a failed launch spent the mark: DebugArmed = %v, want [org/m]", got)
	}

	// Once the launch can succeed, the run it starts is the armed one.
	l.mu.Lock()
	l.failFor = ""
	l.mu.Unlock()
	_, release, err := p.Acquire(context.Background(), "org/m")
	if err != nil {
		t.Fatal(err)
	}
	release()
	if !l.specFor("org/m").DebugLog {
		t.Error("the launch after the failed one is not at debug")
	}
	if got := p.DebugArmed(); len(got) != 0 {
		t.Errorf("the mark survived the launch that consumed it: %v", got)
	}
}

// Disarm is the inverse of arm: a mark set by mistake can be taken back before
// it is spent, and the mark is matched however the id is spelled, like every
// other per-model lookup in the pool.
func TestDisarmTakesTheMarkBackWhateverTheSpelling(t *testing.T) {
	l := newFakeLauncher()
	src := &fakeSource{models: map[string]int64{"org/m": 1 << 20}}
	p := newTestPool(t, l, src, PoolOptions{MaxResidentBytes: 1 << 30})

	if err := p.ArmDebugLog("org/m"); err != nil {
		t.Fatal(err)
	}
	if err := p.DisarmDebugLog("ORG/M"); err != nil {
		t.Fatal(err)
	}
	if got := p.DebugArmed(); len(got) != 0 {
		t.Fatalf("DebugArmed = %v after disarm, want none", got)
	}
	_, release, err := p.Acquire(context.Background(), "org/m")
	if err != nil {
		t.Fatal(err)
	}
	release()
	if l.specFor("org/m").DebugLog {
		t.Error("a disarmed model was launched at debug")
	}
}
