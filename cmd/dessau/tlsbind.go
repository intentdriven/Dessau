package main

import (
	"crypto/tls"
	"log/slog"
	"net"
	"path/filepath"

	"github.com/intentdriven/Dessau/internal/bind"
	"github.com/intentdriven/Dessau/internal/config"
	"github.com/intentdriven/Dessau/internal/pairing"
)

// serverKeyFile is where this server's own TLS private key lives: beside
// config.json, under this account's own state directory, never under the
// shared root where a co-tenant could pre-plant one.
func serverKeyFile(paths config.Paths) string {
	return filepath.Join(filepath.Dir(paths.Config), "server-key.pem")
}

// acquireTLSBind takes the TLS listeners the same plan names, on the TLS port.
//
// The addresses are the bind plan's and nothing else's (adr-2609091123526871,
// rules 1 and 3). A TLS listener built independently — on the wildcard, or on
// cfg.Host — would serve the network from a process whose plain port had
// narrowed to this Mac, which is the fail-open every rule in that record exists
// to prevent. Loopback is first here too, so a bind that narrowed to this Mac
// still has a TLS port for a client running on it.
//
// Nothing here is fatal and nothing here probes. The plain port is the
// singleton's contention point and the port-ownership challenge contacts it
// alone; a TLS port already held is a port this server does not take, not a
// reason to refuse to start. The plain server goes on exactly as before, the
// advertisement drops its fingerprint, and the reason is logged.
func acquireTLSBind(plan bind.Plan, cfg config.Config, id *pairing.Identity, reg *pairing.Registry, log *slog.Logger) []net.Listener {
	port := cfg.EffectiveTLSPort()
	if port == 0 || id == nil {
		return nil
	}
	tlsCfg := reg.TLSConfig(id)
	var out []net.Listener
	for _, addr := range plan.Addrs(port) {
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			log.Error("could not listen for paired clients", "addr", addr, "err", err)
			continue
		}
		out = append(out, tls.NewListener(ln, tlsCfg))
	}
	return out
}

// tlsSANs are the names and addresses the server's own certificate is minted
// for: whatever the bind acquired, plus this Mac's own name.
//
// They decide nothing about trust — a client pins the key, and the leaf is
// reissued from that key whenever these change, which is why a renamed Mac or a
// new address costs no client its pairing.
func tlsSANs(plan bind.Plan, hostname string) []string {
	sans := []string{hostname, hostname + ".local"}
	if plan.Extra != "" && plan.Extra != "0.0.0.0" && plan.Extra != "::" {
		sans = append(sans, plan.Extra)
	}
	return sans
}
