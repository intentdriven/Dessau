package selftest

import (
	"path/filepath"
	"testing"
	"time"
)

// The self-test and the jobs wait for thresholds of their own
// (iss-2610041956214381). The self-test loads and benchmarks a model, which an
// ordinary pause in use should not set off, so it waits for SelfTestQuiet —
// four hours unless set — while a job such as the context probe keeps Quiet,
// the context probe's idle_threshold_sec.

// tenMinutesQuiet is a server whose last client request ended ten minutes
// ago: past a five-minute threshold, well short of an hour's.
func tenMinutesQuiet(t *testing.T) *fakeServer {
	t.Helper()
	srv := newFakeServer(t, "org/a")
	srv.mu.Lock()
	srv.lastRequest = time.Now().Add(-10 * time.Minute)
	srv.mu.Unlock()
	return srv
}

func TestTheSelfTestWaitsForItsOwnThreshold(t *testing.T) {
	srv := tenMinutesQuiet(t)
	r := New(Options{
		Server: srv, Path: filepath.Join(t.TempDir(), FileName),
		Tick: 5 * time.Millisecond, Poll: 2 * time.Millisecond,
		Quiet: 5 * time.Minute, SelfTestQuiet: time.Hour,
	})
	t.Cleanup(r.Close)
	r.SetEnabled(true)
	waitFor(t, "the self-test to be held by the request ten minutes ago", func() bool {
		return r.Status().HeldBy == HeldByRecent
	})
	if acquired, _, _ := srv.snapshot(); len(acquired) != 0 {
		t.Errorf("the self-test loaded %v ten minutes after a request, with its own threshold an hour", acquired)
	}

	// Its own threshold moved, live, and the next tick runs.
	r.SetSelfTestQuiet(time.Minute)
	waitFor(t, "the self-test to run once its own threshold is a minute", func() bool { return len(runsIn(t, r)) == 1 })
}

func TestAJobWaitsForItsThresholdNotTheSelfTests(t *testing.T) {
	srv := tenMinutesQuiet(t)
	job := &gatewayJob{srv: srv}
	r := New(Options{
		Server: srv, Path: filepath.Join(t.TempDir(), FileName),
		Tick: 5 * time.Millisecond, Poll: 2 * time.Millisecond,
		Quiet: 5 * time.Minute, SelfTestQuiet: time.Hour,
		Jobs: []Job{job}, SelfTest: func() bool { return false },
	})
	t.Cleanup(r.Close)
	r.SetEnabled(true)
	waitFor(t, "the job to run on its own five-minute threshold despite the self-test's hour", func() bool {
		return job.runs.Load() >= 1
	})
}

func TestTheSelfTestRunsOnItsOwnThresholdWhenTheJobsIsLonger(t *testing.T) {
	srv := tenMinutesQuiet(t)
	r := New(Options{
		Server: srv, Path: filepath.Join(t.TempDir(), FileName),
		Tick: 5 * time.Millisecond, Poll: 2 * time.Millisecond,
		Quiet: time.Hour, SelfTestQuiet: 5 * time.Minute,
	})
	t.Cleanup(r.Close)
	r.SetEnabled(true)
	waitFor(t, "the self-test to run on its own five-minute threshold despite the jobs' hour", func() bool {
		return len(runsIn(t, r)) == 1
	})
}

// Left unset, the self-test waits four hours and a job five minutes: the
// figures config.DefaultSelfTestIdleThresholdSec and
// config.DefaultIdleThresholdSec stand for.
func TestTheThresholdsDefaultToFourHoursAndFiveMinutes(t *testing.T) {
	if DefaultSelfTestQuiet != 4*time.Hour {
		t.Errorf("DefaultSelfTestQuiet = %v, want 4h", DefaultSelfTestQuiet)
	}
	r := New(Options{Server: newFakeServer(t), Path: filepath.Join(t.TempDir(), FileName)})
	job, self := r.Thresholds()
	if job != DefaultQuiet || self != DefaultSelfTestQuiet {
		t.Errorf("thresholds = %v for jobs, %v for the self-test; want %v and %v", job, self, DefaultQuiet, DefaultSelfTestQuiet)
	}
	r.SetQuiet(2 * time.Minute)
	r.SetSelfTestQuiet(3 * time.Hour)
	if job, self := r.Thresholds(); job != 2*time.Minute || self != 3*time.Hour {
		t.Errorf("after setting each: %v for jobs, %v for the self-test; want 2m and 3h", job, self)
	}
}
