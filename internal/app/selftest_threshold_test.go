package app

import (
	"testing"
	"time"

	"github.com/intentdriven/Dessau/internal/config"
)

// The idle loop is handed two thresholds, one per setting
// (iss-2610041956214381): the self-test's own, self_test_idle_threshold_sec,
// four hours unless set, and the context probe's, idle_threshold_sec. Both are
// in force from the start and follow every save; a save that names neither
// moves neither.
func TestTheSelfTestAndTheProbeWaitForTheirOwnThresholds(t *testing.T) {
	a := newTestApp(t)
	job, self := a.SelfTest.Thresholds()
	if job != config.DefaultIdleThresholdSec*time.Second || self != config.DefaultSelfTestIdleThresholdSec*time.Second {
		t.Errorf("at defaults the loop waits %v for the probe and %v for the self-test; want %ds and %ds",
			job, self, config.DefaultIdleThresholdSec, config.DefaultSelfTestIdleThresholdSec)
	}

	c := config.Default()
	c.IdleThresholdSec = 120
	c.SelfTestIdleThresholdSec = 7200
	if err := a.SetConfig(c); err != nil {
		t.Fatal(err)
	}
	if job, self := a.SelfTest.Thresholds(); job != 120*time.Second || self != 7200*time.Second {
		t.Errorf("after a save the loop waits %v for the probe and %v for the self-test; want 2m0s and 2h0m0s", job, self)
	}

	c = a.Config()
	c.LogLevel = config.LogLevelDetailed
	if err := a.SetConfig(c); err != nil {
		t.Fatal(err)
	}
	if got := a.Config(); got.IdleThresholdSec != 120 || got.SelfTestIdleThresholdSec != 7200 {
		t.Errorf("a save of the log level moved the thresholds: probe %d, self-test %d",
			got.IdleThresholdSec, got.SelfTestIdleThresholdSec)
	}
}

// A file carrying the self-test's threshold puts it in force at construction.
func TestASavedSelfTestThresholdIsInForceFromTheStart(t *testing.T) {
	c := config.Default()
	c.SelfTestIdleThresholdSec = config.MaxIdleThresholdSec
	a, _ := statsApp(t, c)
	if _, self := a.SelfTest.Thresholds(); self != config.MaxIdleThresholdSec*time.Second {
		t.Errorf("the self-test waits %v, want the saved %ds", self, config.MaxIdleThresholdSec)
	}
}
