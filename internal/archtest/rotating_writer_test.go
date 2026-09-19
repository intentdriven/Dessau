package archtest_test

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/intentdriven/Gropius/internal/applog"
	"github.com/intentdriven/Gropius/internal/stats"
)

// Gropius writes three bounded line files on this Mac — its own log, the
// request statistics store and the self-test's results — and every one of them
// is opened under one discipline: through an os.Root on its own directory,
// never following a link, never waiting on a pipe, and refusing a handle that
// turns out not to be this account's own regular file.
//
// WHY IT IS A RULE AND NOT A PREFERENCE. The discipline was copied rather than
// shared, and iss-2609091714393599 recorded exactly how that ends: "the two
// drifting on the discipline they share — one gaining a check on the opened
// handle, or a mode, or a symlink refusal that the other does not". A copy is
// invisible when it is right and silent when it is wrong, and these three files
// sit in directories whose names anything running as this account can guess. So
// the primitive lives in one place (applog.OpenIn) and this is the test that
// every writer is still behind it: it plants the two things a name can be
// standing in for, and asks each writer to write.
//
// WHAT IT CANNOT DO. It proves the refusal, not the route: a writer that
// re-derived the same checks inline would pass. What makes the route visible is
// that there is one exported open and the packages call it.
func TestEveryBoundedWriterRefusesAPlantedFile(t *testing.T) {
	writers := []struct {
		name string
		// file is the name the writer is about to use inside dir.
		file func() string
		// write asks the writer to write one line into dir.
		write func(t *testing.T, dir string)
	}{
		{
			name: "the log",
			file: func() string { return applog.DefaultName },
			write: func(t *testing.T, dir string) {
				t.Helper()
				l, _ := applog.Open(applog.Options{Dir: dir, Stderr: io.Discard})
				l.Logger.Info("a line that must not reach a planted file")
				l.Close()
			},
		},
		{
			name: "the statistics store",
			// The store's first file of the day, which is the name it reaches
			// for when the directory holds nothing it can append to.
			file: func() string {
				return fmt.Sprintf("stats-%s-001.jsonl", time.Now().UTC().Format("20060102"))
			},
			write: func(t *testing.T, dir string) {
				t.Helper()
				s := stats.NewStore(dir, stats.StoreOptions{Log: discardLog()})
				// An unopenable store is a refusal in itself; either way
				// nothing may reach the planted name.
				_ = s.SetEnabled(true)
				if err := s.AppendRequest(stats.Record{Model: "org/a", At: time.Now().Unix(), Class: stats.ClassOK}); err != nil {
					t.Fatal(err)
				}
				_ = s.Flush()
				s.Close()
			},
		},
	}

	plants := []struct {
		name string
		// plant puts something under name inside dir and returns a function
		// reporting what the writer must not have changed.
		plant func(t *testing.T, dir, name string) func(t *testing.T)
	}{
		{
			name: "a link to somewhere else",
			plant: func(t *testing.T, dir, name string) func(t *testing.T) {
				t.Helper()
				target := filepath.Join(t.TempDir(), "target")
				if err := os.WriteFile(target, nil, 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, filepath.Join(dir, name)); err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					t.Helper()
					b, err := os.ReadFile(target)
					if err != nil {
						t.Fatal(err)
					}
					if len(b) != 0 {
						t.Errorf("%d bytes were written through a planted link", len(b))
					}
				}
			},
		},
		{
			name: "a file another account could read",
			plant: func(t *testing.T, dir, name string) func(t *testing.T) {
				t.Helper()
				path := filepath.Join(dir, name)
				const planted = "{\"planted\":true}\n"
				if err := os.WriteFile(path, []byte(planted), 0o600); err != nil {
					t.Fatal(err)
				}
				// Not a mode this account's own writers can produce: they
				// create at 0600 and a umask only takes bits away.
				if err := os.Chmod(path, 0o644); err != nil {
					t.Fatal(err)
				}
				return func(t *testing.T) {
					t.Helper()
					b, err := os.ReadFile(path)
					if err != nil {
						t.Fatal(err)
					}
					if string(b) != planted {
						t.Errorf("a writer appended to a file it did not make (mode 0644): %q", string(b))
					}
				}
			},
		},
	}

	for _, w := range writers {
		for _, p := range plants {
			t.Run(w.name+"/"+p.name, func(t *testing.T) {
				dir := filepath.Join(t.TempDir(), "d")
				if err := os.Mkdir(dir, 0o700); err != nil {
					t.Fatal(err)
				}
				check := p.plant(t, dir, w.file())
				w.write(t, dir)
				check(t)
			})
		}
	}
}

func discardLog() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
