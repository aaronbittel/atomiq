package main

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/aaronbittel/atomiq/internal/model"
)

func TestWorkItemPost(t *testing.T) {
	t.Run("valid work item", func(t *testing.T) {
		t.Chdir("../..")

		workspaceModel := &model.WorkspaceModel{
			Workspace: model.Workspace{
				Columns: []model.Column{
					{Name: "Backlog"},
				},
			},
		}

		app := newTestApplication(t, workspaceModel)
		ts := newTestServer(t, app.routes())
		defer ts.Close()

		form := url.Values{}
		form.Set("columnIdx", "0")
		form.Set("name", "New Work Item")

		resp := ts.postForm(t, "/work-item", form)
		assertRedirect(t, resp, http.StatusSeeOther, "/")

		resp = ts.get(t, "/")
		assertStatusCode(t, http.StatusOK, resp.StatusCode)

		if !strings.Contains(resp.Body, "New Work Item") {
			t.Errorf("expected \"New Work Item\" in html")
		}
	})

	t.Run("blank work item name", func(t *testing.T) {
		t.Chdir("../..")

		workspaceModel := &model.WorkspaceModel{
			Workspace: model.Workspace{
				Columns: []model.Column{
					{Name: "Backlog"},
				},
			},
		}

		app := newTestApplication(t, workspaceModel)
		ts := newTestServer(t, app.routes())
		defer ts.Close()

		form := url.Values{}
		form.Set("columnIdx", "0")
		form.Set("name", "   ")

		resp := ts.postForm(t, "/work-item", form)
		assertRedirect(t, resp, http.StatusSeeOther, "/")

		resp = ts.get(t, "/")
		assertStatusCode(t, http.StatusOK, resp.StatusCode)

		if !strings.Contains(resp.Body, "<h2>Backlog</h2>") {
			t.Fatal("expected \"<h2>Backlog</h2>\" column to exist")
		}

		if strings.Contains(resp.Body, "class=\"work-item-box\"") {
			t.Fatal("unexpected work item div")
		}

		if !strings.Contains(resp.Body, "div class=\"error\"") {
			t.Error("expected div class=\"error\"")
		}

		if !strings.Contains(resp.Body, "<span>work item must not be blank</span>") {
			t.Error("expected \"<span>work item must not be blank</span>\"")
		}
	})
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

	app := newTestApplication(t, workspaceModel)
	ts := newTestServer(t, app.routes())
	defer ts.Close()

	form := url.Values{}
	form.Set("columnIdx", "0")
	form.Set("_method", "DELETE")

	resp := ts.postForm(t, "/work-item/"+workItem1.ID, form)
	assertRedirect(t, resp, http.StatusSeeOther, "/")

	resp = ts.get(t, "/")
	assertStatusCode(t, http.StatusOK, resp.StatusCode)

	if strings.Contains(resp.Body, "Todo 1") {
		t.Error("Todo 1 should have been deleted")
	}

	if !strings.Contains(resp.Body, "Todo 2") {
		t.Errorf("expected \"Todo 2\" to still exists")
	}
}
