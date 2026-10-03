package runtime

import "golang.org/x/sys/unix"

// bootSessionUUID returns this boot's identifier, or "" if it cannot be read.
// It is the session marker that makes a recorded pgid meaningful: pgids are only
// comparable within one boot.
//
// It is deliberately not kern.boottime. That value is defined as walltime minus
// uptime, and XNU adjusts the globals behind it by the correction delta on every
// calendar clock STEP — the first post-boot NTP sync, a re-discipline after
// sleep/wake, a manual clock change — so it moves within a single boot. Measured
// on an Apple Silicon Mac while writing this, kern.boottime moved 80 ms inside
// one uninterrupted boot while this UUID did not change at all. Keyed on the
// clock, both the reap at startup and the carry-forward in add() silently became
// no-ops after any such step, leaving orphaned model servers holding gigabytes
// of GPU memory until the next reboot.
//
// kern.bootsessionuuid is generated once per boot and never adjusted. The
// per-pid start-time check below remains the authority on pid recycling; this
// only says which boot the ledger belongs to.
func bootSessionUUID() string {
	s, err := unix.Sysctl("kern.bootsessionuuid")
	if err != nil || !isBootSessionUUID(s) {
		return ""
	}
	return s
}

// processStartNs returns a process's start time in nanoseconds. The (time, ok)
// pair distinguishes "process gone / unreadable" (ok=false) from a real value.
//
// P_starttime is a stored field of the exported extern_proc, stamped once when
// the process was forked and never recomputed: on the same Mac as above, pid 1's
// value did not move across the interval in which kern.boottime did. So the
// anti-recycle check this feeds is itself immune to the clock steps that made
// the boot-time stamp unusable.
func processStartNs(pid int) (int64, bool) {
	kp, err := unix.SysctlKinfoProc("kern.proc.pid", pid)
	if err != nil || kp == nil {
		return 0, false
	}
	return kp.Proc.P_starttime.Nano(), true
}
