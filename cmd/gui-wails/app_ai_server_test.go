package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAIServerReachableDefaultEmptyUsesProbe(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer srv.Close()

	if !aiServerReachable(srv.URL) {
		t.Fatal("expected reachable when /v1/models returns 200")
	}
	if aiServerReachable("http://127.0.0.1:1") {
		t.Fatal("expected unreachable on closed port")
	}
}

func TestAIServerReachableRejectsNonOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	if aiServerReachable(srv.URL) {
		t.Fatal("503 must count as unavailable")
	}
}
