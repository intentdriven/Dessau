package gateway

import (
	"testing"
	"time"

	"github.com/intentdriven/Dessau/internal/config"
	"github.com/intentdriven/Dessau/internal/runtime"
)

// One save that pins the model the probe is measuring and switches the probe
// off leaves the model in memory: the pin is in force before the switch stops
// the run, so the stop's own unload meets it (iss-2609211754251373, the
// review of the probe-pins fix).
func TestASaveThatPinsAndSwitchesTheProbeOffKeepsTheModel(t *testing.T) {
	a, _ := probeStackWith(t, 20_000, 300*time.Millisecond)
	on := a.Config()
	on.ContextProbe = true
	if err := a.SetConfig(on); err != nil {
		t.Fatal(err)
	}
	// Loaded and answering the probe's step: a model still loading when the
	// run is cancelled is a load its only caller gave up, which the pool
	// drops whatever the pin says, and is not what is held here.
	answering := func() bool {
		res := a.Pool.Residency().Models
		return a.SelfTest.Status().Job != "" && len(res) == 1 &&
			res[0].State == runtime.ResidencyLoaded && res[0].InFlight > 0
	}
	deadline := time.Now().Add(20 * time.Second)
	for !answering() && time.Now().Before(deadline) {
		time.Sleep(2 * time.Millisecond)
	}
	if !answering() {
		t.Fatal("the probe never had the model loaded and answering")
	}
	c := a.Config()
	c.ContextProbe = false
	c.Models = map[string]config.ModelSettings{"org/m": {Pinned: true}}
	if err := a.SetConfig(c); err != nil {
		t.Fatal(err)
	}
	deadline = time.Now().Add(10 * time.Second)
	for a.SelfTest.Status().Job != "" && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if res := a.Pool.Residency(); len(res.Models) != 1 {
		t.Errorf("the model pinned in the save that stopped the probe was unloaded: %+v", res.Models)
	}
}
