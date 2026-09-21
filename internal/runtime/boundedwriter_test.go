package runtime

import (
	"bytes"
	"strings"
	"sync"
	"testing"
)

// The bytes in an armed model server's log are chosen by whoever is sending
// requests, so the bound is the control: a looping client, or the context
// probe's one filler body that is larger than the bound on its own, stops the
// file at the bound rather than filling the disk. The writer never refuses a
// write and never blocks — a writer that stopped draining would fill the pipe
// and stall the model server, which would turn a diagnostic into an outage —
// so every write is reported as accepted whether or not it reached the file.
func TestTheDebugLogStopsAtItsBound(t *testing.T) {
	const bound = 100

	t.Run("a looping client writes until the bound and then nothing", func(t *testing.T) {
		var sink bytes.Buffer
		w := newBoundedWriter(&sink, bound)
		line := []byte(strings.Repeat("x", 30) + "\n")
		for i := 0; i < 20; i++ {
			n, err := w.Write(line)
			if err != nil || n != len(line) {
				t.Fatalf("write %d reported (%d, %v), want (%d, nil): a refused write stalls the model server", i, n, err, len(line))
			}
		}
		got := sink.String()
		body, final, ok := strings.Cut(got, debugLogBoundLine)
		if !ok {
			t.Fatalf("the log carries no final line saying it stopped at its bound: %q", got)
		}
		if len(body) != bound {
			t.Errorf("the log holds %d bytes before the final line, want exactly the bound %d", len(body), bound)
		}
		if final != "" {
			t.Errorf("bytes were written after the final line: %q", final)
		}
	})

	t.Run("a single write larger than the bound is cut at the bound", func(t *testing.T) {
		var sink bytes.Buffer
		w := newBoundedWriter(&sink, bound)
		filler := []byte(strings.Repeat("f", 3*bound))
		if n, err := w.Write(filler); err != nil || n != len(filler) {
			t.Fatalf("the oversized write reported (%d, %v), want (%d, nil)", n, err, len(filler))
		}
		got := sink.String()
		body, final, ok := strings.Cut(got, debugLogBoundLine)
		if !ok {
			t.Fatalf("the log carries no final line: %q", got)
		}
		if body != strings.Repeat("f", bound) {
			t.Errorf("the part that fits is %d bytes, want the first %d of the write", len(body), bound)
		}
		if final != "" {
			t.Errorf("bytes were written after the final line: %q", final)
		}
		// And the writer stays shut.
		if n, err := w.Write([]byte("more")); err != nil || n != 4 {
			t.Errorf("a write after the bound reported (%d, %v), want (4, nil)", n, err)
		}
		if sink.String() != got {
			t.Error("a write after the bound reached the file")
		}
	})

	t.Run("the final line says the model is still serving", func(t *testing.T) {
		for _, want := range []string{"stopped", "still serving"} {
			if !strings.Contains(debugLogBoundLine, want) {
				t.Errorf("the final line %q does not say %q", debugLogBoundLine, want)
			}
		}
		if !strings.HasSuffix(debugLogBoundLine, "\n") {
			t.Error("the final line does not end the file with a newline")
		}
	})

	t.Run("concurrent writers never exceed the bound", func(t *testing.T) {
		var sink bytes.Buffer
		w := newBoundedWriter(&sink, bound)
		var wg sync.WaitGroup
		for g := 0; g < 8; g++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := 0; i < 50; i++ {
					w.Write([]byte("0123456789"))
				}
			}()
		}
		wg.Wait()
		body, _, ok := strings.Cut(sink.String(), debugLogBoundLine)
		if !ok {
			t.Fatal("the log carries no final line")
		}
		if len(body) != bound {
			t.Errorf("the log holds %d bytes before the final line, want %d", len(body), bound)
		}
	})
}
