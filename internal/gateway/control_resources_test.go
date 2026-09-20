package gateway

import (
	"testing"

	"github.com/intentdriven/Dessau/internal/capability"
	"github.com/intentdriven/Dessau/internal/config"
)

// The roll-up's free-disk figure is the capability package's one reader's,
// on the snapshot the panel already polls, so the Models tab and the search
// tab cannot give two answers to how much room the downloads have.
func TestTheSnapshotCarriesFreeDiskFromTheOneReader(t *testing.T) {
	srv, a := newTestControlApp(t, config.Default())
	state, _ := getJSON(t, srv, "/api/state")
	machine, _ := state["machine"].(map[string]any)
	got, _ := machine["free_disk"].(float64)
	want := capability.Assess(a.Paths.Models, a.MachineRAM(), a.Pool.MemoryBudget()).FreeDisk
	if want <= 0 {
		t.Skip("the models volume reports no free space to measure against")
	}
	// Free space moves between two readings; a tenth is far more than it
	// moves in a test.
	if got <= 0 || got < float64(want)*0.9 || got > float64(want)*1.1 {
		t.Errorf("free_disk = %v, want about %d from capability.Assess", got, want)
	}
}
