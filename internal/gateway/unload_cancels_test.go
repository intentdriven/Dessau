package gateway

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// An operator's Unload of a model the context probe is measuring cancels
// that measurement: the model is not loaded back at the next idle tick, it
// leaves the queue, and it reads as an incomplete probe the operator can
// start again with Measure now (maintainer's decision, 2026-10-03;
// iss-2610031818057157).
func TestUnloadCancelsTheMeasurementItInterrupts(t *testing.T) {
	a, _ := probeStackWith(t, 20_000, 3*time.Second)
	ctrl := &Control{App: a}
	mux := http.NewServeMux()
	ctrl.Routes(mux)
	panel := httptest.NewServer(mux)
	t.Cleanup(panel.Close)

	if err := a.MeasureNow("org/m"); err != nil {
		t.Fatal(err)
	}
	inFlight := func() bool {
		res := a.Pool.Residency().Models
		return a.SelfTest.Status().Job != "" && len(res) == 1 && res[0].InFlight > 0
	}
	deadline := time.Now().Add(20 * time.Second)
	for !inFlight() && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if !inFlight() {
		t.Fatalf("the probe never held the model with a request in flight")
	}
	resp, err := http.Post(panel.URL+"/api/models/unload", "application/json", strings.NewReader(`{"model":"org/m"}`))
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Unload got %d: %s", resp.StatusCode, raw)
	}
	// Many idle ticks on this stack's cadence: a measurement that resumed
	// would have loaded the model back by now.
	time.Sleep(500 * time.Millisecond)
	if res := a.Pool.Residency(); len(res.Models) != 0 {
		t.Errorf("the measurement resumed after Unload: %+v", res.Models)
	}
	if q := a.Probe.Queued(); len(q) != 0 {
		t.Errorf("the measurement is still queued after Unload: %v", q)
	}
	if m, _ := a.Registry.Get("org/m"); !m.ProbeIncomplete {
		t.Error("the cancelled measurement does not read as incomplete")
	}
}
