package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Update checks are off unless the operator turns them on, daily once on.
func TestUpdateChecksAreOffByDefaultAndDailyWhenOn(t *testing.T) {
	c := Default()
	if c.UpdateCheck {
		t.Error("update checks are on in the default configuration")
	}
	if got := c.EffectiveUpdateCheckInterval(); got != 24*time.Hour {
		t.Errorf("interval = %v, want daily by default", got)
	}
	c.UpdateCheckIntervalHours = 720
	if got := c.EffectiveUpdateCheckInterval(); got != 30*24*time.Hour {
		t.Errorf("interval = %v, want thirty days", got)
	}
}

// The interval is one hour to thirty days; the settings path refuses
// anything else by name.
func TestTheUpdateCheckIntervalIsBounded(t *testing.T) {
	for _, h := range []int{0, 1, 24, 720} {
		c := Default()
		c.UpdateCheckIntervalHours = h
		if err := c.Validate(); err != nil {
			t.Errorf("%d hours refused: %v", h, err)
		}
	}
	for _, h := range []int{-1, 721, 100000} {
		c := Default()
		c.UpdateCheckIntervalHours = h
		err := c.Validate()
		if err == nil || !strings.Contains(err.Error(), "update_check_interval_hours") {
			t.Errorf("%d hours: err = %v, want a refusal naming the setting", h, err)
		}
	}
}

// An out-of-range interval written into config.json by hand is repaired on
// load and named, so the file loads, the default is in force, and a save of
// any other setting is never refused over it.
func TestAnOutOfRangeIntervalInTheFileIsRepairedOnLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"host":"127.0.0.1","port":11535,"decode_concurrency":1,"update_check_enabled":true,"update_check_interval_hours":99999}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, n, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.UpdateCheck {
		t.Error("the switch was lost with the interval")
	}
	if cfg.UpdateCheckIntervalHours != 0 {
		t.Errorf("interval = %d, want repaired to the default", cfg.UpdateCheckIntervalHours)
	}
	found := false
	for _, r := range n.Repaired {
		found = found || strings.HasPrefix(r, "update_check_interval_hours=")
	}
	if !found {
		t.Errorf("the repair is not named: %v", n.Repaired)
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("the repaired configuration does not validate: %v", err)
	}
}
