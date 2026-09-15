package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The self-test is off until the operator turns it on, and a configuration
// written before the field existed loads with it off: the same state as
// today, and the only one a build that did not know the field could mean.
func TestTheSelfTestIsOffByDefaultAndSurvivesARoundTrip(t *testing.T) {
	if Default().SelfTest {
		t.Fatal("a fresh install has the self-test on")
	}
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"host":"127.0.0.1","port":8080}`), 0o600); err != nil {
		t.Fatal(err)
	}
	c, _, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if c.SelfTest {
		t.Error("a configuration that never named the self-test loads with it on")
	}

	c.SelfTest = true
	if err := Save(path, c); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	if !strings.Contains(string(raw), `"self_test": true`) {
		t.Errorf("the switch is saved under some other key:\n%s", raw)
	}
	again, _, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !again.SelfTest {
		t.Error("the switch did not survive a save and a load")
	}
}
