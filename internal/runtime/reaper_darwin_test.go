package runtime

import "golang.org/x/sys/unix"

// kernelBootSession is the oracle the ledger's stamp is checked against, read
// independently of bootSessionUUID.
func kernelBootSession() (string, error) {
	return unix.Sysctl("kern.bootsessionuuid")
}
