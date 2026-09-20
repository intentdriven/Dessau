package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/Dessau/internal/config"
	"github.com/intentdriven/Dessau/internal/registry"
)

// The self-test follows its switch and nothing else: off on a fresh install
// with nothing on disk, on the moment a save turns it on, off again the
// moment a save turns it off, and gone when the app closes.
func TestTheSelfTestFollowsItsSwitch(t *testing.T) {
	a, paths := statsApp(t, config.Default())
	if a.SelfTest.Enabled() {
		t.Fatal("a fresh install has the self-test running")
	}
	if _, err := os.Stat(paths.SelfTest); !os.IsNotExist(err) {
		t.Fatalf("a fresh install has a self-test directory (%v)", err)
	}

	on := config.Default()
	on.SelfTest = true
	if err := a.SetConfig(on); err != nil {
		t.Fatal(err)
	}
	if !a.SelfTest.Enabled() {
		t.Error("saving the switch on did not start the self-test")
	}
	if err := a.SetConfig(config.Default()); err != nil {
		t.Fatal(err)
	}
	if a.SelfTest.Enabled() {
		t.Error("saving the switch off did not stop the self-test")
	}

	if err := a.SetConfig(on); err != nil {
		t.Fatal(err)
	}
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	if a.SelfTest.Enabled() {
		t.Error("closing the app left the self-test running")
	}
}

// A configuration file with the switch on starts the loop at construction,
// the way every other switch in the file is in force from the start.
func TestASavedSwitchStartsTheSelfTestAtConstruction(t *testing.T) {
	c := config.Default()
	c.SelfTest = true
	a, paths := statsApp(t, c)
	if !a.SelfTest.Enabled() {
		t.Error("the saved switch did not start the self-test")
	}
	if want := filepath.Join(paths.Account, "selftest"); paths.SelfTest != want {
		t.Errorf("results live at %q, want %q: beside the statistics store, under this account", paths.SelfTest, want)
	}
}

// The two switches sit together on the Settings pane and are not one switch:
// turning request statistics on starts no self-test, and turning the
// self-test on records no statistics.
func TestTheSelfTestAndStatisticsSwitchesAreIndependent(t *testing.T) {
	a, _ := statsApp(t, config.Default())
	stats := config.Default()
	stats.Statistics = true
	if err := a.SetConfig(stats); err != nil {
		t.Fatal(err)
	}
	if a.SelfTest.Enabled() {
		t.Error("turning statistics on started the self-test")
	}
	self := config.Default()
	self.SelfTest = true
	if err := a.SetConfig(self); err != nil {
		t.Fatal(err)
	}
	if a.Stats.Enabled() {
		t.Error("turning the self-test on started statistics recording")
	}
	if !a.SelfTest.Enabled() {
		t.Error("the self-test switch did not start the self-test")
	}
}

// Fits is the self-test's promise never to evict, answered from what the pool
// holds and what the model would be charged: a model this Mac cannot hold
// beside what is loaded does not fit, an unknown model does not fit, and a
// small ready one does.
func TestFitsAnswersFromTheChargeAndTheBudget(t *testing.T) {
	a := newTestApp(t)
	for _, m := range []registry.Model{
		{RepoID: "org/small", Path: a.Paths.ModelDir("org/small"), State: registry.StateReady, Bytes: 1 << 20},
		{RepoID: "org/vast", Path: a.Paths.ModelDir("org/vast"), State: registry.StateReady, Bytes: 1 << 50},
		{RepoID: "org/unsized", Path: a.Paths.ModelDir("org/unsized"), State: registry.StateReady},
	} {
		if err := a.Registry.Put(m); err != nil {
			t.Fatal(err)
		}
	}
	srv := selfTestServer{a}
	if !srv.Fits("org/small") {
		t.Error("a small model with nothing loaded does not fit")
	}
	if srv.Fits("org/vast") {
		t.Error("a model larger than any Mac fits")
	}
	if srv.Fits("org/unsized") {
		t.Error("a model with no size fits; its charge cannot be known")
	}
	if srv.Fits("org/absent") {
		t.Error("a model the registry does not have fits")
	}
}
