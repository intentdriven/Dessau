package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The probe is off until the operator turns it on; the idle threshold has a
// default that applies to the probe and the self-test alike, and both
// settings survive a round trip under the keys the docs name.
func TestProbeSettingsAreValidated(t *testing.T) {
	d := Default()
	if d.ContextProbe {
		t.Fatal("a fresh install has the context probe on")
	}
	if d.IdleThresholdSec != 0 || d.EffectiveIdleThresholdSec() != DefaultIdleThresholdSec {
		t.Fatalf("a fresh install's idle threshold: stored %d, effective %d, want 0 and %d",
			d.IdleThresholdSec, d.EffectiveIdleThresholdSec(), DefaultIdleThresholdSec)
	}

	path := filepath.Join(t.TempDir(), "config.json")
	c := Default()
	c.ContextProbe = true
	c.IdleThresholdSec = 120
	if err := c.Validate(); err != nil {
		t.Fatalf("a valid pair was refused: %v", err)
	}
	if err := Save(path, c); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	for _, key := range []string{`"context_probe": true`, `"idle_threshold_sec": 120`} {
		if !strings.Contains(string(raw), key) {
			t.Errorf("the file does not carry %s:\n%s", key, raw)
		}
	}
	again, _, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !again.ContextProbe || again.IdleThresholdSec != 120 {
		t.Errorf("read back probe=%v threshold=%d", again.ContextProbe, again.IdleThresholdSec)
	}

	// A save is refused for a threshold this build cannot use.
	for _, sec := range []int{-1, MinIdleThresholdSec - 1, MaxIdleThresholdSec + 1} {
		c := Default()
		c.IdleThresholdSec = sec
		if err := c.Validate(); err == nil {
			t.Errorf("idle_threshold_sec=%d was accepted", sec)
		}
	}
}

// A file carrying a threshold this build cannot use loads with the default
// and says so, the way every other repaired figure does.
func TestAnUnusableIdleThresholdIsRepairedOnLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"host":"127.0.0.1","port":8080,"idle_threshold_sec":7}`), 0o600); err != nil {
		t.Fatal(err)
	}
	c, notices, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if c.IdleThresholdSec != 0 || c.EffectiveIdleThresholdSec() != DefaultIdleThresholdSec {
		t.Errorf("loaded threshold %d (effective %d), want the default", c.IdleThresholdSec, c.EffectiveIdleThresholdSec())
	}
	if !strings.Contains(strings.Join(notices.Repaired, ","), "idle_threshold_sec=7") {
		t.Errorf("the repair was not reported: %v", notices.Repaired)
	}
}
