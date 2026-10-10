package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	kenyageo "github.com/rmiyoyo/kenya-geo"
	"github.com/rmiyoyo/kenya-geo/api"
)

func TestDefaultAddr(t *testing.T) {
	t.Setenv("PORT", "")
	if got := defaultAddr(); got != ":8080" {
		t.Errorf("without PORT: %q", got)
	}
	t.Setenv("PORT", "9000")
	if got := defaultAddr(); got != ":9000" {
		t.Errorf("with PORT=9000: %q", got)
	}
}

func TestCheck(t *testing.T) {
	srv := httptest.NewServer(api.New(kenyageo.Default()))
	defer srv.Close()
	if err := check(strings.TrimPrefix(srv.URL, "http://")); err != nil {
		t.Errorf("healthy server: %v", err)
	}

	broken := httptest.NewServer(http.NotFoundHandler())
	defer broken.Close()
	if err := check(strings.TrimPrefix(broken.URL, "http://")); err == nil {
		t.Error("server answering 404 passed the check")
	}

	addr := broken.Listener.Addr().String()
	broken.Close()
	if err := check(addr); err == nil {
		t.Error("closed server passed the check")
	}
	if err := check("no-port"); err == nil {
		t.Error("bad address passed the check")
	}
}
