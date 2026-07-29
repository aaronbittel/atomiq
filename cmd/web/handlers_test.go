package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"testing"

	"github.com/aaronbittel/atomiq/internal/model"
	"github.com/google/go-cmp/cmp"
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
		form.Set("revision", "0")

		resp := ts.postForm(t, "/work-item", form)
		assertRedirect(t, resp, http.StatusSeeOther, "/")

		resp = ts.get(t, "/")
		assertStatusCode(t, http.StatusOK, resp.StatusCode)

		assertContains(t, resp.Body, "New Work Item")
	})

	t.Run("client errors", func(t *testing.T) {
		t.Run("invalid work item name", func(t *testing.T) {
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
			form.Set("revision", "0")

			resp := ts.postForm(t, "/work-item", form)
			assertRedirect(t, resp, http.StatusSeeOther, "/")

			resp = ts.get(t, "/")
			assertStatusCode(t, http.StatusOK, resp.StatusCode)

			assertContains(t, resp.Body, "Backlog")
			assertContains(t, resp.Body, "div class=\"error\"")
			assertContains(t, resp.Body, "<span>work item must not be blank</span>")
			assertNotContains(t, resp.Body, "class=\"work-item-box\"")

		})

		tests := []struct {
			name     string
			ws       model.Workspace
			mutate   func(form url.Values)
			wantCode int
		}{
			{
				name: "invalid columnIdx access",
				mutate: func(form url.Values) {
					form.Set("columnIdx", "1")
				},
				wantCode: http.StatusUnprocessableEntity,
			},
			{
				name: "columnIdx not a number",
				mutate: func(form url.Values) {
					form.Set("columnIdx", "not-a-number")
				},
				wantCode: http.StatusUnprocessableEntity,
			},
			{
				name: "revision conflict",
				mutate: func(form url.Values) {
					form.Set("revision", "1")
				},
				wantCode: http.StatusConflict,
			},
			{
				name: "revision not a number",
				mutate: func(form url.Values) {
					form.Set("revision", "not-a-number")
				},
				wantCode: http.StatusUnprocessableEntity,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				app := newTestApplication(t, model.NewWorkspaceModel(tt.ws))
				ts := newTestServer(t, app.routes())
				defer ts.Close()

				form := url.Values{}
				form.Set("columnIdx", "0")
				form.Set("name", "New Work Item")
				form.Set("revision", "0")

				tt.mutate(form)

				resp := ts.postForm(t, "/work-item", form)
				assertStatusCode(t, tt.wantCode, resp.StatusCode)
			})
		}
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
		form.Set("_method", "DELETE")
		form.Set("revision", "0")

		resp := ts.postForm(t, "/work-item/"+workItem1.ID.String(), form)
		assertRedirect(t, resp, http.StatusSeeOther, "/")

		resp = ts.get(t, "/")
		assertStatusCode(t, http.StatusOK, resp.StatusCode)

		assertNotContains(t, resp.Body, "Todo 1")
		assertContains(t, resp.Body, "Todo 2")
	})

	t.Run("client error", func(t *testing.T) {
		t.Run("item id format", func(t *testing.T) {
			app := newTestApplication(t, &model.WorkspaceModel{})
			ts := newTestServer(t, app.routes())
			defer ts.Close()

			form := url.Values{}
			form.Set("_method", "DELETE")
			form.Set("revision", "0")

			resp := ts.postForm(t, "/work-item/invalid-format", form)
			assertStatusCode(t, http.StatusUnprocessableEntity, resp.StatusCode)
		})

		t.Run("item not found", func(t *testing.T) {
			item := model.NewWorkItem("Item")
			unknownID := newUnknownWorkItemID(t, item.ID)

			ws := model.Workspace{
				Columns: []model.Column{
					{
						Name:      "Backlog",
						WorkItems: []model.WorkItem{item},
					},
				},
			}

			app := newTestApplication(t, model.NewWorkspaceModel(ws))
			ts := newTestServer(t, app.routes())
			defer ts.Close()

			form := url.Values{}
			form.Set("_method", "DELETE")
			form.Set("revision", "0")

			resp := ts.postForm(t, "/work-item/"+unknownID.String(), form)

			assertStatusCode(t, http.StatusNotFound, resp.StatusCode)
		})

		tests := []struct {
			name     string
			mutate   func(form url.Values)
			wantCode int
		}{
			{
				name: "revision not a number",
				mutate: func(form url.Values) {
					form.Set("revision", "not a number")
				},
				wantCode: http.StatusUnprocessableEntity,
			},
			{
				name: "revision conflict",
				mutate: func(form url.Values) {
					form.Set("revision", "1")
				},
				wantCode: http.StatusConflict,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				item := model.NewWorkItem("item")

				ws := model.Workspace{
					Columns: []model.Column{
						{
							Name:      "Backlog",
							WorkItems: []model.WorkItem{item},
						},
					},
				}

				app := newTestApplication(t, model.NewWorkspaceModel(ws))
				ts := newTestServer(t, app.routes())
				defer ts.Close()

				form := url.Values{}
				form.Set("_method", "DELETE")
				form.Set("revision", "0")

				tt.mutate(form)

				resp := ts.postForm(t, "/work-item/"+item.ID.String(), form)
				assertStatusCode(t, tt.wantCode, resp.StatusCode)
			})
		}
	})
}

func TestWorkItemMove(t *testing.T) {
	t.Run("valid move", func(t *testing.T) {
		t.Chdir("../../")

		A := model.NewWorkItem("A")
		B := model.NewWorkItem("B")

		ws := model.Workspace{
			Columns: []model.Column{
				{
					Name:      "Backlog",
					WorkItems: []model.WorkItem{A, B},
				},
			},
		}
		wm := model.NewWorkspaceModel(ws)

		app := newTestApplication(t, wm)
		ts := newTestServer(t, app.routes())
		defer ts.Close()

		form := url.Values{}
		form.Set("_method", "PATCH")
		form.Set("revision", "0")
		form.Set("direction", "down")

		resp := ts.postForm(t, fmt.Sprintf("/work-item/%s/move", A.ID), form)
		assertRedirect(t, resp, http.StatusSeeOther, "/")

		want := model.WorkspaceView{
			Revision: 1,
			Columns: []model.ColumnView{
				{
					Name: "Backlog",
					WorkItems: []model.WorkItemView{
						{
							ID:   B.ID,
							Name: B.Name,
						},
						{
							ID:   A.ID,
							Name: A.Name,
						},
					},
				},
			},
		}
		got := wm.WorkspaceView()

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("workspace view mismatch (-want +got):\n%s", diff)
		}

		resp = ts.get(t, "/")

		assertStatusCode(t, http.StatusOK, resp.StatusCode)
		assertContains(t, resp.Body, `name="revision" value="1"`)
	})

	t.Run("client error", func(t *testing.T) {
		t.Chdir("../../")

		item := model.NewWorkItem("Item")

		tests := []struct {
			name       string
			url        string
			mutateForm func(from url.Values)
			wantCode   int
		}{
			{
				name: "invalid direction",
				mutateForm: func(form url.Values) {
					form.Set("direction", "invalid")
				},
				wantCode: http.StatusUnprocessableEntity,
			},
			{
				name: "revision not a number",
				mutateForm: func(form url.Values) {
					form.Set("revision", "invalid")
				},
				wantCode: http.StatusUnprocessableEntity,
			},
			{
				name: "revision conflict",
				mutateForm: func(form url.Values) {
					form.Set("revision", "1")
				},
				wantCode: http.StatusConflict,
			},
			{
				name:     "invalid id format",
				url:      "/work-item/invalid-format/move",
				wantCode: http.StatusUnprocessableEntity,
			},
			{
				name:     "item not found",
				url:      fmt.Sprintf("/work-item/%s/move", newUnknownWorkItemID(t, item.ID)),
				wantCode: http.StatusNotFound,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				ws := model.Workspace{
					Columns: []model.Column{
						{
							Name:      "Backlog",
							WorkItems: []model.WorkItem{item},
						},
					},
				}

				app := newTestApplication(t, model.NewWorkspaceModel(ws))
				ts := newTestServer(t, app.routes())
				defer ts.Close()

				form := url.Values{}
				form.Set("_method", "PATCH")
				form.Set("direction", "down")
				form.Set("revision", "0")
				if tt.mutateForm != nil {
					tt.mutateForm(form)
				}

				url := fmt.Sprintf("/work-item/%s/move", item.ID)
				if tt.url != "" {
					url = tt.url
				}

				resp := ts.postForm(t, url, form)
				assertStatusCode(t, tt.wantCode, resp.StatusCode)

				resp = ts.get(t, "/")

				assertStatusCode(t, http.StatusOK, resp.StatusCode)
				assertContains(t, resp.Body, `name="revision" value="0"`)
			})
		}
	})
}

func newUnknownWorkItemID(t *testing.T, existing model.WorkItemID) model.WorkItemID {
	t.Helper()

	for {
		id := model.NewWorkItem("Not inserted").ID
		if id != existing {
			return id
		}
	}
}
