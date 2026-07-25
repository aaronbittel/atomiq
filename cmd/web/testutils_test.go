package main

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/aaronbittel/atomiq/internal/model"
	"github.com/alexedwards/scs/v2"
)

func newTestApplication(t *testing.T, workspaceModel *model.WorkspaceModel) *application {
	return &application{
		workspaceModel: workspaceModel,
		logger:         slog.New(slog.DiscardHandler),
		sessionManager: scs.New(),
	}
}

type testServer struct {
	*httptest.Server
}

func newTestServer(t *testing.T, handler http.Handler) *testServer {
	s := httptest.NewServer(handler)

	s.Client().CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	s.Client().Jar = jar

	return &testServer{s}
}

type testResponse struct {
	Header     http.Header
	Cookies    []*http.Cookie
	Body       string
	StatusCode int
}

func (ts *testServer) get(t *testing.T, url string) testResponse {
	req, err := http.NewRequest(http.MethodGet, ts.URL+url, nil)
	if err != nil {
		t.Fatal(err)
	}

	return ts.do(t, req)
}

func (ts *testServer) postForm(t *testing.T, url string, form url.Values) testResponse {
	req, err := http.NewRequest(http.MethodPost, ts.URL+url, strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return ts.do(t, req)
}

func assertStatusCode(t *testing.T, want, got int) {
	if want != got {
		t.Fatalf("expected %q, got %q", http.StatusText(want), http.StatusText(got))
	}
}

func assertRedirect(t *testing.T, resp testResponse, wantCode int, wantLocation string) {
	assertStatusCode(t, wantCode, resp.StatusCode)

	gotLocation := resp.Header.Get("Location")
	if wantLocation != gotLocation {
		t.Fatalf("expected location header to be %q, got %q", wantLocation, gotLocation)
	}
}

func (ts *testServer) do(t *testing.T, req *http.Request) testResponse {
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	return testResponse{
		Header:     resp.Header,
		Cookies:    resp.Cookies(),
		Body:       string(bytes.TrimSpace(body)),
		StatusCode: resp.StatusCode,
	}
}
