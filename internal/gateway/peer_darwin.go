//go:build darwin && cgo

package gateway

/*
#include <stdlib.h>
#include <string.h>
#include <stdint.h>
#include <arpa/inet.h>
#include <libproc.h>
#include <sys/proc_info.h>

// dessau_addr_eq reports whether a TCP socket's own address and its peer's
// are lip and rip, of the family v6 says.
static int dessau_addr_eq(const struct in_sockinfo *in, int v6, const unsigned char *lip, const unsigned char *rip) {
	if (!v6) {
		if (!(in->insi_vflag & INI_IPV4)) return 0;
		return memcmp(&in->insi_laddr.ina_46.i46a_addr4, lip, 4) == 0 &&
		       memcmp(&in->insi_faddr.ina_46.i46a_addr4, rip, 4) == 0;
	}
	if (!(in->insi_vflag & INI_IPV6)) return 0;
	return memcmp(&in->insi_laddr.ina_6, lip, 16) == 0 &&
	       memcmp(&in->insi_faddr.ina_6, rip, 16) == 0;
}

// dessau_peer_uid looks, among the processes this account may inspect, for
// the TCP socket whose own end is lip:lport and whose peer is rip:rport, and
// writes the effective uid of the process holding it. It returns 1 when one
// is found, 0 when none is, and -1 when the process list cannot be read.
static int dessau_peer_uid(int v6, const unsigned char *lip, int lport,
                           const unsigned char *rip, int rport, unsigned int *uid) {
	int bytes = proc_listpids(PROC_ALL_PIDS, 0, NULL, 0);
	if (bytes <= 0) return -1;
	int cap = bytes + 64 * (int)sizeof(pid_t);
	pid_t *pids = malloc(cap);
	if (pids == NULL) return -1;
	bytes = proc_listpids(PROC_ALL_PIDS, 0, pids, cap);
	if (bytes <= 0) {
		free(pids);
		return -1;
	}
	int n = bytes / (int)sizeof(pid_t);
	int found = 0;
	for (int i = 0; i < n && !found; i++) {
		pid_t pid = pids[i];
		if (pid <= 0) continue;
		int fdbytes = proc_pidinfo(pid, PROC_PIDLISTFDS, 0, NULL, 0);
		if (fdbytes <= 0) continue;
		struct proc_fdinfo *fds = malloc(fdbytes);
		if (fds == NULL) continue;
		fdbytes = proc_pidinfo(pid, PROC_PIDLISTFDS, 0, fds, fdbytes);
		int nfd = fdbytes > 0 ? fdbytes / (int)sizeof(struct proc_fdinfo) : 0;
		for (int j = 0; j < nfd; j++) {
			if (fds[j].proc_fdtype != PROX_FDTYPE_SOCKET) continue;
			struct socket_fdinfo si;
			if (proc_pidfdinfo(pid, fds[j].proc_fd, PROC_PIDFDSOCKETINFO, &si, sizeof si) != (int)sizeof si) continue;
			if (si.psi.soi_kind != SOCKINFO_TCP) continue;
			const struct in_sockinfo *in = &si.psi.soi_proto.pri_tcp.tcpsi_ini;
			if (ntohs((uint16_t)in->insi_lport) != lport || ntohs((uint16_t)in->insi_fport) != rport) continue;
			if (!dessau_addr_eq(in, v6, lip, rip)) continue;
			struct proc_bsdinfo bi;
			if (proc_pidinfo(pid, PROC_PIDTBSDINFO, 0, &bi, sizeof bi) != (int)sizeof bi) continue;
			*uid = bi.pbi_uid;
			found = 1;
			break;
		}
		free(fds);
	}
	free(pids);
	return found;
}
*/
import "C"

import (
	"errors"
	"net/netip"
	"unsafe"
)

// peerLookupSupported says whether lookupPeerUID can answer in this build.
const peerLookupSupported = true

var errPeerLookupFailed = errors.New("the list of processes could not be read")

// lookupPeerUID asks libproc which process holds the other end of the
// connection the server sees as local<-remote: the socket whose own address is
// remote and whose peer is local. Only the processes this account may
// inspect are looked at, so a connection from another account's process is
// not found.
func lookupPeerUID(local, remote netip.AddrPort) (uint32, bool, error) {
	l, r := local.Addr().Unmap(), remote.Addr().Unmap()
	if !l.IsValid() || !r.IsValid() || l.Is4() != r.Is4() {
		return 0, false, nil
	}
	var own, peer []byte
	v6 := 0
	if r.Is4() {
		a, b := r.As4(), l.As4()
		own, peer = a[:], b[:]
	} else {
		a, b := r.As16(), l.As16()
		own, peer = a[:], b[:]
		v6 = 1
	}
	var uid C.uint
	switch C.dessau_peer_uid(C.int(v6),
		(*C.uchar)(unsafe.Pointer(&own[0])), C.int(remote.Port()),
		(*C.uchar)(unsafe.Pointer(&peer[0])), C.int(local.Port()), &uid) {
	case 1:
		return uint32(uid), true, nil
	case 0:
		return 0, false, nil
	default:
		return 0, false, errPeerLookupFailed
	}
}
