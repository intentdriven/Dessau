package gateway

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/Dessau/internal/config"
	"github.com/intentdriven/Dessau/internal/selftest"
)

// writeSelfTestRuns puts runs into the results file the way the loop would,
// oldest first, one line each.
func writeSelfTestRuns(t *testing.T, paths config.Paths, runs ...selftest.Run) {
	t.Helper()
	if err := os.MkdirAll(paths.SelfTest, 0o700); err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	for _, run := range runs {
		line, err := json.Marshal(run)
		if err != nil {
			t.Fatal(err)
		}
		b.Write(line)
		b.WriteByte('\n')
	}
	if err := os.WriteFile(filepath.Join(paths.SelfTest, selftest.FileName), []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
}

// A fresh install answers: off, nothing measured — and never a path.
func TestAFreshInstallHasNoSelfTestResults(t *testing.T) {
	srv, _ := newTestControlApp(t, config.Default())
	view, raw := getJSON(t, srv, "/api/selftest")
	if on, _ := view["enabled"].(bool); on {
		t.Error("a fresh install reports the self-test as on")
	}
	if latest, _ := view["latest"].([]any); len(latest) != 0 {
		t.Errorf("latest = %v, want none", latest)
	}
	if strings.Contains(raw, "/") && strings.Contains(raw, "selftest/") {
		t.Errorf("the body names the results location:\n%s", raw)
	}
}

// The endpoint carries the latest run per model, by model, and the recent
// runs newest first; a model measured twice appears once in latest with its
// newer run.
func TestTheSelfTestEndpointCarriesTheLatestRunPerModel(t *testing.T) {
	srv, a := newTestControlApp(t, config.Default())
	writeSelfTestRuns(t, a.Paths,
		selftest.Run{Kind: selftest.KindRun, Model: "org/b", At: 100, Outcome: selftest.OutcomeOK,
			Tests: []selftest.Test{{Name: "tg128", Parallel: 1, TokensPerSec: 40}}},
		selftest.Run{Kind: selftest.KindRun, Model: "org/a", At: 200, Outcome: selftest.OutcomeYielded},
		selftest.Run{Kind: selftest.KindRun, Model: "Org/B", At: 300, Outcome: selftest.OutcomeOK,
			Tests: []selftest.Test{{Name: "tg128", Parallel: 1, TokensPerSec: 42}}},
	)
	resp, err := srv.Client().Get(srv.URL + "/api/selftest")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var view SelfTestView
	if err := json.NewDecoder(resp.Body).Decode(&view); err != nil {
		t.Fatal(err)
	}
	if len(view.Latest) != 2 || view.Latest[0].Model != "org/a" || view.Latest[1].Model != "Org/B" || view.Latest[1].At != 300 {
		t.Fatalf("latest = %+v; want org/a then org/b's run at 300", view.Latest)
	}
	if len(view.Recent) != 3 || view.Recent[0].At != 300 || view.Recent[2].At != 100 {
		t.Fatalf("recent = %+v; want newest first", view.Recent)
	}
	if view.Latest[1].Tests[0].TokensPerSec != 42 {
		t.Errorf("the newer run's figure did not win: %+v", view.Latest[1])
	}
}

// The switch is reported live, from the loop, not from the file.
func TestTheSelfTestEndpointReportsTheSwitch(t *testing.T) {
	on := config.Default()
	on.SelfTest = true
	srv, _ := newTestControlApp(t, on)
	view, _ := getJSON(t, srv, "/api/selftest")
	if enabled, _ := view["enabled"].(bool); !enabled {
		t.Error("the switch is on and the endpoint says off")
	}
}

// Recent is bounded: a long file does not become a long response.
func TestTheSelfTestEndpointBoundsTheRecentRuns(t *testing.T) {
	runs := make([]selftest.Run, 0, selfTestRecent+10)
	for i := 0; i < selfTestRecent+10; i++ {
		runs = append(runs, selftest.Run{Kind: selftest.KindRun, Model: "org/a", At: int64(i), Outcome: selftest.OutcomeOK})
	}
	view := selfTestView(true, runs)
	if len(view.Recent) != selfTestRecent || view.Recent[0].At != int64(selfTestRecent+9) {
		t.Errorf("recent has %d runs, first at %d; want %d newest first", len(view.Recent), view.Recent[0].At, selfTestRecent)
	}
	if len(view.Latest) != 1 || view.Latest[0].At != int64(selfTestRecent+9) {
		t.Errorf("latest = %+v", view.Latest)
	}
}

// The one branch that handles a path-bearing error answers with a constant
// and keeps the path for the log: a directory planted where the file should
// be is a 500 whose body does not say where the file is.
func TestASelfTestReadThatFailsNamesNoPath(t *testing.T) {
	srv, a := newTestControlApp(t, config.Default())
	planted := filepath.Join(a.Paths.SelfTest, selftest.FileName)
	if err := os.MkdirAll(planted, 0o700); err != nil {
		t.Fatal(err)
	}
	resp, err := srv.Client().Get(srv.URL + "/api/selftest")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status %d, want 500; body %s", resp.StatusCode, body)
	}
	if strings.Contains(string(body), a.Paths.SelfTest) || strings.Contains(string(body), "/") {
		t.Errorf("the body names a path:\n%s", body)
	}
}

// The file is parsed once per change, not once per ask: the same size and
// modification time answer from the last reading, and a new line is seen.
func TestTheSelfTestViewIsReadOncePerChange(t *testing.T) {
	srv, a := newTestControlApp(t, config.Default())
	writeSelfTestRuns(t, a.Paths, selftest.Run{Kind: selftest.KindRun, Model: "org/a", At: 1, Outcome: selftest.OutcomeOK})
	first, _ := getJSON(t, srv, "/api/selftest")
	if latest, _ := first["latest"].([]any); len(latest) != 1 {
		t.Fatalf("latest = %v", latest)
	}
	// The same bytes again: served from the cache, which the test can only
	// see by the reading not failing when the file is made unreadable.
	path := filepath.Join(a.Paths.SelfTest, selftest.FileName)
	if err := os.Chmod(path, 0o000); err != nil {
		t.Fatal(err)
	}
	again, _ := getJSON(t, srv, "/api/selftest")
	if latest, _ := again["latest"].([]any); len(latest) != 1 {
		t.Errorf("an unchanged file was read again rather than served from the last reading")
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	// A change is seen: a second line, and a later modification time.
	time.Sleep(20 * time.Millisecond)
	writeSelfTestRuns(t, a.Paths,
		selftest.Run{Kind: selftest.KindRun, Model: "org/a", At: 1, Outcome: selftest.OutcomeOK},
		selftest.Run{Kind: selftest.KindRun, Model: "org/b", At: 2, Outcome: selftest.OutcomeOK})
	changed, _ := getJSON(t, srv, "/api/selftest")
	if latest, _ := changed["latest"].([]any); len(latest) != 2 {
		t.Errorf("a changed file was not read again: latest = %v", latest)
	}
}
