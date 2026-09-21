package gateway

import (
	"net/http"
	"testing"
	"time"

	"github.com/intentdriven/Dessau/internal/config"
	"github.com/intentdriven/Dessau/internal/mlxtest"
	"github.com/intentdriven/Dessau/internal/registry"
)

// The models list says, per entry, whether a conversation with that model is
// recorded (itd-2609091715089488): true only while the machine-wide switch
// is on AND the model is not excepted. It is in the base entry every client
// reads, so the two field-set tests in gateway_test.go name it; these hold
// its value.

// transcriptGateway is residencyGateway's sibling for the transcript state:
// two ready models, the exception set as the configuration given says, and
// the machine-wide switch as the test says.
func transcriptGateway(t *testing.T, cfg config.Config, on bool) http.Handler {
	t.Helper()
	fake := mlxtest.Start(mlxtest.Options{ModelArg: "/m"})
	t.Cleanup(fake.Close)
	added := time.Unix(1757145600, 0)
	models := &stubModels{models: []registry.Model{
		{RepoID: "org/warm", State: registry.StateReady, Path: "/models/org/warm", AddedAt: added},
		{RepoID: "org/cold", State: registry.StateReady, Path: "/models/org/cold", AddedAt: added},
	}}
	g := New(Options{Config: cfg, Pool: &stubPool{srv: fake}, Models: models,
		TranscriptOn: func() bool { return on }})
	return g.Handler()
}

// An exception set under one spelling of a repo id bites under the
// registry's: the listing folds the join, on the pattern of
// TestListModelsJoinsResidencyWhateverTheSpelling, so nothing hangs on the
// operator typing the id the way the registry spells it.
func TestListModelsSaysAnExceptedModelIsNotRecordedWhateverTheSpelling(t *testing.T) {
	cfg := config.Default()
	cfg.Models = map[string]config.ModelSettings{"ORG/WARM": {NoTranscript: true}}
	entries, _ := listModelsEntriesFrom(t, transcriptGateway(t, cfg, true), "", "203.0.113.50:9999")

	if got := entryByID(t, entries, "org/warm")["recording"]; got != false {
		t.Errorf("recording = %v for a model excepted under another spelling, want false", got)
	}
	if got := entryByID(t, entries, "org/cold")["recording"]; got != true {
		t.Errorf("recording = %v for a recorded model while the switch is on, want true", got)
	}
}

// With the switch off nothing is recorded, excepted or not: the field says
// what is true of this model on this server, not what the exception alone
// would say.
func TestListModelsSaysNothingIsRecordedWhileTheSwitchIsOff(t *testing.T) {
	cfg := config.Default()
	cfg.Models = map[string]config.ModelSettings{"org/warm": {NoTranscript: true}}
	entries, _ := listModelsEntriesFrom(t, transcriptGateway(t, cfg, false), "", "203.0.113.50:9999")
	for _, id := range []string{"org/warm", "org/cold"} {
		if got := entryByID(t, entries, id)["recording"]; got != false {
			t.Errorf("recording = %v for %s while the switch is off, want false", got, id)
		}
	}
}
