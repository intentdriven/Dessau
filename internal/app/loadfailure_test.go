package app

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/Dessau/internal/config"
	"github.com/intentdriven/Dessau/internal/registry"
	"github.com/intentdriven/Dessau/internal/runtime"
	"github.com/intentdriven/Dessau/internal/stats"
)

// The pool's verdict on a load that never became ready is written onto the
// model with the provenance in force, so it outlives the process and every
// surface reads one record; a load another path interrupted — a client
// that hung up, an unload, an eviction — is not the model's failure and
// leaves no mark; a load that succeeds lifts one (iss-2609211334570516).
func TestALoadFailureIsRecordedWithItsProvenanceAndAnInterruptedLoadIsNot(t *testing.T) {
	a := newTestApp(t)
	readyModel(t, a, "org/m", 131072)
	rec := stats.New(stats.Options{})
	obs := poolObserver{rec: rec, log: slog.New(slog.DiscardHandler), failed: a.recordLoadFailure}
	obs.LoadFinished("org/m", time.Second, &runtime.NotReadyError{
		Err:    errors.New("org/m could not load: ValueError: Model type glm_ocr not supported."),
		Reason: "could not load: ValueError: Model type glm_ocr not supported.",
	}, config.Sampling{})
	m, _ := a.Registry.Get("org/m")
	if !m.LoadFailed() {
		t.Fatal("a load that failed left no mark on the model")
	}
	prov := (probeSources{a}).Provenance("org/m")
	if got := m.LoadFailure; got.Reason != "could not load: ValueError: Model type glm_ocr not supported." ||
		got.Runtime != prov.Runtime || got.BudgetBytes != prov.BudgetBytes ||
		got.DecodeConcurrency != prov.DecodeConcurrency || got.ServedContext != prov.ServedContext || got.At == 0 {
		t.Errorf("LoadFailure = %+v, want the reason under the provenance %+v", got, prov)
	}
	obs.LoadFinished("org/m", time.Second, nil, config.Sampling{})
	if m, _ := a.Registry.Get("org/m"); m.LoadFailed() {
		t.Error("a load that succeeded left the mark standing")
	}
	obs.LoadFinished("org/m", time.Second, &runtime.NotReadyError{
		Err:    errors.New("model server for org/m exited during startup: signal: terminated"),
		Reason: "the model server exited during startup: signal: terminated", Interrupted: true,
	}, config.Sampling{})
	if m, _ := a.Registry.Get("org/m"); m.LoadFailed() {
		t.Error("a load another path interrupted was recorded as the model's failure")
	}
	// A reason that would not pass the registry's bound is still recorded,
	// cut to it, rather than lost: the mark is what stops the retries.
	obs.LoadFinished("org/m", time.Second, &runtime.NotReadyError{
		Err: errors.New("x"), Reason: strings.Repeat("y", 2*registry.MaxLoadFailureReasonBytes),
	}, config.Sampling{})
	if m, _ := a.Registry.Get("org/m"); !m.LoadFailed() || len(m.LoadFailure.Reason) != registry.MaxLoadFailureReasonBytes {
		t.Errorf("an overlong reason was not recorded at the bound: %+v", m.LoadFailure)
	}
}

// While a failure stands, idle work leaves the model alone — it is not a
// probe candidate and not the self-test's pick — and a request for it is
// refused with the recorded reason at once; a hand retry or a moved
// provenance makes it eligible again.
func TestAFailedModelIsSkippedByIdleWorkAndRefusedWithItsReason(t *testing.T) {
	a := newTestApp(t)
	readyModel(t, a, "org/m", 131072)
	readyModel(t, a, "org/other", 131072)
	prov := (probeSources{a}).Provenance("org/m")
	fail := func() {
		t.Helper()
		if err := a.Registry.SetLoadFailure("org/m", &registry.LoadFailure{
			Reason: "could not load: ValueError: Model type glm_ocr not supported.", At: 1,
			Runtime: prov.Runtime, BudgetBytes: prov.BudgetBytes, DecodeConcurrency: prov.DecodeConcurrency, ServedContext: prov.ServedContext,
		}); err != nil {
			t.Fatal(err)
		}
	}
	fail()
	var cands []string
	for _, c := range (probeSources{a}).Candidates() {
		cands = append(cands, c.RepoID)
	}
	if len(cands) != 1 || cands[0] != "org/other" {
		t.Errorf("probe candidates = %v, want the failed model left out", cands)
	}
	if ready := (selfTestServer{a}).Ready(); len(ready) != 1 || ready[0] != "org/other" {
		t.Errorf("the self-test's ready list = %v, want the failed model left out", ready)
	}
	if due := a.Probe.Due([]string{"org/m", "org/other"}, time.Now()); due == "org/m" {
		t.Error("the probe made the failed model due")
	}
	// A request for it: refused at once, with the reason, as a not-ready
	// refusal — the class the gateway tells an entitled client the text of.
	started := time.Now()
	_, _, err := a.Pool.Acquire(context.Background(), "org/m")
	var notReady *runtime.NotReadyError
	if !errors.As(err, &notReady) {
		t.Fatalf("Acquire = %T %v, want a NotReadyError carrying the recorded reason", err, err)
	}
	if !strings.Contains(err.Error(), "ValueError: Model type glm_ocr not supported") || !strings.Contains(err.Error(), "Measure now") {
		t.Errorf("the refusal does not carry the reason and the way out: %q", err)
	}
	if time.Since(started) > time.Second {
		t.Errorf("the refusal took %s, want at once", time.Since(started))
	}
	// Measure now is the hand retry: the mark goes, the model is queued.
	if err := a.MeasureNow("org/m"); err != nil {
		t.Fatal(err)
	}
	if m, _ := a.Registry.Get("org/m"); m.LoadFailed() {
		t.Error("Measure now left the mark standing")
	}
	if due := a.Probe.Due([]string{"org/m", "org/other"}, time.Now()); due != "org/m" {
		t.Errorf("after Measure now the probe's due model = %q, want org/m", due)
	}
	// So is Load from the panel.
	fail()
	a.ForgetLoadFailure("org/m")
	if m, _ := a.Registry.Get("org/m"); m.LoadFailed() {
		t.Error("a hand load left the mark standing")
	}
	// And a save that moves the model's served window lifts it, through the
	// same re-judging every measurement gets.
	fail()
	c := a.Config()
	c.Models = map[string]config.ModelSettings{"org/m": {ServedContext: 32768}}
	if err := a.SetConfig(c); err != nil {
		t.Fatal(err)
	}
	if m, _ := a.Registry.Get("org/m"); m.LoadFailed() {
		t.Error("a moved served window left the failure standing")
	}
}
