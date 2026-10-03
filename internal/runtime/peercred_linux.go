package runtime

import (
	"errors"
	"os"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

// socketPeerUID asks the kernel which account the other end of a connected
// Unix socket runs as.
//
// Dessau only ever runs on Apple Silicon Macs; this exists so that the module
// builds and its tests run in a Linux container. It is the same check, not a
// stand-in that accepts any peer: SO_PEERCRED is Linux's equivalent of
// LOCAL_PEERCRED, and on a connected socket it answers with the credentials
// the listener had when it called listen.
//
// One Linux difference is closed here. The uid comes back translated into this
// process's user namespace, and a listener whose uid has no mapping there
// reads as the overflow uid. Were Dessau itself running as that uid, such a
// listener would pass for this account, so an unmapped peer is refused.
func socketPeerUID(fd int) (uint32, error) {
	cred, err := unix.GetsockoptUcred(fd, unix.SOL_SOCKET, unix.SO_PEERCRED)
	if err != nil {
		return 0, err
	}
	if cred.Uid == overflowUID() {
		return 0, errors.New("the listener's account is not mapped into this user namespace")
	}
	return cred.Uid, nil
}

// overflowUID is the uid the kernel reports for an account with no mapping in
// this user namespace: /proc/sys/kernel/overflowuid, and the kernel's default
// of 65534 when that cannot be read.
func overflowUID() uint32 {
	if b, err := os.ReadFile("/proc/sys/kernel/overflowuid"); err == nil {
		if n, err := strconv.ParseUint(strings.TrimSpace(string(b)), 10, 32); err == nil {
			return uint32(n)
		}
	}
	return 65534
}
