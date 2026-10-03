package capability

import "golang.org/x/sys/unix"

// PhysicalMemory returns installed memory in bytes, or 0 if it cannot be read.
//
// Dessau only ever runs on Apple Silicon Macs; this Linux reading exists so
// that the module builds and its tests run in a Linux container. It is a real
// reading, not an invented figure: sysinfo(2) is the kernel's own count of
// usable RAM, the figure /proc/meminfo reports as MemTotal, scaled by the unit
// the kernel states it in. Zero on failure is the contract the macOS reading
// keeps, and every caller already treats zero as unmeasured.
func PhysicalMemory() int64 {
	var si unix.Sysinfo_t
	if err := unix.Sysinfo(&si); err != nil {
		return 0
	}
	unit := uint64(si.Unit)
	if unit == 0 {
		unit = 1
	}
	return int64(uint64(si.Totalram) * unit)
}
