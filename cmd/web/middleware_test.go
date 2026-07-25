package main

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestMethodOverride(t *testing.T) {
	form := url.Values{}
	form.Set("_method", "DELETE")

	req := httptest.NewRequest(
		http.MethodPost,
		"/work-item/abcdefgh",
		strings.NewReader(form.Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()

	var gotMethod string
	called := false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
	})

	app := application{logger: slog.New(slog.DiscardHandler)}

	app.methodOverride(next).ServeHTTP(rr, req)

	if !called {
		t.Fatal("next handler was not called")
	}

	if gotMethod != http.MethodDelete {
		t.Fatalf("got method %q, want %q", gotMethod, http.MethodDelete)
	}

	if rr.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestMethodOverrideKeepsPostWithoutDeleteOverride(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/work-item", strings.NewReader("name=test"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()

	var gotMethod string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
	})

	app := application{logger: slog.New(slog.DiscardHandler)}
	app.methodOverride(next).ServeHTTP(rr, req)

	if gotMethod != http.MethodPost {
		t.Fatalf("got method %q, want %q", gotMethod, http.MethodPost)
	}
}
