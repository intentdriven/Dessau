package main

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
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
