package runtime

import "golang.org/x/sys/unix"

// socketPeerUID asks the kernel which account the other end of a connected
// Unix socket runs as. LOCAL_PEERCRED answers with the credentials the
// listener had when it called listen, which is the account that owns the
// model server peerIs is checking.
func socketPeerUID(fd int) (uint32, error) {
	cred, err := unix.GetsockoptXucred(fd, unix.SOL_LOCAL, unix.LOCAL_PEERCRED)
	if err != nil {
		return 0, err
	}
	return cred.Uid, nil
}
