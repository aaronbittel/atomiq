package main

import (
	"log/slog"
	"net/http"
	"net/url"
	"testing"

	"github.com/aaronbittel/atomiq/internal/model"
)

func TestWorkItemPost(t *testing.T) {
	t.Run("valid work item", func(t *testing.T) {
		t.Chdir("../..")

		workspaceModel := model.NewWorkspaceModel(
			model.Workspace{
				Columns: []model.Column{
					{Name: "Backlog"},
				},
			},
		)

		app := newTestApplication(t, workspaceModel)
		app.logger = slog.New(slog.NewTextHandler(t.Output(), nil))
		ts := newTestServer(t, app.routes())
		defer ts.Close()

		form := url.Values{}
		form.Set("columnIdx", "0")
		form.Set("name", "New Work Item")

		resp := ts.postForm(t, "/work-item", form)
		assertRedirect(t, resp, http.StatusSeeOther, "/")

		resp = ts.get(t, "/")
		assertStatusCode(t, http.StatusOK, resp.StatusCode)

		assertContains(t, resp.Body, "New Work Item")
	})

	t.Run("blank work item name", func(t *testing.T) {
		t.Chdir("../..")

		workspaceModel := model.NewWorkspaceModel(
			model.Workspace{
				Columns: []model.Column{
					{Name: "Backlog"},
				},
			},
		)

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

		assertContains(t, resp.Body, "Backlog")
		assertContains(t, resp.Body, "div class=\"error\"")
		assertContains(t, resp.Body, "<span>work item must not be blank</span>")
		assertNotContains(t, resp.Body, "class=\"work-item-box\"")

	})

	t.Run("invalid columnIdx access", func(t *testing.T) {
		app := newTestApplication(t, &model.WorkspaceModel{})
		ts := newTestServer(t, app.routes())
		defer ts.Close()

		form := url.Values{}
		form.Set("columnIdx", "1")
		form.Set("name", "New Work Item")

		resp := ts.postForm(t, "/work-item", form)
		assertStatusCode(t, http.StatusUnprocessableEntity, resp.StatusCode)
	})

	t.Run("columnIdx not an integer", func(t *testing.T) {
		app := newTestApplication(t, &model.WorkspaceModel{})
		ts := newTestServer(t, app.routes())
		defer ts.Close()

		form := url.Values{}
		form.Set("columnIdx", "abc")
		form.Set("name", "New Work Item")

		resp := ts.postForm(t, "/work-item", form)
		assertStatusCode(t, http.StatusUnprocessableEntity, resp.StatusCode)
	})
}

func TestWorkItemDelete(t *testing.T) {
	t.Run("valid work item deletion", func(t *testing.T) {
		t.Chdir("../..")

		workItem1 := model.NewWorkItem("Todo 1")

		workspaceModel := model.NewWorkspaceModel(
			model.Workspace{
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
		)

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

		assertNotContains(t, resp.Body, "Todo 1")
		assertContains(t, resp.Body, "Todo 2")
	})

	tests := []struct {
		name      string
		columnIdx string
		url       string
	}{
		{
			name:      "path id too short",
			columnIdx: "0",
			url:       "/work-item/short",
		},
		{
			name:      "columnIdx not a number",
			columnIdx: "abc",
			url:       "/work-item/12345678",
		},
		{
			name:      "invalid columnIdx access",
			columnIdx: "1",
			url:       "/work-item/12345678",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApplication(t, &model.WorkspaceModel{})
			ts := newTestServer(t, app.routes())
			defer ts.Close()

			form := url.Values{}
			form.Set("columnIdx", tt.columnIdx)
			form.Set("_method", "DELETE")

			resp := ts.postForm(t, tt.url, form)
			assertStatusCode(t, http.StatusUnprocessableEntity, resp.StatusCode)
		})
	}
}
