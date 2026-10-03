package runtime

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// privateSocket is a socket path in a directory newSocketDir made, removed
// when the test ends — what the pool hands a launch.
func privateSocket(t *testing.T) string {
	t.Helper()
	dir, err := newSocketDir()
	if err != nil {
		t.Fatalf("newSocketDir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	path, err := socketPath(dir, 1)
	if err != nil {
		t.Fatalf("socketPath: %v", err)
	}
	return path
}

// The directory a model server's socket lives in is this account's alone:
// a directory, not a link, mode 0700, owned by the account running Dessau.
// That directory is the whole of the access control — a Unix socket's own
// mode is not honoured everywhere, a directory's search bit is — so it is
// checked where it is made and not merely assumed.
func TestTheSocketDirectoryIsThisAccountsAlone(t *testing.T) {
	dir, err := newSocketDir()
	if err != nil {
		t.Fatalf("newSocketDir: %v", err)
	}
	defer os.RemoveAll(dir)

	fi, err := os.Lstat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !fi.IsDir() {
		t.Fatalf("%s is not a directory: %v", dir, fi.Mode())
	}
	if perm := fi.Mode().Perm(); perm != 0o700 {
		t.Errorf("socket directory mode = %o, want 700", perm)
	}
	if uid := fi.Sys().(*syscall.Stat_t).Uid; int(uid) != os.Geteuid() {
		t.Errorf("socket directory is owned by uid %d, want this account's %d", uid, os.Geteuid())
	}
	if err := checkSocketDir(dir, os.Geteuid()); err != nil {
		t.Errorf("checkSocketDir refused the directory newSocketDir made: %v", err)
	}
	// Room for any socket name the pool will ever give it.
	if _, err := socketPath(dir, ^uint64(0)); err != nil {
		t.Errorf("the socket directory leaves no room for the longest socket name: %v", err)
	}
}

// A directory that is not this account's alone is refused, whatever it holds:
// one another account can open, one another account owns, a link (which
// could be re-pointed between the check and the bind), and a file.
func TestASocketDirectoryThatIsNotPrivateIsRefused(t *testing.T) {
	base := shortTempDir(t)
	mk := func(name string, mode os.FileMode) string {
		p := filepath.Join(base, name)
		if err := os.Mkdir(p, mode); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(p, mode); err != nil { // past the umask
			t.Fatal(err)
		}
		return p
	}
	open := mk("open", 0o755)
	group := mk("group", 0o770)
	private := mk("private", 0o700)
	link := filepath.Join(base, "link")
	if err := os.Symlink(private, link); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(base, "file")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	for name, c := range map[string]struct {
		dir string
		uid int
	}{
		"readable by others":     {open, os.Geteuid()},
		"open to the group":      {group, os.Geteuid()},
		"owned by other account": {private, os.Geteuid() + 1},
		"a link":                 {link, os.Geteuid()},
		"a file":                 {file, os.Geteuid()},
		"missing":                {filepath.Join(base, "missing"), os.Geteuid()},
	} {
		if err := checkSocketDir(c.dir, c.uid); err == nil {
			t.Errorf("%s: checkSocketDir accepted it", name)
		}
	}
	if err := checkSocketDir(private, os.Geteuid()); err != nil {
		t.Errorf("checkSocketDir refused a private directory: %v", err)
	}
}

// macOS holds a Unix socket's path to 104 bytes, terminator included. A path
// past that cannot be bound at all, so it is refused with a reason rather
// than left to fail as a model server that never answers.
func TestASocketPathPastTheKernelsLimitIsRefused(t *testing.T) {
	long := "/" + strings.Repeat("d", maxSocketPath)
	if _, err := socketPath(long, 1); err == nil {
		t.Fatal("socketPath accepted a path longer than a Unix socket may have")
	}
	p, err := socketPath("/s", 7)
	if err != nil {
		t.Fatalf("socketPath refused a short path: %v", err)
	}
	if len(p) > maxSocketPath {
		t.Fatalf("socketPath returned %d bytes", len(p))
	}
}

// Launch refuses a socket outside a private directory before any process
// exists, so the launcher cannot be talked into binding where another account
// could reach.
func TestLaunchRefusesASocketOutsideAPrivateDirectory(t *testing.T) {
	l := stubbedLauncher(t, `echo started > "$0.ran"; exit 0`)
	open := filepath.Join(shortTempDir(t), "open")
	if err := os.Mkdir(open, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(open, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, sock := range map[string]string{
		"no socket at all":      "",
		"in an open directory":  filepath.Join(open, "m1"),
		"in a missing one":      filepath.Join(open, "missing", "m1"),
		"past the kernel limit": "/" + strings.Repeat("d", maxSocketPath) + "/m1",
	} {
		p, err := l.Launch(context.Background(), Spec{RepoID: "org/name", ModelPath: plainModelDir(t), Socket: sock})
		if p != nil {
			<-p.Done()
		}
		if err == nil {
			t.Errorf("%s: Launch started a model server", name)
		}
	}
	if _, err := os.Stat(l.Paths.VenvPython() + ".ran"); err == nil {
		t.Error("the interpreter ran for a refused launch")
	}
}

// Something under the socket's name that is not a socket is not the
// launcher's to clear: Launch refuses before any process exists and leaves
// the file as it was, rather than removing whatever it finds there.
func TestLaunchRefusesAFileInTheSocketsPlace(t *testing.T) {
	sock := privateSocket(t)
	if err := os.WriteFile(sock, []byte("not a socket"), 0o600); err != nil {
		t.Fatal(err)
	}
	l := stubbedLauncher(t, `echo started > "$0.ran"; exit 0`)
	p, err := l.Launch(context.Background(), Spec{RepoID: "org/name", ModelPath: plainModelDir(t), Socket: sock})
	if p != nil {
		<-p.Done()
	}
	if err == nil {
		t.Error("Launch started a model server with a file in its socket's place")
	}
	if b, err := os.ReadFile(sock); err != nil || string(b) != "not a socket" {
		t.Errorf("the file in the socket's place was not left as it was: %q, %v", b, err)
	}
	if _, err := os.Stat(l.Paths.VenvPython() + ".ran"); err == nil {
		t.Error("the interpreter ran for a refused launch")
	}
}

// A socket left under the name by a run that ended without removing it is
// cleared before the launch, and the launch's own socket is removed once the
// process has gone, so neither the next launch nor anything reading the
// directory meets a name nothing answers on.
func TestASocketIsClearedBeforeLaunchAndRemovedAfterExit(t *testing.T) {
	sock := privateSocket(t)
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	ln.(*net.UnixListener).SetUnlinkOnClose(false)
	ln.Close() // a stale socket: the name is there, nothing answers

	// The stand-in records whether the name was still there when it started,
	// then leaves a socket of its own behind, as a killed server does.
	l := stubbedLauncher(t, `if [ -e "`+sock+`" ]; then echo stale > "$0.saw"; fi; exit 0`)
	p, err := l.Launch(context.Background(), Spec{RepoID: "org/name", ModelPath: plainModelDir(t), Socket: sock})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	<-p.Done()
	if _, err := os.Stat(l.Paths.VenvPython() + ".saw"); err == nil {
		t.Error("the stale socket was still there when the model server started")
	}

	// Leave a socket under the name while a process runs, then let it exit.
	l = stubbedLauncher(t, `sleep 0.3; exit 0`)
	p, err = l.Launch(context.Background(), Spec{RepoID: "org/name", ModelPath: plainModelDir(t), Socket: sock})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	ln, err = net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	ln.(*net.UnixListener).SetUnlinkOnClose(false)
	ln.Close()
	<-p.Done()
	if _, err := os.Lstat(sock); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the socket outlived its model server: %v", err)
	}
}

// A socket directory a crashed run left behind is removed at the next start;
// one a running Dessau is using is not, and neither is one that is not this
// account's alone.
func TestStaleSocketDirectoriesAreSweptAndLiveOnesKept(t *testing.T) {
	root := shortTempDir(t)
	dead := deadPID(t)
	mk := func(pid int, mode os.FileMode) string {
		dir, err := os.MkdirTemp(root, fmt.Sprintf("%s%d-", socketDirPrefix, pid))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(dir, mode); err != nil {
			t.Fatal(err)
		}
		return dir
	}
	stale := mk(dead, 0o700)
	if err := os.WriteFile(filepath.Join(stale, "m1"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	live := mk(os.Getpid(), 0o700)
	notOurs := mk(dead, 0o755)
	other := filepath.Join(root, "unrelated")
	if err := os.Mkdir(other, 0o700); err != nil {
		t.Fatal(err)
	}

	sweepSocketDirs(root)

	if _, err := os.Lstat(stale); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("a dead run's socket directory was kept: %v", err)
	}
	for _, kept := range []string{live, notOurs, other} {
		if _, err := os.Lstat(kept); err != nil {
			t.Errorf("%s was removed: %v", filepath.Base(kept), err)
		}
	}
}

// The transport a caller is handed reaches the model server through its
// socket and nowhere else: the URL's host is a label the transport never
// resolves, and no proxy from the environment is consulted.
func TestTheChildTransportReachesTheServerThroughItsSocket(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:1")
	sock := privateSocket(t)
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "over the socket: "+r.URL.Path)
	})}
	go srv.Serve(ln)
	defer srv.Close()

	c := &http.Client{Transport: childTransport(sock), Timeout: 5 * time.Second}
	resp, err := c.Get(childBaseURL + "/health")
	if err != nil {
		t.Fatalf("GET over the socket: %v", err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if string(b) != "over the socket: /health" {
		t.Errorf("body = %q", b)
	}
	if !strings.HasPrefix(childBaseURL, "http://") || strings.ContainsAny(strings.TrimPrefix(childBaseURL, "http://"), ":0123456789") {
		t.Errorf("childBaseURL %q names an address or a port; it is a label", childBaseURL)
	}
}

// macOS clears from a temporary directory what has not been touched for three
// days (dirhelper, nightly), and a pinned model's server runs for longer than
// that: its socket would vanish under it, and every request after would find
// nothing to connect to. So the launcher keeps a running server's socket, and
// the directory it is in, fresh for as long as the process lives.
func TestARunningServersSocketIsKeptFresh(t *testing.T) {
	sock := privateSocket(t)
	l := stubbedLauncher(t, `sleep 1; exit 0`)
	l.socketRefresh = 50 * time.Millisecond
	p, err := l.Launch(context.Background(), Spec{RepoID: "org/name", ModelPath: plainModelDir(t), Socket: sock})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	defer func() { <-p.Done() }()
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	ln.(*net.UnixListener).SetUnlinkOnClose(false)
	defer ln.Close()

	old := time.Now().Add(-4 * 24 * time.Hour)
	for _, path := range []string{sock, filepath.Dir(sock)} {
		if err := os.Chtimes(path, old, old); err != nil {
			t.Fatal(err)
		}
	}
	time.Sleep(300 * time.Millisecond)
	for _, path := range []string{sock, filepath.Dir(sock)} {
		fi, err := os.Lstat(path)
		if err != nil {
			t.Fatalf("%s: %v", filepath.Base(path), err)
		}
		if age := time.Since(fi.ModTime()); age > time.Hour {
			t.Errorf("%s was left %s old while its server ran", filepath.Base(path), age.Round(time.Hour))
		}
	}
}

// The hourly touch is made only of a directory that is still this account's
// alone. Where the socket directory's name has become a link, the touch would
// follow it to whatever it points at; where it has been opened to others, it
// is no longer the directory the pool made private, and the pool replaces it
// at the next load rather than keeping it alive.
func TestTheSocketRefreshTouchesOnlyAPrivateDirectory(t *testing.T) {
	old := time.Now().Add(-4 * 24 * time.Hour)
	refresh := func(sock string) {
		t.Helper()
		done := make(chan struct{})
		finished := make(chan struct{})
		go func() {
			keepSocketFresh(sock, 10*time.Millisecond, done)
			close(finished)
		}()
		time.Sleep(200 * time.Millisecond)
		close(done)
		<-finished
	}
	stillOld := func(what, path string) {
		t.Helper()
		fi, err := os.Lstat(path)
		if err != nil {
			t.Fatalf("%s: %v", what, err)
		}
		if age := time.Since(fi.ModTime()); age < time.Hour {
			t.Errorf("%s was touched (%s old)", what, age.Round(time.Millisecond))
		}
	}

	t.Run("a link", func(t *testing.T) {
		sock := privateSocket(t)
		dir := filepath.Dir(sock)
		decoy := filepath.Join(shortTempDir(t), "decoy")
		if err := os.Mkdir(decoy, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(decoy, old, old); err != nil {
			t.Fatal(err)
		}
		moved := dir + ".moved"
		if err := os.Rename(dir, moved); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { os.RemoveAll(moved) })
		if err := os.Symlink(decoy, dir); err != nil {
			t.Fatal(err)
		}
		refresh(sock)
		stillOld("the directory the link points at", decoy)
	})

	t.Run("open to others", func(t *testing.T) {
		sock := privateSocket(t)
		dir := filepath.Dir(sock)
		if err := os.Chmod(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(dir, old, old); err != nil {
			t.Fatal(err)
		}
		refresh(sock)
		stillOld("the open directory", dir)
	})
}

// A connection is used only when the process listening on the other end runs
// as this account. Where the socket's name has been taken over — its
// directory gone and made again by somebody else, a temporary directory
// another account can write — the listener is theirs, and a prompt sent to
// it would be handed to them.
func TestTheChildTransportRefusesAListenerOfAnotherAccount(t *testing.T) {
	sock := privateSocket(t)
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			c.Close()
		}
	}()

	if c, err := dialChild(context.Background(), sock, os.Geteuid()); err != nil {
		t.Fatalf("a listener of this account's was refused: %v", err)
	} else {
		c.Close()
	}
	if c, err := dialChild(context.Background(), sock, os.Geteuid()+1); err == nil {
		c.Close()
		t.Fatal("a listener running as another account was used")
	}
}

// shortTempDir is a temporary directory with a short path: t.TempDir is named
// after the test and can push a socket path past the kernel's limit.
func shortTempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "ds")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

// deadPID is the pid of a process that has exited and been reaped.
func deadPID(t *testing.T) int {
	t.Helper()
	p, err := os.StartProcess("/usr/bin/true", []string{"true"}, &os.ProcAttr{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Wait(); err != nil {
		t.Fatal(err)
	}
	return p.Pid
}

// A socket directory is removed only while it is still a directory this
// account owns: a name another account has taken since is theirs, and what
// is in it is not this account's to delete.
func TestASocketDirectoryOfAnotherAccountIsNotRemoved(t *testing.T) {
	dir := filepath.Join(shortTempDir(t), "dir")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	keep := filepath.Join(dir, "theirs")
	if err := os.WriteFile(keep, []byte("theirs"), 0o600); err != nil {
		t.Fatal(err)
	}
	removeSocketDir(dir, os.Geteuid()+1)
	if _, err := os.Lstat(keep); err != nil {
		t.Errorf("a directory owned by another account was emptied: %v", err)
	}
	removeSocketDir(dir, os.Geteuid())
	if _, err := os.Lstat(dir); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("this account's own socket directory was kept: %v", err)
	}
}
