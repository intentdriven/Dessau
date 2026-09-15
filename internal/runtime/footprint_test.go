package runtime

import (
	"os/exec"
	"testing"
)

// The one production reader: a live process reports a footprint, a dead
// one reports none, and top's figure parses with its suffixes.
func TestExecProcessReportsAFootprintWhileAliveAndNoneAfter(t *testing.T) {
	cmd := exec.Command("/bin/sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	p := &execProcess{cmd: cmd, done: make(chan struct{})}
	if got := p.Footprint(); got <= 0 {
		t.Errorf("a live process reports %d", got)
	}
	_ = cmd.Process.Kill()
	_ = cmd.Wait()
	close(p.done)
	if got := p.Footprint(); got != 0 {
		t.Errorf("a dead process reports %d", got)
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
