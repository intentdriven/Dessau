package selftest

import (
	"net/http"
	"testing"
	"time"
)

// A request to a model server goes through the transport the pool handed
// over, which dials the server's private socket. With none, the request is
// refused rather than sent through the client's own transport, which would
// look the placeholder host up and go wherever that led.
func TestAnUpstreamClientNeedsTheUpstreamsTransport(t *testing.T) {
	base := &http.Client{Timeout: time.Minute}
	if _, err := UpstreamClient(base, nil); err == nil {
		t.Fatal("UpstreamClient accepted an Upstream with no transport")
	}
	rt := &http.Transport{}
	c, err := UpstreamClient(base, rt)
	if err != nil {
		t.Fatal(err)
	}
	if c.Transport != rt {
		t.Error("the client does not use the Upstream's transport")
	}
	if c.Timeout != base.Timeout {
		t.Error("the client lost the caller's own settings")
	}
	if base.Transport != nil {
		t.Error("the caller's client was changed")
	}
}
