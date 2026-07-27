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
	for _, method := range []string{http.MethodDelete, http.MethodPatch} {
		t.Run(method, func(t *testing.T) {
			form := url.Values{}
			form.Set("_method", method)

			req := httptest.NewRequest(
				http.MethodPost,
				"/work-item/abcdefgh",
				strings.NewReader(form.Encode()),
			)
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			rr := httptest.NewRecorder()

			var got string
			called := false

			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				got = r.Method
				w.WriteHeader(http.StatusOK)
			})

			app := application{logger: slog.New(slog.DiscardHandler)}

			app.methodOverride(next).ServeHTTP(rr, req)

			if !called {
				t.Fatal("next handler was not called")
			}

			if got != method {
				t.Fatalf("got method %q, want %q", got, method)
			}

			assertStatusCode(t, http.StatusOK, rr.Code)
		})
	}

	t.Run("get", func(t *testing.T) {
		form := url.Values{}
		form.Set("_method", "GET")

		req := httptest.NewRequest(
			http.MethodPost,
			"/work-item/abcdefgh",
			strings.NewReader(form.Encode()),
		)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		rr := httptest.NewRecorder()

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

		app := application{logger: slog.New(slog.DiscardHandler)}
		app.methodOverride(next).ServeHTTP(rr, req)

		assertStatusCode(t, http.StatusUnprocessableEntity, rr.Code)
	})
}

func TestMethodOverrideKeepsPostWithoutOverride(t *testing.T) {
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
