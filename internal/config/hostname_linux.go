package config

import (
	"os"
	"strings"
)

// Dessau only ever runs on Apple Silicon Macs. These two readers exist so that
// the module builds and its tests run in a Linux container. Linux has no
// scutil and no separate readable Computer Name, so each answers with what the
// macOS readers themselves fall back to when scutil says nothing: the system
// hostname, without a ".local" suffix.

// LocalHostName returns this machine's hostname, or "" if it cannot be read.
// The macOS reader's fallback, and the name a Linux mDNS responder publishes.
func LocalHostName() string {
	h, err := os.Hostname()
	if err != nil {
		return ""
	}
	return strings.TrimSuffix(h, ".local")
}

// ComputerName returns LocalHostName: Linux keeps no readable name apart from
// the hostname, and the macOS reader falls back to the same thing.
func ComputerName() string {
	return LocalHostName()
}
