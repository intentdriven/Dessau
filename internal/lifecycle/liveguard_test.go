package lifecycle

import (
	"path/filepath"
	"strings"
	"testing"
)

// The live environments refuse to be built inside a test binary, which is the
// backstop under every seam in this package: a test that reaches one of these
// by accident would quit an application, raise an authorisation panel, download
// a runtime and replace a bundle on the machine running the suite
// (iss-2609111240578491).
func TestTheLiveEnvironmentsRefuseToBeBuiltInATest(t *testing.T) {
	env, _, _ := testEnv()

	if _, err := liveInstallEnv(env); err == nil {
		t.Error("liveInstallEnv built the world a real install acts on, inside a test")
	} else if !strings.Contains(err.Error(), "inside a test") {
		t.Errorf("the refusal does not say why: %v", err)
	}
	if _, err := liveUninstallEnv(env); err == nil {
		t.Error("liveUninstallEnv built the world a real uninstall acts on, inside a test")
	}
	// Update is the one where reaching the live path in a test would download
	// a release and replace the application the suite is running from.
	if _, err := liveUpdateEnv(env); err == nil {
		t.Error("liveUpdateEnv built the world a real update acts on, inside a test")
	} else if !strings.Contains(err.Error(), "runUpdate") {
		t.Errorf("the refusal does not name the seam a test should be using instead: %v", err)
	}
}

// And the one way past it is deliberate, named, and lasts for one test.
//
// Opening it also closes the one probe that touches the Mac: the live install
// and update environments choose the destination by asking whether this
// account can write /Applications, which creates and removes a file there —
// and a test binary doing that raced the installer tripwire in
// internal/archtest (iss-2609120444017291). With the guard open, the answer is
// "no" without asking, so the destination is this account's own, on every Mac.
func TestTheGuardCanBeOpenedForOneTest(t *testing.T) {
	env, _, _ := testEnv()

	t.Run("opened", func(t *testing.T) {
		allowLiveEnvInTest(t)
		ie, err := liveInstallEnv(env)
		if err != nil {
			t.Fatalf("the guard was opened and still refused: %v", err)
		}
		if ie.Dest != filepath.Join(ie.Home, "Applications", bundleName) {
			t.Errorf("Dest = %q; an opened guard probed the Mac for its destination", ie.Dest)
		}
		if destinationWritable() {
			t.Error("the destination probe still answers for the Mac while the guard is open")
		}
	})

	// The subtest above has ended, so the guard is back.
	if _, err := liveInstallEnv(env); err == nil {
		t.Error("the guard stayed open after the test that opened it ended")
	}
}
