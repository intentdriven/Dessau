package runtime

import "golang.org/x/sys/unix"

// socketPeerUID asks the kernel which account the other end of a connected
// Unix socket runs as.
//
// Dessau only ever runs on Apple Silicon Macs; this exists so that the module
// builds and its tests run in a Linux container. It is the same check, not a
// stand-in that accepts any peer: SO_PEERCRED is Linux's equivalent of
// LOCAL_PEERCRED, and on a connected socket it answers with the credentials
// the listener had when it called listen.
func socketPeerUID(fd int) (uint32, error) {
	cred, err := unix.GetsockoptUcred(fd, unix.SOL_SOCKET, unix.SO_PEERCRED)
	if err != nil {
		return 0, err
	}
	return cred.Uid, nil
}
