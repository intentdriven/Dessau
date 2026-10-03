package capability

import (
	"os/exec"
	"strconv"
	"strings"
)

// PhysicalMemory returns installed memory in bytes, or 0 if it cannot be read.
//
// Exported and living here so that the process pool, this filter and the app
// all read one figure: the app needs it to check a saved budget against the
// machine and to show what share of it a budget is, and a second reading is a
// second answer waiting to happen.
func PhysicalMemory() int64 {
	// The absolute path, not the name: this figure decides how much memory
	// Dessau will fill, and resolving the command through PATH would let
	// anything earlier on it answer that question.
	out, err := exec.Command("/usr/sbin/sysctl", "-n", "hw.memsize").Output()
	if err != nil {
		return 0
	}
	n, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	if err != nil {
		return 0
	}
	return n
}
