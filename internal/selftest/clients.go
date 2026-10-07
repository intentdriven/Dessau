package selftest

import (
	"sync"
	"time"
)

// Clients is the idle clock's count of client requests, kept where every
// client request passes — the gateway — rather than on the model that serves
// it.
//
// The pool's own view (ModelActivity) is per resident model, and a model's
// figures leave with it: a request that ended unreachable on a wedged model
// that was then unloaded or stopped left no trace the idle loop could read,
// and a request that never reached the pool — still uploading its body,
// refused, or failed before a model was chosen — left none to begin with. On
// 2026-10-04 that let the self-test start a minute after a client's last
// unreachable request although the idle threshold was an hour
// (iss-2610041945030758). This count is kept beside the pool's, not instead
// of it.
//
// The zero value is ready to use, and a nil *Clients counts nothing.
type Clients struct {
	mu       sync.Mutex
	inFlight int
	last     time.Time
}

// Begin counts one client request as started and returns the function that
// counts it as ended. The request holds the Mac busy for as long as it is in
// flight, and the clock is stamped at both ends, so a request that ran for
// ten minutes is quiet from its end, not its start. Calling end more than once
// counts it once.
func (c *Clients) Begin() (end func()) {
	if c == nil {
		return func() {}
	}
	c.mu.Lock()
	c.inFlight++
	c.last = time.Now()
	c.mu.Unlock()
	var once sync.Once
	return func() {
		once.Do(func() {
			c.mu.Lock()
			c.inFlight--
			c.last = time.Now()
			c.mu.Unlock()
		})
	}
}

// Snapshot is how many client requests are in flight and when the last one
// started or ended; zero for both when none has been seen.
func (c *Clients) Snapshot() (inFlight int, last time.Time) {
	if c == nil {
		return 0, time.Time{}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.inFlight, c.last
}
