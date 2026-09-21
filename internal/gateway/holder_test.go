package gateway

import (
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/Dessau/internal/config"
	"github.com/intentdriven/Dessau/internal/mlxtest"
	"github.com/intentdriven/Dessau/internal/registry"
	"github.com/intentdriven/Dessau/internal/runtime"
	"github.com/intentdriven/Dessau/internal/selftest"
)

// holdingGateway is a keyless install whose pool refuses every load for want
// of room, with the idle loop reporting the run in progress and the pool
// reporting what is resident.
func holdingGateway(t *testing.T, key string, status selftest.Status, resident []runtime.Resident) http.Handler {
	t.Helper()
	fake := mlxtest.Start(mlxtest.Options{ModelArg: "/m"})
	t.Cleanup(fake.Close)
	cfg := config.Default()
	cfg.APIKey = key
	return New(Options{
		Config: cfg,
		Pool:   &stubPool{srv: fake, acquireErr: &runtime.NoRoomError{Limit: 41 << 30}, resident: resident},
		Models: &stubModels{models: []registry.Model{
			{RepoID: "org/warm", State: registry.StateReady, Path: "/models/org/warm"},
			{RepoID: "org/ocr", State: registry.StateReady, Path: "/models/org/ocr"},
		}},
		Log:      slog.New(slog.DiscardHandler),
		IdleJobs: func() selftest.Status { return status },
	}).Handler()
}

// When the memory a load needs is held by a model an idle job is holding,
// the refusal says so to a client this server owes an account of itself:
// which model, which job, and for how long — it is the server's own work,
// not the size of the model asked for. The refusal for a budget that is
// genuinely full is the sentence it always was, and an unentitled client is
// told nothing either way (iss-2609211334576018).
func TestARefusalNamesTheModelAnIdleJobIsHolding(t *testing.T) {
	since := time.Now().Add(-4*time.Minute - 12*time.Second)
	held := selftest.Status{Job: "context-probe", Model: "Org/OCR", Step: "calibrating at 1024 tokens", Since: since}
	resident := []runtime.Resident{{RepoID: "org/ocr", State: runtime.ResidencyLoading, Charge: 41 << 30}}
	plain := "not enough memory to load another model, and no model in memory can be freed (limit 41.0 GB)"

	t.Run("held by the probe, on this machine", func(t *testing.T) {
		w := completionForAs(t, holdingGateway(t, "", held, resident), "org/warm", "127.0.0.1:52001")
		body := w.Body.String()
		if w.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want 503", w.Code)
		}
		for _, want := range []string{plain, "org/ocr", "context probe", "4m", "the server's own idle work"} {
			if !strings.Contains(body, want) {
				t.Errorf("body = %s, want it to carry %q", body, want)
			}
		}
	})
	t.Run("held by the self-test, keyed on the network", func(t *testing.T) {
		st := selftest.Status{Job: "self-test", Model: "org/ocr", Since: since}
		w := completionAs(t, holdingGateway(t, "bh_secret", st, resident), "203.0.113.50:9999", "bh_secret")
		if body := w.Body.String(); !strings.Contains(body, "org/ocr") || !strings.Contains(body, "self-test") {
			t.Errorf("body = %s, want the holder and the job", body)
		}
	})
	t.Run("a genuinely full budget is the sentence it was", func(t *testing.T) {
		w := completionForAs(t, holdingGateway(t, "", selftest.Status{}, resident), "org/warm", "127.0.0.1:52001")
		if body := w.Body.String(); !strings.Contains(body, plain) || strings.Contains(body, "org/ocr") || strings.Contains(body, "idle") {
			t.Errorf("body = %s, want only the plain refusal", body)
		}
	})
	t.Run("a job whose model is not in memory names nothing", func(t *testing.T) {
		w := completionForAs(t, holdingGateway(t, "", held, nil), "org/warm", "127.0.0.1:52001")
		if body := w.Body.String(); strings.Contains(body, "org/ocr") {
			t.Errorf("body = %s, names a model that holds no memory", body)
		}
	})
	t.Run("an unentitled client is told nothing", func(t *testing.T) {
		w := completionForAs(t, holdingGateway(t, "", held, resident), "org/warm", "203.0.113.50:9999")
		if body := strings.TrimSpace(w.Body.String()); strings.Contains(body, "org/ocr") || strings.Contains(body, "probe") || !strings.Contains(body, genericRefusal) {
			t.Errorf("body = %s, want the generic refusal alone", body)
		}
	})
}
