package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The self-test has an idle threshold of its own (iss-2610041956214381):
// four hours unless set, held to the same bounds as the context probe's, and
// carried in config.json under its own key. Setting one threshold moves
// nothing about the other.
func TestTheSelfTestHasAnIdleThresholdOfItsOwn(t *testing.T) {
	if DefaultSelfTestIdleThresholdSec != 14400 {
		t.Fatalf("DefaultSelfTestIdleThresholdSec = %d, want 14400 (4 hours)", DefaultSelfTestIdleThresholdSec)
	}
	d := Default()
	if d.SelfTestIdleThresholdSec != 0 || d.EffectiveSelfTestIdleThresholdSec() != DefaultSelfTestIdleThresholdSec {
		t.Fatalf("a fresh install's self-test threshold: stored %d, effective %d, want 0 and %d",
			d.SelfTestIdleThresholdSec, d.EffectiveSelfTestIdleThresholdSec(), DefaultSelfTestIdleThresholdSec)
	}
	if got := d.EffectiveSelfTestIdleThreshold().Hours(); got != 4 {
		t.Errorf("a fresh install's self-test threshold is %v hours, want 4", got)
	}

	// Independent of the context probe's, both ways.
	c := Default()
	c.IdleThresholdSec = 120
	if c.EffectiveSelfTestIdleThresholdSec() != DefaultSelfTestIdleThresholdSec {
		t.Errorf("setting idle_threshold_sec moved the self-test's threshold to %d", c.EffectiveSelfTestIdleThresholdSec())
	}
	c = Default()
	c.SelfTestIdleThresholdSec = 7200
	if c.EffectiveIdleThresholdSec() != DefaultIdleThresholdSec {
		t.Errorf("setting self_test_idle_threshold_sec moved the context probe's threshold to %d", c.EffectiveIdleThresholdSec())
	}

	path := filepath.Join(t.TempDir(), "config.json")
	if err := Save(path, c); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	if !strings.Contains(string(raw), `"self_test_idle_threshold_sec": 7200`) {
		t.Errorf("the file does not carry the self-test's threshold under its key:\n%s", raw)
	}
	again, notices, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if again.SelfTestIdleThresholdSec != 7200 || len(notices.Repaired) != 0 {
		t.Errorf("read back %d with repairs %v", again.SelfTestIdleThresholdSec, notices.Repaired)
	}

	// The same bounds as the context probe's, and zero for the default.
	for _, sec := range []int{0, MinIdleThresholdSec, 14400, MaxIdleThresholdSec} {
		c := Default()
		c.SelfTestIdleThresholdSec = sec
		if err := c.Validate(); err != nil {
			t.Errorf("self_test_idle_threshold_sec=%d was refused: %v", sec, err)
		}
	}
	for _, sec := range []int{-1, MinIdleThresholdSec - 1, MaxIdleThresholdSec + 1} {
		c := Default()
		c.SelfTestIdleThresholdSec = sec
		err := c.Validate()
		if err == nil {
			t.Errorf("self_test_idle_threshold_sec=%d was accepted", sec)
			continue
		}
		if !strings.Contains(err.Error(), "self_test_idle_threshold_sec") {
			t.Errorf("self_test_idle_threshold_sec=%d was refused in words that do not name it: %v", sec, err)
		}
	}
}

// A file carrying a self-test threshold this build cannot use loads with the
// default and says so, rather than refusing to start or leaving a value a
// later save would be refused over.
func TestAnUnusableSelfTestIdleThresholdIsRepairedOnLoad(t *testing.T) {
	for _, stored := range []string{"7", "86401"} {
		path := filepath.Join(t.TempDir(), "config.json")
		body := `{"host":"127.0.0.1","port":8080,"self_test_idle_threshold_sec":` + stored + `}`
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		c, notices, err := Load(path)
		if err != nil {
			t.Fatal(err)
		}
		if c.SelfTestIdleThresholdSec != 0 || c.EffectiveSelfTestIdleThresholdSec() != DefaultSelfTestIdleThresholdSec {
			t.Errorf("loaded %s as %d (effective %d), want the default", stored,
				c.SelfTestIdleThresholdSec, c.EffectiveSelfTestIdleThresholdSec())
		}
		if !strings.Contains(strings.Join(notices.Repaired, ","), "self_test_idle_threshold_sec="+stored) {
			t.Errorf("the repair of %s was not reported: %v", stored, notices.Repaired)
		}
		if err := c.Validate(); err != nil {
			t.Errorf("the repaired configuration is refused: %v", err)
		}
	}
}
