package gateway

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/intentdriven/Dessau/internal/config"
	"github.com/intentdriven/Dessau/internal/runtime"
	"github.com/intentdriven/Dessau/internal/stats"
)

// The operator's switch turns the route off for every caller, and says so
// only to a caller already entitled to it: one that is not gets the refusal
// it always got, and learns nothing about the setting (itd-2610031024247803
// criterion 7).
func TestTheSwitchOffRefusesTheRoute(t *testing.T) {
	cfg := config.Default()
	cfg.APIUnloadOff = true
	g, pool := unloadGatewayWith(t, cfg)

	code, body := serveUnload(g.Handler(), unloadRequest(thisMac, `{"model":"org/warm"}`, nil))
	if code != http.StatusForbidden || !strings.Contains(body, "turned off") {
		t.Errorf("a program on this Mac, switch off: status %d, %s", code, body)
	}
	code, body = serveUnload(g.Handler(), unloadRequest(aLANHost, `{"model":"org/warm"}`, nil))
	if code != http.StatusForbidden || strings.Contains(body, "turned off") {
		t.Errorf("a caller not entitled learned of the switch: status %d, %s", code, body)
	}
	if n := len(pool.releasedIDs()); n != 0 {
		t.Errorf("%d model(s) unloaded with the route turned off", n)
	}
}

// An absent key means on: a configuration that never mentions the switch
// serves the route.
func TestAnAbsentSwitchMeansOn(t *testing.T) {
	cfg := config.Default()
	if cfg.APIUnloadOff {
		t.Fatal("the default configuration turns the route off")
	}
	g, _ := unloadGatewayWith(t, cfg)
	if code, body := serveUnload(g.Handler(), unloadRequest(thisMac, `{"model":"org/warm"}`, nil)); code != http.StatusOK {
		t.Errorf("status %d, %s", code, body)
	}
}

// The control panel's Unload is the operator's own and the switch does not
// reach it: with the route off, the panel still unloads (criterion 7).
func TestTheSwitchOffLeavesThePanelsUnloadWorking(t *testing.T) {
	cfg := config.Default()
	cfg.APIUnloadOff = true
	a, _, mux := newWiredControl(t, cfg, "org/m")
	srv := serve(t, mux)
	loadModel(t, a, "org/m")

	resp, err := srv.Client().Post(srv.URL+"/api/models/unload", "application/json", strings.NewReader(`{"model":"org/m"}`))
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("the panel's unload with the API route off: %d %s", resp.StatusCode, b)
	}
	if got := a.Pool.Resident(); len(got) != 0 {
		t.Errorf("Resident() = %+v after the panel's unload, want none", got)
	}
}

// The pool is told which kind of caller released a model, and a paired client
// by the name it was paired under — never the key (criterion 8).
func TestTheReleaseSaysWhichKindOfCallerAsked(t *testing.T) {
	g, pool := unloadGateway(t, "")
	serveUnload(g.Handler(), unloadRequest(thisMac, `{"model":"org/warm"}`, nil))

	gk, poolK := unloadGateway(t, "bh_secret")
	serveUnload(gk.Handler(), unloadRequest(aLANHost, `{"model":"org/warm"}`,
		map[string]string{"Authorization": "Bearer bh_secret"}))
	serveUnload(gk.Handler(), unloadRequest(thisMac, `{"model":"org/warm"}`, nil))

	if got := pool.releasedBy(); len(got) != 1 || got[0] != (runtime.Caller{Kind: stats.CallerThisMac}) {
		t.Errorf("a program on this Mac: %v", got)
	}
	want := []runtime.Caller{{Kind: stats.CallerAPIKey}, {Kind: stats.CallerThisMac}}
	got := poolK.releasedBy()
	if len(got) != len(want) {
		t.Fatalf("keyed install: %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("keyed install, release %d: %+v, want %+v", i, got[i], want[i])
		}
		if strings.Contains(got[i].Kind+got[i].Client, "bh_secret") {
			t.Errorf("the caller carries the key: %+v", got[i])
		}
	}

	// A paired client, through the TLS listener's own admission.
	gp, poolP := unloadGateway(t, "bh_secret")
	var srv *pairedServer
	srv = newPairedServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gp.TLSHandler(srv.registry).ServeHTTP(w, r)
	}))
	cert := srv.pair(t, newClientKey(t), "Bob's iPad")
	req, _ := http.NewRequest(http.MethodPost, "https://"+srv.tlsAddr+"/v1/dessau/unload", strings.NewReader(`{"model":"org/warm"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := srv.client(cert).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("a paired client: %d %s", resp.StatusCode, b)
	}
	if got := poolP.releasedBy(); len(got) != 1 || got[0] != (runtime.Caller{Kind: stats.CallerPairedClient, Client: "Bob's iPad"}) {
		t.Errorf("a paired client: %+v", got)
	}
}
