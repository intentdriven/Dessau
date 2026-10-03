package runtime

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// A model server listens on a Unix socket in a directory only the serving
// account can open (iss-2610030846581757). The directory is the access
// control: whoever cannot search it cannot reach the socket, whatever the
// socket's own mode. So it is made fresh by this process, checked where it is
// made, checked again before every launch, and removed when the pool closes.

// maxSocketPath is the longest path a Unix socket may have on macOS:
// sockaddr_un's sun_path is 104 bytes, the terminating NUL included.
const maxSocketPath = 103

// socketDirPrefix begins the name of every socket directory, followed by the
// pid of the process that made it, so a later start can tell a directory a
// running Dessau is using from one a crashed run left.
const socketDirPrefix = "dessau-"

// socketNameRoom is the longest socket name socketPath gives: "m" and the
// decimal of a uint64.
const socketNameRoom = len("/m") + 20

// socketRoots are where a socket directory is made, in order of preference:
// this account's own temporary directory, which macOS makes private to it,
// and then /tmp, for an account whose temporary directory is too deep to
// leave room for a socket name under the kernel's limit.
func socketRoots() []string {
	return []string{os.TempDir(), "/tmp"}
}

// newSocketDir makes a fresh directory for model-server sockets, private to
// this account, with room under it for any socket name socketPath gives.
//
// os.MkdirTemp creates the directory itself, under a name nothing else has
// used, so it is never one somebody else made earlier; it is checked anyway
// before it is handed out.
func newSocketDir() (string, error) {
	var last error
	for _, root := range socketRoots() {
		dir, err := os.MkdirTemp(root, socketDirPrefix+strconv.Itoa(os.Getpid())+"-")
		if err != nil {
			last = err
			continue
		}
		if len(dir)+socketNameRoom > maxSocketPath {
			os.Remove(dir)
			last = fmt.Errorf("a socket under %s would be longer than %d bytes", root, maxSocketPath)
			continue
		}
		// MkdirTemp asks for 0700; the umask can only take bits away, and a
		// chmod makes the mode this check expects whatever it was.
		if err := os.Chmod(dir, 0o700); err != nil {
			os.Remove(dir)
			return "", err
		}
		if err := checkSocketDir(dir, os.Geteuid()); err != nil {
			os.Remove(dir)
			return "", err
		}
		return dir, nil
	}
	return "", fmt.Errorf("no private directory for model-server sockets: %w", last)
}

// checkSocketDir refuses a socket directory that is not owner's alone: one
// that is missing, a link (which could be re-pointed between this check and
// the bind), not a directory, owned by another account, or open to anybody
// but its owner.
func checkSocketDir(dir string, owner int) error {
	fi, err := os.Lstat(dir)
	if err != nil {
		return fmt.Errorf("socket directory: %w", err)
	}
	if fi.Mode()&fs.ModeSymlink != 0 {
		return errors.New("socket directory is a link")
	}
	if !fi.IsDir() {
		return errors.New("socket directory is not a directory")
	}
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return errors.New("socket directory: cannot determine the owner")
	}
	if int(st.Uid) != owner {
		return fmt.Errorf("socket directory belongs to another account (uid %d)", st.Uid)
	}
	if perm := fi.Mode().Perm(); perm != 0o700 {
		return fmt.Errorf("socket directory is open to other accounts (mode %o)", perm)
	}
	return nil
}

// socketPath is the n-th socket in dir, refused when it would be longer than
// a Unix socket's path may be.
func socketPath(dir string, n uint64) (string, error) {
	p := filepath.Join(dir, "m"+strconv.FormatUint(n, 10))
	if len(p) > maxSocketPath {
		return "", fmt.Errorf("socket path is %d bytes, longer than the %d a Unix socket may have", len(p), maxSocketPath)
	}
	return p, nil
}

// removeSocket removes a socket left under path. Anything that is not a
// socket is left where it is: the name is the launcher's, but a file
// somebody put there is not, and the bind refuses it.
func removeSocket(path string) error {
	fi, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if fi.Mode()&fs.ModeSocket == 0 {
		return fmt.Errorf("%s is not a socket", filepath.Base(path))
	}
	return os.Remove(path)
}

// sweepSocketDirs removes the socket directories under root that a run which
// is no longer running left behind — a crash, a kill — and reports how many
// it removed. A directory whose pid is alive is kept, whoever that pid now
// belongs to: keeping a stale one costs a few bytes, removing a live one
// strands a running Dessau's model servers. Only a directory that is this
// account's alone is touched, and only through its own sockets.
func sweepSocketDirs(root string) int {
	entries, err := os.ReadDir(root)
	if err != nil {
		return 0
	}
	removed := 0
	for _, e := range entries {
		rest, ok := strings.CutPrefix(e.Name(), socketDirPrefix)
		if !ok {
			continue
		}
		pidText, _, ok := strings.Cut(rest, "-")
		pid, err := strconv.Atoi(pidText)
		if !ok || err != nil || pid <= 0 || processAlive(pid) {
			continue
		}
		dir := filepath.Join(root, e.Name())
		if checkSocketDir(dir, os.Geteuid()) != nil {
			continue
		}
		names, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, n := range names {
			_ = os.Remove(filepath.Join(dir, n.Name()))
		}
		if os.Remove(dir) == nil {
			removed++
		}
	}
	return removed
}

// processAlive reports whether pid names a process, this account's or not.
func processAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}
