package runtime

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strconv"
	"strings"
)

// Dessau only ever runs on Apple Silicon Macs. These two readers exist so that
// the module builds and its tests run in a Linux container, and they are the
// real Linux equivalents of the macOS ones rather than figures made up to
// satisfy a caller: the reaper SIGKILLs on what they say, so an invented
// answer would be an invented kill.

// bootSessionUUID returns this boot's identifier, or "" if it cannot be read.
//
// /proc/sys/kernel/random/boot_id is Linux's kern.bootsessionuuid: a random
// UUID generated once per boot and never adjusted. But a boot is not enough on
// Linux: every container on a host shares its boot_id, while pids — and so the
// pgids the ledger records — are numbered per pid namespace. Two containers
// sharing a data root would each read the other's ledger as this session's.
// So the session is the boot and the pid namespace together, folded into the
// UUID shape the ledger stores. Either one unreadable reads as no session,
// which reaps nothing.
func bootSessionUUID() string {
	b, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
	if err != nil {
		return ""
	}
	boot := strings.TrimSpace(string(b))
	if !isBootSessionUUID(boot) {
		return ""
	}
	ns, err := os.Readlink("/proc/self/ns/pid")
	if err != nil || ns == "" {
		return ""
	}
	return sessionOf(boot, ns)
}

// sessionOf folds a boot id and a pid namespace into one identifier in the
// 8-4-4-4-12 shape isBootSessionUUID accepts.
func sessionOf(boot, pidNamespace string) string {
	sum := sha256.Sum256([]byte(boot + "\x00" + pidNamespace))
	h := hex.EncodeToString(sum[:16])
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32]
}

// userHz is the unit /proc/<pid>/stat states a start time in. It is the
// kernel's USER_HZ, which is part of the user-space ABI and 100 on every
// architecture Go supports on Linux; it is not the configurable kernel HZ.
const userHz = 100

// processStartNs returns a process's start time in nanoseconds since boot. The
// (time, ok) pair distinguishes "process gone / unreadable" (ok=false) from a
// real value.
//
// Field 22 of /proc/<pid>/stat is the start time the kernel stamped when the
// process was forked, counted in USER_HZ ticks since boot and never
// recomputed, so it is immune to wall-clock steps just as P_starttime is. The
// reaper only ever compares two readings of it within one boot session.
func processStartNs(pid int) (int64, bool) {
	b, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return 0, false
	}
	return parseStatStartNs(string(b))
}

// parseStatStartNs reads the start time out of one /proc/<pid>/stat line.
//
// The second field is the command name in parentheses, and a process chooses
// its own name — spaces and parentheses included — so the fields are counted
// from the LAST closing parenthesis, never split from the start of the line.
func parseStatStartNs(stat string) (int64, bool) {
	i := strings.LastIndexByte(stat, ')')
	if i < 0 {
		return 0, false
	}
	// After the name come fields 3 (state) onwards; start time is field 22.
	fields := strings.Fields(stat[i+1:])
	const startField = 22 - 3
	if len(fields) <= startField {
		return 0, false
	}
	ticks, err := strconv.ParseInt(fields[startField], 10, 64)
	if err != nil || ticks < 0 {
		return 0, false
	}
	return ticks * (1_000_000_000 / userHz), true
}
