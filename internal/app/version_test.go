package app

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/intentdriven/Dessau/internal/registry"
)

// testCommit is the commit every fake Hub in this package reports as current.
const testCommit = "0123456789abcdef0123456789abcdef01234567"

// atCommit answers a download's commit lookup with testCommit and serves the
// listing and the files at that commit from the routes a fake already has for
// "main", so the fakes written before downloads were pinned to a commit keep
// describing one repository.
func atCommit(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if r.URL.Query().Has("expand[]") && strings.HasPrefix(p, "/api/models/") {
			fmt.Fprintf(w, `{"id":%q,"sha":%q}`, strings.TrimPrefix(p, "/api/models/"), testCommit)
			return
		}
		r2 := r.Clone(r.Context())
		for _, pair := range [][2]string{
			{"/tree/" + testCommit, "/tree/main"},
			{"/resolve/" + testCommit + "/", "/resolve/main/"},
		} {
			r2.URL.Path = strings.Replace(r2.URL.Path, pair[0], pair[1], 1)
			r2.URL.RawPath = strings.Replace(r2.URL.RawPath, pair[0], pair[1], 1)
		}
		h.ServeHTTP(w, r2)
	})
}

// A completed download records the version it fetched — the commit and each
// file's hash — so that a later check has something to compare the Hub's
// answer with, and the panel can say which version is on disk.
func TestDownloadRecordsTheVersionItFetched(t *testing.T) {
	a := newTestApp(t)
	hub := fakeHub(t)
	a.Hub.BaseURL = hub.URL

	if err := a.Download("org/repo"); err != nil {
		t.Fatalf("Download: %v", err)
	}
	waitFor(t, "the model to become ready", func() bool {
		m, err := a.Registry.Get("org/repo")
		return err == nil && m.Ready()
	})
	m, _ := a.Registry.Get("org/repo")
	if m.Commit != testCommit {
		t.Errorf("Commit = %q, want %q", m.Commit, testCommit)
	}
	for _, f := range []string{"config.json", "model.safetensors", "tokenizer.json"} {
		if m.FileHashes[f] == "" {
			t.Errorf("no hash recorded for %s: %v", f, m.FileHashes)
		}
	}
}

// A re-download that fails leaves the model it did not replace exactly as it
// was recorded, version included: the files on disk are still that version.
func TestAFailedRedownloadKeepsTheRecordedVersion(t *testing.T) {
	a := newTestApp(t)
	hub := fakeHub(t)
	a.Hub.BaseURL = hub.URL
	if err := a.Download("org/repo"); err != nil {
		t.Fatalf("Download: %v", err)
	}
	waitFor(t, "the model to become ready", func() bool {
		m, err := a.Registry.Get("org/repo")
		return err == nil && m.Ready()
	})
	before, _ := a.Registry.Get("org/repo")

	broken := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "gone", http.StatusNotFound)
	}))
	defer broken.Close()
	a.Hub.BaseURL = broken.URL
	var dlErr error
	waitFor(t, "the re-download to start", func() bool {
		dlErr = a.Download("org/repo")
		return !errors.Is(dlErr, ErrAlreadyDownloading)
	})
	if dlErr != nil {
		t.Fatalf("Download: %v", dlErr)
	}
	waitFor(t, "the failed attempt to settle", func() bool {
		m, err := a.Registry.Get("org/repo")
		return err == nil && m.State != registry.StateDownloading
	})
	after, _ := a.Registry.Get("org/repo")
	if after.State != registry.StateReady {
		t.Fatalf("state = %q, want ready", after.State)
	}
	if after.Commit != before.Commit || len(after.FileHashes) != len(before.FileHashes) {
		t.Errorf("the failed re-download changed the recorded version: %q %v, was %q %v",
			after.Commit, after.FileHashes, before.Commit, before.FileHashes)
	}
}

// gitBlobID is the id git gives a file's content, which is what the Hub's tree
// reports as the oid of a file the repository stores in git.
func gitBlobID(b []byte) string {
	h := sha1.New()
	fmt.Fprintf(h, "blob %d\x00", len(b))
	h.Write(b)
	return hex.EncodeToString(h.Sum(nil))
}
