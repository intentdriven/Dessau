package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/Dessau/internal/config"
)

// standInServer is a stand-in for the pinned mlx_lm.server: the names the
// launcher uses from it, shaped as the real module has them (main() parses
// its flags and calls the module-level run(host, port, ModelProvider(args));
// APIHandler takes the response generator first and reads allowed_origins
// off its cli_args). Its own run() is the TCP server the launcher must never
// reach, so it leaves a marker and exits. Its handler answers a POST with the
// model and the keys it was given, so a test can see what reached it.
const standInServer = `
import argparse, json, os, sys
from http.server import BaseHTTPRequestHandler

def get_system_fingerprint():
    return "stand-in"

class LRUPromptCache:
    def __init__(self, size):
        self.size = size

class ModelProvider:
    def __init__(self, cli_args):
        self.cli_args = cli_args

class ResponseGenerator:
    def __init__(self, model_provider, prompt_cache):
        self.model_provider = model_provider
    @property
    def cli_args(self):
        return self.model_provider.cli_args
    def stop_and_join(self):
        pass

class APIHandler(BaseHTTPRequestHandler):
    def __init__(self, response_generator, *args, system_fingerprint=None, **kwargs):
        self.response_generator = response_generator
        super().__init__(*args, **kwargs)

    def _reply(self, code, obj):
        body = json.dumps(obj).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        origins = self.response_generator.cli_args.allowed_origins
        origin = self.headers.get("Origin")
        if "*" in origins:
            self.send_header("Access-Control-Allow-Origin", "*")
        elif origin in origins:
            self.send_header("Access-Control-Allow-Origin", origin)
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_GET(self):
        self._reply(200, {"status": "ok"})

    def do_POST(self):
        n = int(self.headers.get("Content-Length"))
        body = json.loads(self.rfile.read(n))
        self._reply(200, {"model": body.get("model"), "keys": sorted(body)})

def run(host, port, model_provider, server_class=None, handler_class=None):
    with open(os.environ["STANDIN_TCP_MARKER"], "w") as f:
        f.write("%s:%s" % (host, port))
    sys.exit(3)

def main():
    p = argparse.ArgumentParser()
    p.add_argument("--model")
    p.add_argument("--host", default="127.0.0.1")
    p.add_argument("--port", type=int, default=8080)
    p.add_argument("--allowed-origins", type=lambda x: x.split(","), default="*")
    p.add_argument("--draft-model", default=None)
    p.add_argument("--adapter-path", default=None)
    p.add_argument("--trust-remote-code", action="store_true")
    p.add_argument("--log-level", default="INFO")
    p.add_argument("--prompt-cache-size", type=int, default=10)
    p.add_argument("--decode-concurrency", type=int, default=32)
    p.add_argument("--max-tokens", type=int, default=512)
    args = p.parse_args()
    run(args.host, args.port, ModelProvider(args))
`

// poisonedPackage is an mlx_lm a PYTHONPATH or the working directory could
// offer in place of the installed one. It must never be imported.
const poisonedPackage = `
import os
with open(os.environ["STANDIN_POISON_MARKER"], "w") as f:
    f.write("imported")
raise SystemExit(4)
`

// standInRuntime is a managed runtime whose interpreter is a real Python with
// the stand-in installed as its mlx_lm, the way the real runtime has the
// pinned one: in the virtualenv's own site-packages, the one place -I still
// imports from. Skips where no Python is available.
func standInRuntime(t *testing.T) config.Paths {
	t.Helper()
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("no python3 on PATH to run the launcher with")
	}
	paths := config.NewPaths(t.TempDir())
	if out, err := exec.Command(python, "-m", "venv", "--without-pip", paths.Venv).CombinedOutput(); err != nil {
		t.Skipf("cannot make a virtualenv with %s: %v\n%s", python, err, out)
	}
	site, err := filepath.Glob(filepath.Join(paths.Venv, "lib", "python3*", "site-packages"))
	if err != nil || len(site) != 1 {
		t.Fatalf("no site-packages in the virtualenv: %v %v", site, err)
	}
	pkg := filepath.Join(site[0], "mlx_lm")
	if err := os.MkdirAll(pkg, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, src := range map[string]string{"__init__.py": "", "server.py": standInServer} {
		if err := os.WriteFile(filepath.Join(pkg, name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(paths.Logs, 0o755); err != nil {
		t.Fatal(err)
	}
	return paths
}

// The launcher, run by a real interpreter against a stand-in server, binds
// its socket and nothing else, and answers there with the server's own
// handler — and that handler is reached only as Dessau allows: the request's
// model is the one the server was launched with whatever the request named,
// a request naming a draft model or adapters is refused, and no origin is
// allowed cross-origin access. Nothing on PYTHONPATH is imported in place of
// the installed server.
func TestTheLauncherServesOnlyOnItsPrivateSocket(t *testing.T) {
	paths := standInRuntime(t)
	marks := t.TempDir()
	tcpMarker := filepath.Join(marks, "tcp")
	poisonMarker := filepath.Join(marks, "poison")
	t.Setenv("STANDIN_TCP_MARKER", tcpMarker)
	t.Setenv("STANDIN_POISON_MARKER", poisonMarker)
	poison := filepath.Join(t.TempDir(), "mlx_lm")
	if err := os.MkdirAll(poison, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(poison, "__init__.py"), []byte(poisonedPackage), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PYTHONPATH", filepath.Dir(poison))

	model := plainModelDir(t)
	sock := privateSocket(t)
	l := &ExecLauncher{Paths: paths, LogDir: paths.Logs}
	p, err := l.Launch(context.Background(), Spec{RepoID: "org/name", ModelPath: model, Socket: sock})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), stopBound)
		defer cancel()
		_ = p.Stop(ctx)
	}()
	logOf := func() string {
		b, _ := os.ReadFile(filepath.Join(paths.Logs, logFileName("org/name")))
		return string(b)
	}

	c := &http.Client{Transport: childTransport(sock), Timeout: 5 * time.Second}
	deadline := time.Now().Add(20 * time.Second)
	for {
		resp, err := c.Get(childBaseURL + "/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				break
			}
		}
		select {
		case <-p.Done():
			t.Fatalf("the launcher exited before it answered (%v); its log:\n%s", p.Err(), logOf())
		default:
		}
		if time.Now().After(deadline) {
			t.Fatalf("the launcher never answered on its socket; its log:\n%s", logOf())
		}
		time.Sleep(50 * time.Millisecond)
	}

	post := func(body string, header http.Header) (int, map[string]any, http.Header) {
		t.Helper()
		req, err := http.NewRequest(http.MethodPost, childBaseURL+"/v1/chat/completions", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		for k, v := range header {
			req.Header[k] = v
		}
		resp, err := c.Do(req)
		if err != nil {
			t.Fatalf("POST %s: %v\nlog:\n%s", body, err, logOf())
		}
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		var got map[string]any
		_ = json.Unmarshal(raw, &got)
		return resp.StatusCode, got, resp.Header
	}

	// The model a request names is not what it gets.
	other := t.TempDir()
	code, got, hdr := post(`{"model":`+strconv.Quote(other)+`,"messages":[]}`,
		http.Header{"Origin": {"http://alice.example"}})
	if code != http.StatusOK {
		t.Fatalf("an ordinary request was answered %d: %v", code, got)
	}
	if got["model"] != model {
		t.Errorf("the server was handed model %v, want the launch path %q", got["model"], model)
	}
	if acao := hdr.Get("Access-Control-Allow-Origin"); acao != "" {
		t.Errorf("the server allows cross-origin access to %q", acao)
	}
	// A request that names no model gets the launch path too.
	if _, got, _ := post(`{"messages":[]}`, nil); got["model"] != model {
		t.Errorf("a request with no model was handed %v, want %q", got["model"], model)
	}

	// draft_model and adapters are refused, as the gateway refuses them, by
	// their presence whatever their value.
	for _, body := range []string{
		`{"model":"x","messages":[],"draft_model":` + strconv.Quote(other) + `}`,
		`{"model":"x","messages":[],"draft_model":null}`,
		`{"model":"x","messages":[],"adapters":` + strconv.Quote(other) + `}`,
	} {
		if code, got, _ := post(body, nil); code != http.StatusBadRequest {
			t.Errorf("%s was answered %d (%v), want 400", body, code, got)
		}
	}

	// A negative Content-Length is refused rather than handed on: the server
	// would read the body to the end of the stream, past every check.
	raw, err := net.Dial("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	body := `{"model":` + strconv.Quote(other) + `,"messages":[],"draft_model":` + strconv.Quote(other) + `}`
	fmt.Fprintf(raw, "POST /v1/chat/completions HTTP/1.1\r\nHost: x\r\nContent-Type: application/json\r\nContent-Length: -1\r\n\r\n%s", body)
	raw.(*net.UnixConn).CloseWrite()
	raw.SetReadDeadline(time.Now().Add(5 * time.Second))
	answer, _ := io.ReadAll(raw)
	raw.Close()
	if !bytes.HasPrefix(answer, []byte("HTTP/1.0 400")) {
		t.Errorf("a negative Content-Length was answered %q, want a 400", firstLine(answer))
	}

	// Nothing but the socket: the server's own TCP run() was never reached,
	// and the process holds no internet socket.
	if b, err := os.ReadFile(tcpMarker); err == nil {
		t.Errorf("the server's own TCP listener was started (%s)", b)
	}
	if b, err := os.ReadFile(poisonMarker); err == nil {
		t.Errorf("an mlx_lm from PYTHONPATH was imported (%s)", b)
	}
	if out, err := exec.Command("/usr/sbin/lsof", "-a", "-p", strconv.Itoa(p.Pid()), "-i").Output(); err == nil && len(bytes.TrimSpace(out)) > 0 {
		t.Errorf("the model server holds an internet socket:\n%s", out)
	}

	// Stopped, its socket goes with it.
	ctx, cancel := context.WithTimeout(context.Background(), stopBound)
	defer cancel()
	if err := p.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	<-p.Done()
	if _, err := os.Lstat(sock); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the socket outlived its model server: %v", err)
	}
}

// firstLine is the status line of a raw HTTP answer.
func firstLine(b []byte) string {
	line, _, _ := bytes.Cut(b, []byte("\r\n"))
	return string(line)
}

// The launcher refuses a socket outside a directory only this account can
// open, in Python as well as in Go: it is the last thing between a path and
// a bind, and the Go side is not the only thing that could start it.
func TestTheLauncherRefusesASocketInAnOpenDirectory(t *testing.T) {
	paths := standInRuntime(t)
	t.Setenv("STANDIN_TCP_MARKER", filepath.Join(t.TempDir(), "tcp"))
	open := filepath.Join(shortTempDir(t), "open")
	if err := os.Mkdir(open, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(open, 0o755); err != nil {
		t.Fatal(err)
	}
	out := runLauncherExpectingRefusal(t, paths, launchArgs(Spec{ModelPath: plainModelDir(t), Socket: filepath.Join(open, "m1")}))
	if !strings.Contains(out, "socket directory is open to other accounts") {
		t.Errorf("the refusal does not say why:\n%s", out)
	}
	if _, err := os.Lstat(filepath.Join(open, "m1")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("a socket was bound in the open directory: %v", err)
	}
}

// launcherRefusalBound is how long a launcher that must refuse is given to
// exit. Refusing is the interpreter starting and the launcher reading its
// arguments, well inside a second; a launcher still running at the bound went
// past the check that should have stopped it and is serving.
const launcherRefusalBound = 10 * time.Second

// runLauncherExpectingRefusal runs the launcher with argv and returns what it
// wrote. It fails the test when the launcher exits zero, when the refusal is
// not marked as the launcher's, and — at launcherRefusalBound rather than at
// the test binary's timeout — when the launcher does not exit at all.
func runLauncherExpectingRefusal(t *testing.T, paths config.Paths, argv []string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), launcherRefusalBound)
	defer cancel()
	cmd := exec.CommandContext(ctx, paths.VenvPython(), argv...)
	cmd.WaitDelay = time.Second
	raw, err := cmd.CombinedOutput()
	out := string(raw)
	if ctx.Err() != nil {
		t.Fatalf("the launcher was still running after %s: it did not refuse, and went on to serve; its output:\n%s", launcherRefusalBound, out)
	}
	if err == nil {
		t.Fatalf("the launcher exited 0 where it should have refused; its output:\n%s", out)
	}
	if !strings.Contains(out, "dessau-serve") {
		t.Errorf("the refusal does not say whose it is:\n%s", out)
	}
	return out
}

// A launch that names something for the server to load besides its model —
// a draft model, adapter weights — or that trusts a tokenizer's remote code is
// refused by the launcher itself, before it binds: the Go side never passes
// these flags, and the launcher is the last thing between them and load().
func TestTheLauncherRefusesALaunchThatLoadsSomethingElse(t *testing.T) {
	paths := standInRuntime(t)
	t.Setenv("STANDIN_TCP_MARKER", filepath.Join(t.TempDir(), "tcp"))
	other := t.TempDir()
	for _, c := range []struct {
		name   string
		flags  []string
		reason string
	}{
		{"a draft model", []string{"--draft-model", other}, "a draft model is not loaded"},
		{"adapter weights", []string{"--adapter-path", other}, "adapter weights are not loaded"},
		{"remote tokenizer code", []string{"--trust-remote-code"}, "remote tokenizer code is not trusted"},
	} {
		t.Run(c.name, func(t *testing.T) {
			sock := privateSocket(t)
			argv := append(launchArgs(Spec{ModelPath: plainModelDir(t), Socket: sock}), c.flags...)
			out := runLauncherExpectingRefusal(t, paths, argv)
			if !strings.Contains(out, c.reason) {
				t.Errorf("the refusal does not say %q:\n%s", c.reason, out)
			}
			if _, err := os.Lstat(sock); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("a socket was bound for a refused launch: %v", err)
			}
		})
	}
}

// Something under the socket's name that is not a socket is not the
// launcher's to remove: it refuses to start and leaves the file as it was,
// rather than unlinking whatever it finds and binding in its place.
func TestTheLauncherRefusesAFileInItsSocketsPlace(t *testing.T) {
	paths := standInRuntime(t)
	t.Setenv("STANDIN_TCP_MARKER", filepath.Join(t.TempDir(), "tcp"))
	sock := privateSocket(t)
	if err := os.WriteFile(sock, []byte("not a socket"), 0o600); err != nil {
		t.Fatal(err)
	}
	out := runLauncherExpectingRefusal(t, paths, launchArgs(Spec{ModelPath: plainModelDir(t), Socket: sock}))
	if !strings.Contains(out, "something other than a socket") {
		t.Errorf("the refusal does not say why:\n%s", out)
	}
	if b, err := os.ReadFile(sock); err != nil || string(b) != "not a socket" {
		t.Errorf("the file in the socket's place was not left as it was: %q, %v", b, err)
	}
}
