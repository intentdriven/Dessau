package lifecycle

import "errors"

// hostMacOSVersion has no answer off a Mac. Dessau only ever runs on Apple
// Silicon Macs; this exists so that the module builds and its tests run in a
// Linux container. The error is the safe answer: belowFloor reads an
// unreadable version as below the floor, so `dessau update` refuses before it
// downloads or quits anything.
func hostMacOSVersion() (string, error) {
	return "", errors.New("the macOS version cannot be read: not supported on this platform")
}
