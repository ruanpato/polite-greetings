package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthcheck(t *testing.T) {
	for status, want := range map[int]int{http.StatusOK: 0, http.StatusServiceUnavailable: 1} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(status) }))
		if got := healthcheck(srv.URL); got != want {
			t.Errorf("server answering %d: exit code %d, want %d", status, got, want)
		}
		srv.Close()
	}
	if got := healthcheck("http://127.0.0.1:1/readyz"); got != 1 {
		t.Errorf("no server: exit code %d, want 1", got)
	}
}
