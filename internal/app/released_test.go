package app

import (
	"log/slog"
	"strings"
	"testing"

	"github.com/intentdriven/Dessau/internal/runtime"
	"github.com/intentdriven/Dessau/internal/stats"
)

// A model a program released is one sparse line saying so, with the kind of
// caller that asked and a paired client by name, and every reason the pool
// can give has a statistics reason (itd-2610031024247803).
func TestAReleaseIsLoggedWithTheKindOfCaller(t *testing.T) {
	log, _, buf := levelledLog(slog.LevelInfo)
	obs := poolObserver{rec: stats.New(stats.Options{}), log: log}

	obs.EntryReleased("org/repo", runtime.Caller{Kind: stats.CallerAPIKey})
	obs.EntryReleased("org/repo", runtime.Caller{Kind: stats.CallerPairedClient, Client: "Bob's iPad", Fingerprint: "ab12cd34"})

	got := buf.String()
	for _, want := range []string{"model unloaded", "reason=released", "by=api_key", "by=paired_client", `client="Bob's iPad"`, "fingerprint=ab12cd34"} {
		if !strings.Contains(got, want) {
			t.Errorf("the sparse log is missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "statistics do not know") {
		t.Errorf("a release was logged as an unknown reason:\n%s", got)
	}
	if _, ok := stopReasons[runtime.StopReleased]; !ok {
		t.Error("StopReleased has no statistics reason")
	}
}
