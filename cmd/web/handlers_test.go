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

		wm := model.NewWorkspaceModel(
			model.NewWorkspace(
				model.NewColumn("Backlog"),
			),
		)

		app := newTestApplication(t, wm)
		app.logger = slog.New(slog.NewTextHandler(t.Output(), nil))
		ts := newTestServer(t, app.routes())
		defer ts.Close()

		form := url.Values{}
		form.Set("columnIdx", "0")
		form.Set("name", "New Work Item")
		form.Set("revision", "0")

		resp := ts.postForm(t, fmt.Sprintf("/workspaces/%s/work-items", wm.WorkspaceRootID()), form)
		wantLocation := "/workspaces/" + wm.WorkspaceRootID().String()
		assertRedirect(t, resp, http.StatusSeeOther, wantLocation)

		resp = ts.get(t, wantLocation)
		assertStatusCode(t, http.StatusOK, resp.StatusCode)

		assertContains(t, resp.Body, "New Work Item")
	})

	t.Run("client errors", func(t *testing.T) {
		t.Run("invalid work item name", func(t *testing.T) {
			t.Chdir("../..")

			wm := model.NewWorkspaceModel(
				model.NewWorkspace(
					model.NewColumn("Backlog"),
				),
			)

			app := newTestApplication(t, wm)
			ts := newTestServer(t, app.routes())
			defer ts.Close()

			form := url.Values{}
			form.Set("columnIdx", "0")
			form.Set("name", "   ")
			form.Set("revision", "0")

			resp := ts.postForm(t, fmt.Sprintf("/workspaces/%s/work-items", wm.WorkspaceRootID()), form)
			wantLocation := "/workspaces/" + wm.WorkspaceRootID().String()
			assertRedirect(t, resp, http.StatusSeeOther, wantLocation)

			resp = ts.get(t, wantLocation)
			assertStatusCode(t, http.StatusOK, resp.StatusCode)

			assertContains(t, resp.Body, "Backlog")
			assertContains(t, resp.Body, "div class=\"error\"")
			assertContains(t, resp.Body, "<span>work item must not be blank</span>")
			assertNotContains(t, resp.Body, "class=\"work-item-box\"")

		})

		tests := []struct {
			name     string
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
				wm := model.NewWorkspaceModel(model.NewWorkspace())
				app := newTestApplication(t, wm)
				ts := newTestServer(t, app.routes())
				defer ts.Close()

				form := url.Values{}
				form.Set("columnIdx", "0")
				form.Set("name", "New Work Item")
				form.Set("revision", "0")

				tt.mutate(form)

				resp := ts.postForm(t, fmt.Sprintf("/workspaces/%s/work-items", wm.WorkspaceRootID()), form)
				assertStatusCode(t, tt.wantCode, resp.StatusCode)
			})
		}
	})
}

func TestWorkItemDelete(t *testing.T) {
	t.Run("valid work item deletion", func(t *testing.T) {
		t.Chdir("../..")

		const (
			workItem1Name = "Todo 1"
			workItem2Name = "Todo 2"
		)

		var (
			workItem1 = model.NewWorkItem(workItem1Name)
			workItem2 = model.NewWorkItem(workItem2Name)
		)

		wm := model.NewWorkspaceModel(
			model.NewWorkspace(
				model.NewColumn("Backlog", workItem1, workItem2),
			),
		)

		app := newTestApplication(t, wm)
		ts := newTestServer(t, app.routes())
		defer ts.Close()

		form := url.Values{}
		form.Set("_method", "DELETE")
		form.Set("revision", "0")

		resp := ts.postForm(t, fmt.Sprintf("/workspaces/%s/work-items/%s", wm.WorkspaceRootID(), workItem1.ID()), form)
		wantLocation := "/workspaces/" + wm.WorkspaceRootID().String()
		assertRedirect(t, resp, http.StatusSeeOther, wantLocation)

		resp = ts.get(t, wantLocation)
		assertStatusCode(t, http.StatusOK, resp.StatusCode)

		assertNotContains(t, resp.Body, workItem1Name)
		assertContains(t, resp.Body, workItem2Name)
	})

	t.Run("client error", func(t *testing.T) {
		t.Run("item id format", func(t *testing.T) {
			wm := model.NewWorkspaceModel(model.NewWorkspace())
			app := newTestApplication(t, wm)
			ts := newTestServer(t, app.routes())
			defer ts.Close()

			form := url.Values{}
			form.Set("_method", "DELETE")
			form.Set("revision", "0")

			resp := ts.postForm(t, fmt.Sprintf("/workspaces/%s/work-items/invalid-format", wm.WorkspaceRootID()), form)
			assertStatusCode(t, http.StatusUnprocessableEntity, resp.StatusCode)
		})

		t.Run("item not found", func(t *testing.T) {
			item := model.NewWorkItem("Item")
			unknownID := newUnknownWorkItemID(item.ID())

			wm := model.NewWorkspaceModel(model.NewWorkspace(model.NewColumn("Backlog", item)))
			app := newTestApplication(t, wm)
			ts := newTestServer(t, app.routes())
			defer ts.Close()

			form := url.Values{}
			form.Set("_method", "DELETE")
			form.Set("revision", "0")

			resp := ts.postForm(t, fmt.Sprintf("/workspaces%s/work-items/%s", wm.WorkspaceRootID(), unknownID), form)

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
				wm := model.NewWorkspaceModel(model.NewWorkspace(
					model.NewColumn("Backlog", item),
				))

				app := newTestApplication(t, wm)
				ts := newTestServer(t, app.routes())
				defer ts.Close()

				form := url.Values{}
				form.Set("_method", "DELETE")
				form.Set("revision", "0")

				tt.mutate(form)

				resp := ts.postForm(t, fmt.Sprintf("/workspaces/%s/work-items/%s", wm.WorkspaceRootID(), item.ID()), form)
				assertStatusCode(t, tt.wantCode, resp.StatusCode)
			})
		}
	})
}

func TestWorkItemMove(t *testing.T) {
	t.Run("valid move", func(t *testing.T) {
		t.Chdir("../../")

		const (
			nameA = "A"
			nameB = "B"
		)

		var (
			itemA = model.NewWorkItem(nameA)
			itemB = model.NewWorkItem(nameB)
			viewA = model.WorkItemView{ID: itemA.ID(), Name: nameA}
			viewB = model.WorkItemView{ID: itemB.ID(), Name: nameB}
		)

		wm := model.NewWorkspaceModel(model.NewWorkspace(
			model.NewColumn("Backlog", itemA, itemB),
		))

		app := newTestApplication(t, wm)
		ts := newTestServer(t, app.routes())
		defer ts.Close()

		form := url.Values{}
		form.Set("_method", "PATCH")
		form.Set("revision", "0")
		form.Set("direction", "down")

		resp := ts.postForm(t, fmt.Sprintf("/workspaces/%s/work-items/%s/move", wm.WorkspaceRootID(), itemA.ID()), form)
		wantLocation := "/workspaces/" + wm.WorkspaceRootID().String()

		assertRedirect(t, resp, http.StatusSeeOther, wantLocation)

		want := model.WorkspaceView{
			Revision: 1,
			Columns: []model.ColumnView{
				{
					Name:      "Backlog",
					WorkItems: []model.WorkItemView{viewB, viewA},
				},
			},
		}
		got, err := wm.WorkspaceView(wm.WorkspaceRootID())
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("workspace view mismatch (-want +got):\n%s", diff)
		}

		resp = ts.get(t, wantLocation)

		assertStatusCode(t, http.StatusOK, resp.StatusCode)
		assertContains(t, resp.Body, `name="revision" value="1"`)
	})

	t.Run("client error", func(t *testing.T) {
		t.Chdir("../../")

		tests := []struct {
			name       string
			mutateUrl  func(*model.WorkspaceModel, model.WorkItem) string
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
				name: "invalid id format",
				mutateUrl: func(wm *model.WorkspaceModel, item model.WorkItem) string {
					t.Helper()
					return fmt.Sprintf("/workspaces/%s/work-items/invalid-format/move", wm.WorkspaceRootID())
				},
				wantCode: http.StatusUnprocessableEntity,
			},
			{
				name: "item not found",
				mutateUrl: func(wm *model.WorkspaceModel, item model.WorkItem) string {
					return fmt.Sprintf("/workspaces/%s/work-items/%s/move", wm.WorkspaceRootID(), newUnknownWorkItemID(item.ID()))
				},
				wantCode: http.StatusNotFound,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				item := model.NewWorkItem("Item")
				wm := model.NewWorkspaceModel(
					model.NewWorkspace(
						model.NewColumn("Backlog", item),
					),
				)

				app := newTestApplication(t, wm)
				ts := newTestServer(t, app.routes())
				defer ts.Close()

				form := url.Values{}
				form.Set("_method", "PATCH")
				form.Set("direction", "down")
				form.Set("revision", "0")
				if tt.mutateForm != nil {
					tt.mutateForm(form)
				}

				url := fmt.Sprintf("/workspaces/%s/work-items/%s/move", wm.WorkspaceRootID(), item.ID())
				if tt.mutateUrl != nil {
					url = tt.mutateUrl(wm, item)
				}

				resp := ts.postForm(t, url, form)
				assertStatusCode(t, tt.wantCode, resp.StatusCode)

				resp = ts.get(t, "/workspaces/"+wm.WorkspaceRootID().String())

				assertStatusCode(t, http.StatusOK, resp.StatusCode)
				assertContains(t, resp.Body, `name="revision" value="0"`)
			})
		}
	})
}

func TestHome(t *testing.T) {
	t.Run("redirects to workspace root", func(t *testing.T) {
		wm := model.NewWorkspaceModel(model.NewWorkspace())

		app := newTestApplication(t, wm)
		ts := newTestServer(t, app.routes())
		defer ts.Close()

		resp := ts.get(t, "/")
		assertRedirect(t, resp, http.StatusSeeOther, "/workspaces/")
	})
}

func TestWorkspaceRoot(t *testing.T) {
	t.Run("redirects to root workspace", func(t *testing.T) {
		wm := model.NewWorkspaceModel(model.NewWorkspace())

		app := newTestApplication(t, wm)
		ts := newTestServer(t, app.routes())
		defer ts.Close()

		resp := ts.get(t, "/workspaces/")
		assertRedirect(t, resp, http.StatusSeeOther, "/workspaces/"+wm.WorkspaceRootID().String())
	})
}

func TestWorkspaceView(t *testing.T) {
	t.Chdir("../../")

	wm := model.NewWorkspaceModel(model.NewWorkspace())

	app := newTestApplication(t, wm)
	ts := newTestServer(t, app.routes())
	defer ts.Close()

	resp := ts.get(t, "/workspaces/"+wm.WorkspaceRootID().String())
	assertStatusCode(t, http.StatusOK, resp.StatusCode)
}

func newUnknownWorkItemID(existing model.WorkItemID) model.WorkItemID {
	for {
		id := model.NewWorkItem("Not inserted").ID()
		if id != existing {
			return id
		}
	}
}
