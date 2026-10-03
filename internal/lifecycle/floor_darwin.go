package lifecycle

import "golang.org/x/sys/unix"

// hostMacOSVersion reads this Mac's product version — "27.0", "26.5.2" — the
// same value `sw_vers -productVersion` prints, without a subprocess or a PATH
// lookup. kern.osproductversion is the kernel's own copy of it.
func hostMacOSVersion() (string, error) {
	return unix.Sysctl("kern.osproductversion")
}
