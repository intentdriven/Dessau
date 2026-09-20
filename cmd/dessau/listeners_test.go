package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// serveOn stands a listener server up on loopback and hands back its address.
// It is the http.Server main builds, a short read bound aside.
func serveOn(t *testing.T, h http.Handler, read time.Duration) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := listenerServer(h, read)
	go srv.Serve(ln)
	t.Cleanup(func() { srv.Close() })
	return ln.Addr().String()
}

// A peer that sends headers and then stalls the body is cut off within the
// bound (iss-2609190226050845).
//
// The bytes are bounded already — the pairing endpoint reads through a
// LimitReader, the gateway through a MaxBytesReader — but a bound in bytes
// says nothing about time, and a client that promises 4 KiB and delivers one
// byte holds a goroutine and a connection for as long as it keeps the socket
// open. The endpoint that matters here asks for no credential.
func TestAStalledRequestBodyIsCutOffWithinTheBound(t *testing.T) {
	const bound = 250 * time.Millisecond

	read := make(chan error, 1)
	addr := serveOn(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		read <- err
	}), bound)

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	// Headers that promise a body, then one byte of it and silence.
	if _, err := fmt.Fprint(conn,
		"POST /pair HTTP/1.1\r\nHost: localhost\r\n"+
			"Content-Type: application/json\r\nContent-Length: 4096\r\n\r\n{"); err != nil {
		t.Fatal(err)
	}

	select {
	case err := <-read:
		if err == nil {
			t.Fatal("the handler read a whole body out of a client that never sent one")
		}
		// The deadline and not some other refusal: os.ErrDeadlineExceeded is
		// the branch internal/gateway answers 408 on, and an error of another
		// kind would have it answering 400 for a client that merely ran out of
		// time.
		if !errors.Is(err, os.ErrDeadlineExceeded) {
			t.Errorf("the stalled body ended in %v, want a deadline: the connection is being "+
				"dropped for some other reason and the client is told the wrong thing", err)
		}
	case <-time.After(20 * bound):
		t.Fatalf("a client that sent headers and then stalled was still holding a goroutine %s later: "+
			"the body is bounded in bytes and not in time", 20*bound)
	}
}

// The read bound is not a bound on the answer. A completion streams for as
// long as the model takes, and what covered the request read has to be gone by
// the time the handler is answering.
//
// The request's CONTEXT is what this asserts, because that is what the gateway
// runs the generation under: it carries the upstream request to the model and
// the streaming loop checks it. A read deadline still armed while the handler
// works puts net/http's background read into a timeout, and a background read
// that errors cancels this context — so a read bound left covering the answer
// would not cut the bytes off, it would cancel the generation behind them.
// net/http clears the deadline when it starts that background read, which is
// as soon as the body has been read to the end, and this is the test that says
// so out loud.
func TestALongAnswerOutlivesTheReadBound(t *testing.T) {
	const bound = 150 * time.Millisecond

	addr := serveOn(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.ReadAll(r.Body); err != nil {
			t.Errorf("the body of a request delivered at once could not be read: %v", err)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		for _, chunk := range []string{"one", "two", "three"} {
			select {
			case <-r.Context().Done():
				t.Errorf("the request was cancelled while the answer was still being produced: %v",
					r.Context().Err())
				return
			case <-time.After(2 * bound):
			}
			if _, err := io.WriteString(w, chunk); err != nil {
				t.Errorf("a streamed chunk could not be written %s after the body was read: %v",
					2*bound, err)
				return
			}
			w.(http.Flusher).Flush()
		}
	}), bound)

	resp, err := http.Post("http://"+addr+"/v1/chat/completions", "application/json",
		bytes.NewReader([]byte(`{"stream":true}`)))
	if err != nil {
		t.Fatalf("a request whose answer takes longer than the read bound failed: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("a streamed answer was cut off: %v", err)
	}
	if string(body) != "onetwothree" {
		t.Errorf("the streamed answer arrived as %q, want %q", body, "onetwothree")
	}
}

// The same, for a request that carries no body at all — the control panel's
// event stream, which is a GET that is held open for as long as a dashboard
// tab is. There is nothing to read, so net/http starts its background read
// straight away and the deadline goes with it.
func TestARequestWithNoBodyStreamsPastTheReadBound(t *testing.T) {
	const bound = 150 * time.Millisecond

	addr := serveOn(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
			t.Errorf("a bodiless request was cancelled before it could be answered: %v", r.Context().Err())
			return
		case <-time.After(3 * bound):
		}
		if _, err := io.WriteString(w, "served"); err != nil {
			t.Errorf("a bodiless request could not be answered %s in: %v", 3*bound, err)
		}
	}), bound)

	resp, err := http.Get("http://" + addr + "/api/events")
	if err != nil {
		t.Fatalf("a bodiless request whose answer takes longer than the read bound failed: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("a bodiless request's answer was cut off: %v", err)
	}
	if string(body) != "served" {
		t.Errorf("the answer arrived as %q, want %q", body, "served")
	}
}

// Both listeners are built through listenerServer, and the bound has a value.
//
// listenerServer is a constructor a test can call with any figure it likes,
// which makes every test above true of a server this process might never
// build. Reverting either call site in main.go to an http.Server literal —
// which is where it was — leaves the build, the vet and the whole suite green
// while the control is gone (iss-2609190254372265). So the wiring is held
// here, in the shape cmd/dessau/tlsbind_test.go already uses for the branch
// that mounts pairing.
func TestBothListenersAreBuiltThroughListenerServer(t *testing.T) {
	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	body := string(src)
	for _, want := range []string{
		"srv := listenerServer(withLogging(mux, log), requestReadTimeout)",
		"tlsSrv = listenerServer(withLogging(g.TLSHandler(reg), log), requestReadTimeout)",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("main.go no longer builds a listener through listenerServer: %q is gone, "+
				"so the bounds are whatever that call site now writes", want)
		}
	}
	// An http.Server literal in main.go is how the bounds drifted apart in the
	// first place: two literals, and a bound added to one of them.
	if strings.Contains(body, "&http.Server{") {
		t.Error("main.go builds an http.Server by hand again, so a bound put on one listener " +
			"is no longer a bound on the other")
	}
}

// The bounds themselves, on the server listenerServer actually returns. The
// read bound is set and the write bound is not, and neither of those is a
// detail: one of them is the fix and the other would cut a generation off.
func TestTheListenerBoundsAreTheOnesThisServerMeansToHave(t *testing.T) {
	if requestReadTimeout != 30*time.Second {
		t.Errorf("the request read bound is %s, want 30s — the figure internal/gateway already "+
			"puts on the completions body", requestReadTimeout)
	}
	srv := listenerServer(http.NotFoundHandler(), requestReadTimeout)
	if srv.ReadTimeout != requestReadTimeout {
		t.Errorf("the listener's ReadTimeout is %s, want %s: a request body is bounded in bytes "+
			"and not in time again", srv.ReadTimeout, requestReadTimeout)
	}
	if srv.ReadHeaderTimeout != headerReadTimeout || srv.IdleTimeout != idleTimeout {
		t.Errorf("the listener's header and idle bounds are %s and %s, want %s and %s",
			srv.ReadHeaderTimeout, srv.IdleTimeout, headerReadTimeout, idleTimeout)
	}
	if srv.WriteTimeout != 0 {
		t.Errorf("the listener has a WriteTimeout of %s: a completion streams for as long as the "+
			"model answers for, and that is a bound on how long a model may think", srv.WriteTimeout)
	}
}
