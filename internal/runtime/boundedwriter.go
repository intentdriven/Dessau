package runtime

import (
	"io"
	"sync"
)

// debugLogBoundLine is the one line written when the bound is reached. It
// says what happened and that nothing else did: the model keeps serving,
// only its log has stopped.
const debugLogBoundLine = "\n--- log stopped at its bound; the model server is still serving ---\n"

// boundedWriter writes through to w until max bytes have passed, then writes
// the one final line and discards the rest. It never returns an error and
// never blocks on its own account: the model server writes its log to a pipe
// this drains, and a writer that stopped draining would stall the server,
// turning a diagnostic into an outage. Every write is reported as accepted.
// Safe for concurrent use.
type boundedWriter struct {
	mu      sync.Mutex
	w       io.Writer
	max     int64
	n       int64
	stopped bool
}

func newBoundedWriter(w io.Writer, max int64) *boundedWriter {
	return &boundedWriter{w: w, max: max}
}

func (b *boundedWriter) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.stopped {
		return len(p), nil
	}
	room := b.max - b.n
	if int64(len(p)) <= room {
		n, _ := b.w.Write(p)
		b.n += int64(n)
		return len(p), nil
	}
	// The part that fits, then the final line, then nothing more — including
	// the rest of this write, which is the case of one body larger than the
	// whole bound.
	if room > 0 {
		b.w.Write(p[:room])
		b.n = b.max
	}
	b.stopped = true
	b.w.Write([]byte(debugLogBoundLine))
	return len(p), nil
}
