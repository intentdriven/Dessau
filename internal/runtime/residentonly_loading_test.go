package runtime

import (
	"context"
	"errors"
	"testing"
	"time"
)

// A resident-only caller that finds the model mid-load — another caller is
// loading it back — is refused at once, as for a model the pool does not
// hold, rather than waiting on that load for up to ReadyTimeout
// (iss-2609202010357676).
func TestAResidentOnlyAcquireDoesNotWaitOnAnotherCallersLoad(t *testing.T) {
	l := newFakeLauncher()
	l.loadDelayFor["org/a"] = 2 * time.Second
	p := newTestPool(t, l, graceModels(), PoolOptions{MaxResidentBytes: graceBudget})

	go func() {
		if _, release, err := p.Acquire(context.Background(), "org/a"); err == nil {
			release()
		}
	}()
	deadline := time.Now().Add(5 * time.Second)
	for len(p.Residency().Models) == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if m := p.Residency().Models; len(m) != 1 || m[0].State != ResidencyLoading {
		t.Fatalf("residency = %+v, want org/a loading before the resident-only acquire", m)
	}

	start := time.Now()
	_, release, err := p.Acquire(WithResidentOnly(context.Background()), "org/a")
	if release != nil {
		defer release()
	}
	if took := time.Since(start); took > 500*time.Millisecond {
		t.Errorf("the resident-only acquire waited %v on another caller's load", took)
	}
	if !errors.Is(err, ErrNotResident) {
		t.Errorf("err = %v, want ErrNotResident for a model still loading", err)
	}
}
