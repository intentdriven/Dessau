package selftest

import (
	"testing"
	"time"
)

// A client request counts from the moment it starts until it ends, and the
// clock reads the later of the two: a request in flight for ten minutes is
// ten minutes of a busy Mac, not one moment at its end.
func TestClientsCountARequestFromStartToEnd(t *testing.T) {
	var c Clients
	if n, last := c.Snapshot(); n != 0 || !last.IsZero() {
		t.Fatalf("a fresh clock reads %d in flight, last %v", n, last)
	}
	before := time.Now()
	end := c.Begin()
	n, started := c.Snapshot()
	if n != 1 || started.Before(before) {
		t.Fatalf("after Begin: %d in flight, last %v; want 1, at or after %v", n, started, before)
	}
	time.Sleep(time.Millisecond)
	end()
	end() // a second call counts nothing
	n, ended := c.Snapshot()
	if n != 0 || !ended.After(started) {
		t.Errorf("after the end: %d in flight, last %v; want 0, after %v", n, ended, started)
	}
}

// A gateway built without a clock counts nothing and does not fail.
func TestANilClientsCountsNothing(t *testing.T) {
	var c *Clients
	c.Begin()()
	if n, last := c.Snapshot(); n != 0 || !last.IsZero() {
		t.Errorf("a nil clock reads %d in flight, last %v", n, last)
	}
}
