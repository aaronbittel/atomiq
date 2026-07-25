package main

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/aaronbittel/atomiq/internal/model"
	"github.com/alexedwards/scs/v2"
)

func TestWorkItemPost(t *testing.T) {
	t.Chdir("../..")

	workspaceModel := &model.WorkspaceModel{
		Workspace: model.Workspace{
			Columns: []model.Column{
				{Name: "Backlog"},
			},
		},
	}

	app := application{
		workspaceModel: workspaceModel,
		logger:         slog.New(slog.DiscardHandler),
		sessionManager: scs.New(),
	}

	s := httptest.NewServer(app.routes())
	defer s.Close()

	form := url.Values{}
	form.Set("columnIdx", "0")
	form.Set("name", "New Work Item")

	s.Client().CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	resp, err := s.Client().Post(s.URL+"/work-item", "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("expected %q, got %q", http.StatusText(http.StatusSeeOther), http.StatusText(resp.StatusCode))
	}

	resp, err = s.Client().Get(s.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected %q, got %q", http.StatusText(http.StatusOK), http.StatusText(resp.StatusCode))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)

	if !strings.Contains(got, "New Work Item") {
		t.Errorf("expected \"New Work Item\" in html")
	}
}

func TestWorkItemDelete(t *testing.T) {
	t.Chdir("../..")

	workItem1 := model.NewWorkItem("Todo 1")

	workspaceModel := &model.WorkspaceModel{
		Workspace: model.Workspace{
			Columns: []model.Column{
				{
					Name: "Backlog",
					WorkItems: []model.WorkItem{
						workItem1,
						model.NewWorkItem("Todo 2"),
					},
				},
			},
		},
	}

	app := application{
		workspaceModel: workspaceModel,
		logger:         slog.New(slog.NewTextHandler(t.Output(), nil)),
		sessionManager: scs.New(),
	}

	s := httptest.NewServer(app.routes())
	defer s.Close()

	form := url.Values{}
	form.Set("columnIdx", "0")
	form.Set("_method", "DELETE")

	s.Client().CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	resp, err := s.Client().Post(fmt.Sprintf("%s/work-item/%s", s.URL, workItem1.ID), "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("expected %q, got %q", http.StatusText(http.StatusSeeOther), http.StatusText(resp.StatusCode))
	}

	resp, err = s.Client().Get(s.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected %q, got %q", http.StatusText(http.StatusOK), http.StatusText(resp.StatusCode))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)

	if strings.Contains(got, "Todo 1") {
		t.Error("Todo 1 should have been deleted")
	}

	if !strings.Contains(got, "Todo 2") {
		t.Errorf("expected \"Todo 2\" to still exists")
	}
}
