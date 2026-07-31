package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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

		url := workItemAddURL(wm.WorkspaceRootID())
		form := workItemAddForm(0, "New Work Item", 0)
		resp := ts.postForm(t, url, form)

		wantLocation := workspacesViewURL(wm.WorkspaceRootID())
		assertRedirect(t, resp, http.StatusSeeOther, wantLocation)

		resp = ts.get(t, wantLocation)
		assertStatusCode(t, http.StatusOK, resp.StatusCode)

		assertContains(t, resp.Body, "New Work Item")
	})

	t.Run("client errors", func(t *testing.T) {
		t.Run("invalid work item name", func(t *testing.T) {
			t.Chdir("../..")

			wm := model.NewWorkspaceModel(
				model.NewWorkspace(model.NewColumn("Backlog")),
			)

			app := newTestApplication(t, wm)
			ts := newTestServer(t, app.routes())
			defer ts.Close()

			url := workItemAddURL(wm.WorkspaceRootID())
			form := workItemAddForm(0, "    ", 0)

			resp := ts.postForm(t, url, form)
			wantLocation := workspacesViewURL(wm.WorkspaceRootID())
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

				url := workItemAddURL(wm.WorkspaceRootID())
				form := workItemAddForm(0, "New Work Item", 0)
				tt.mutate(form)

				resp := ts.postForm(t, url, form)
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
			model.NewWorkspace(model.NewColumn("Backlog", workItem1, workItem2)),
		)

		app := newTestApplication(t, wm)
		ts := newTestServer(t, app.routes())
		defer ts.Close()

		url := workItemDeleteURL(wm.WorkspaceRootID(), workItem1.ID())
		form := workItemDeleteForm(0)
		resp := ts.postForm(t, url, form)

		wantLocation := workspacesViewURL(wm.WorkspaceRootID())
		assertRedirect(t, resp, http.StatusSeeOther, wantLocation)

		resp = ts.get(t, wantLocation)
		assertStatusCode(t, http.StatusOK, resp.StatusCode)

		assertNotContains(t, resp.Body, workItem1Name)
		assertContains(t, resp.Body, workItem2Name)
	})

	t.Run("recursive deletion removes zoomed child workspaces", func(t *testing.T) {
		t.Chdir("../..")

		parentItemName := "Parent Item"
		parentItem := model.NewWorkItem(parentItemName)
		wm := model.NewWorkspaceModel(
			model.NewWorkspace(model.NewColumn("Backlog", parentItem)),
		)

		app := newTestApplication(t, wm)
		ts := newTestServer(t, app.routes())
		defer ts.Close()

		url := workItemZoomURL(wm.WorkspaceRootID(), parentItem.ID())
		form := workItemZoomForm(0)
		resp := ts.postForm(t, url, form)
		assertStatusCode(t, http.StatusSeeOther, resp.StatusCode)

		childLoc := resp.Header.Get("Location")
		childID := workspaceIDFromViewLocation(t, childLoc)

		childItemID, err := wm.WorkItemAdd(childID, 0, 0, "Child Item")
		if err != nil {
			t.Fatal(err)
		}

		url = workItemZoomURL(childID, childItemID)
		form = workItemZoomForm(1)
		resp = ts.postForm(t, url, form)
		assertStatusCode(t, http.StatusSeeOther, resp.StatusCode)

		childChildLoc := resp.Header.Get("Location")
		childChildID := workspaceIDFromViewLocation(t, childChildLoc)

		url = workItemDeleteURL(wm.WorkspaceRootID(), parentItem.ID())
		form = workItemDeleteForm(1)
		resp = ts.postForm(t, url, form)
		assertRedirect(t, resp, http.StatusSeeOther, workspacesViewURL(wm.WorkspaceRootID()))

		resp = ts.get(t, workspacesViewURL(wm.WorkspaceRootID()))
		assertStatusCode(t, http.StatusOK, resp.StatusCode)
		assertContains(t, resp.Body, `<h1>Root Workspace</h1>`)
		assertNotContains(t, resp.Body, parentItemName)

		resp = ts.get(t, workspacesViewURL(childID))
		assertStatusCode(t, http.StatusNotFound, resp.StatusCode)

		resp = ts.get(t, workspacesViewURL(childChildID))
		assertStatusCode(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("client error", func(t *testing.T) {
		t.Run("item id format", func(t *testing.T) {
			wm := model.NewWorkspaceModel(model.NewWorkspace())

			app := newTestApplication(t, wm)
			ts := newTestServer(t, app.routes())
			defer ts.Close()

			url := fmt.Sprintf("/workspaces/%s/work-items/invalid-format", wm.WorkspaceRootID())
			form := workItemDeleteForm(0)
			resp := ts.postForm(t, url, form)

			assertStatusCode(t, http.StatusUnprocessableEntity, resp.StatusCode)
		})

		t.Run("item not found", func(t *testing.T) {
			item := model.NewWorkItem("Item")
			unknownID := newUnknownWorkItemID(item.ID())

			wm := model.NewWorkspaceModel(model.NewWorkspace(model.NewColumn("Backlog", item)))

			app := newTestApplication(t, wm)
			ts := newTestServer(t, app.routes())
			defer ts.Close()

			url := workItemDeleteURL(wm.WorkspaceRootID(), unknownID)
			form := workItemDeleteForm(0)
			resp := ts.postForm(t, url, form)

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

				url := workItemDeleteURL(wm.WorkspaceRootID(), item.ID())
				form := workItemDeleteForm(0)
				tt.mutate(form)
				resp := ts.postForm(t, url, form)

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

		url := workItemMoveURL(wm.WorkspaceRootID(), itemA.ID())
		form := workItemMoveForm(model.DirectionDown, 0)
		resp := ts.postForm(t, url, form)

		wantLocation := workspacesViewURL(wm.WorkspaceRootID())
		assertRedirect(t, resp, http.StatusSeeOther, wantLocation)

		want := model.WorkspaceView{
			ID:       wm.WorkspaceRootID(),
			Title:    "Root Workspace",
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
			mutateURL  func(*model.WorkspaceModel, model.WorkItem) string
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
				mutateURL: func(wm *model.WorkspaceModel, item model.WorkItem) string {
					t.Helper()
					return fmt.Sprintf("/workspaces/%s/work-items/invalid-format/move", wm.WorkspaceRootID())
				},
				wantCode: http.StatusUnprocessableEntity,
			},
			{
				name: "item not found",
				mutateURL: func(wm *model.WorkspaceModel, item model.WorkItem) string {
					return workItemMoveURL(wm.WorkspaceRootID(), newUnknownWorkItemID(item.ID()))
				},
				wantCode: http.StatusNotFound,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				item := model.NewWorkItem("Item")
				wm := model.NewWorkspaceModel(
					model.NewWorkspace(model.NewColumn("Backlog", item)),
				)

				app := newTestApplication(t, wm)
				ts := newTestServer(t, app.routes())
				defer ts.Close()

				url := workItemMoveURL(wm.WorkspaceRootID(), item.ID())
				if tt.mutateURL != nil {
					url = tt.mutateURL(wm, item)
				}

				form := workItemMoveForm(model.DirectionDown, 0)
				if tt.mutateForm != nil {
					tt.mutateForm(form)
				}

				resp := ts.postForm(t, url, form)

				assertStatusCode(t, tt.wantCode, resp.StatusCode)

				resp = ts.get(t, workspacesViewURL(wm.WorkspaceRootID()))

				assertStatusCode(t, http.StatusOK, resp.StatusCode)
				assertContains(t, resp.Body, `name="revision" value="0"`)
			})
		}
	})
}

func TestWorkItemZoom(t *testing.T) {
	t.Run("zooming into work item twice returns same location", func(t *testing.T) {
		item := model.NewWorkItem("Item A")
		wm := model.NewWorkspaceModel(model.NewWorkspace(model.NewColumn("Column", item)))

		app := newTestApplication(t, wm)
		ts := newTestServer(t, app.routes())
		defer ts.Close()

		url := workItemZoomURL(wm.WorkspaceRootID(), item.ID())
		form := workItemZoomForm(0)
		resp := ts.postForm(t, url, form)

		assertStatusCode(t, http.StatusSeeOther, resp.StatusCode)

		firstLoc := resp.Header.Get("Location")
		if firstLoc == "" {
			t.Fatal("expected redirect location")
		}

		rootLoc := workspacesViewURL(wm.WorkspaceRootID())
		if firstLoc == rootLoc {
			t.Fatalf("expected redirect to child workspace, got root workspace %q", firstLoc)
		}

		url = workItemZoomURL(wm.WorkspaceRootID(), item.ID())
		form = workItemZoomForm(1)
		resp = ts.postForm(t, url, form)

		assertRedirect(t, resp, http.StatusSeeOther, firstLoc)
	})

	t.Run("new zoomed in workspace has revision 0", func(t *testing.T) {
		t.Chdir("../../")

		item := model.NewWorkItem("Item A")
		wm := model.NewWorkspaceModel(model.NewWorkspace(model.NewColumn("Column", item)))

		if _, err := wm.WorkItemAdd(wm.WorkspaceRootID(), 0, 0, "Bump Version"); err != nil {
			t.Fatal(err)
		}

		app := newTestApplication(t, wm)
		ts := newTestServer(t, app.routes())
		defer ts.Close()

		url := workItemZoomURL(wm.WorkspaceRootID(), item.ID())
		form := workItemZoomForm(1)
		resp := ts.postForm(t, url, form)

		assertStatusCode(t, http.StatusSeeOther, resp.StatusCode)
		loc := resp.Header.Get("Location")
		if loc == "" {
			t.Fatal("expected redirect location")
		}

		resp = ts.get(t, loc)

		assertStatusCode(t, http.StatusOK, resp.StatusCode)
		assertContains(t, resp.Body, `name="revision" value="0"`)
	})

	t.Run("invalid", func(t *testing.T) {
		tests := []struct {
			name       string
			mutateForm func(form url.Values)
			mutateURL  func(*model.WorkspaceModel, model.WorkItem) string
			wantCode   int
		}{
			{
				name: "workspace id format",
				mutateURL: func(wm *model.WorkspaceModel, wi model.WorkItem) string {
					return fmt.Sprintf("/workspaces/%s/work-items/%s/zoom", "invalid-id", wi.ID())
				},
				wantCode: http.StatusUnprocessableEntity,
			},
			{
				name: "item id format",
				mutateURL: func(wm *model.WorkspaceModel, wi model.WorkItem) string {
					return fmt.Sprintf("/workspaces/%s/work-items/%s/zoom", wm.WorkspaceRootID(), "invalid-id")
				},
				wantCode: http.StatusUnprocessableEntity,
			},
			{
				name: "revision format",
				mutateForm: func(form url.Values) {
					form.Set("revision", "invalid")
				},
				wantCode: http.StatusUnprocessableEntity,
			},
			{
				name: "work item not found",
				mutateURL: func(wm *model.WorkspaceModel, wi model.WorkItem) string {
					return workItemZoomURL(wm.WorkspaceRootID(), newUnknownWorkItemID(wi.ID()))
				},
				wantCode: http.StatusNotFound,
			},
			{
				name: "work space not found",
				mutateURL: func(wm *model.WorkspaceModel, wi model.WorkItem) string {
					return workItemZoomURL(newUnknownWorkspaceID(wm.WorkspaceRootID()), wi.ID())
				},
				wantCode: http.StatusNotFound,
			},
			{
				name: "revision conflict",
				mutateForm: func(form url.Values) {
					form.Set("revision", "999")
				},
				wantCode: http.StatusConflict,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				item := model.NewWorkItem("Item A")
				wm := model.NewWorkspaceModel(model.NewWorkspace(model.NewColumn("Column", item)))

				app := newTestApplication(t, wm)
				ts := newTestServer(t, app.routes())
				defer ts.Close()

				url := workItemZoomURL(wm.WorkspaceRootID(), item.ID())
				if tt.mutateURL != nil {
					url = tt.mutateURL(wm, item)
				}

				form := workItemZoomForm(0)
				if tt.mutateForm != nil {
					tt.mutateForm(form)
				}

				resp := ts.postForm(t, url, form)

				assertStatusCode(t, tt.wantCode, resp.StatusCode)
			})
		}
	})

	t.Run("child workspace contains link to parent", func(t *testing.T) {
		t.Chdir("../../")

		item := model.NewWorkItem("Item")
		wm := model.NewWorkspaceModel(
			model.NewWorkspace(model.NewColumn("Column", item)),
		)

		app := newTestApplication(t, wm)
		ts := newTestServer(t, app.routes())
		defer ts.Close()

		url := workItemZoomURL(wm.WorkspaceRootID(), item.ID())
		form := workItemZoomForm(0)
		resp := ts.postForm(t, url, form)

		assertStatusCode(t, http.StatusSeeOther, resp.StatusCode)

		loc := resp.Header.Get("Location")
		if loc == "" {
			t.Fatal("expected redirect location")
		}

		resp = ts.get(t, loc)

		assertStatusCode(t, http.StatusOK, resp.StatusCode)
		assertContains(t, resp.Body, fmt.Sprintf(`<a href="/workspaces/%s">Parent Workspace</a>`, wm.WorkspaceRootID()))
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
	t.Run("returns 200 OK", func(t *testing.T) {
		t.Chdir("../../")

		wm := model.NewWorkspaceModel(model.NewWorkspace())

		app := newTestApplication(t, wm)
		ts := newTestServer(t, app.routes())
		defer ts.Close()

		resp := ts.get(t, workspacesViewURL(wm.WorkspaceRootID()))
		assertStatusCode(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("root workspace title", func(t *testing.T) {
		t.Chdir("../../")

		wm := model.NewWorkspaceModel(model.NewWorkspace())

		app := newTestApplication(t, wm)
		ts := newTestServer(t, app.routes())
		defer ts.Close()

		resp := ts.get(t, workspacesViewURL(wm.WorkspaceRootID()))
		assertStatusCode(t, http.StatusOK, resp.StatusCode)
		assertContains(t, resp.Body, `<h1>Root Workspace</h1>`)

	})

	t.Run("child workspace title", func(t *testing.T) {
		t.Chdir("../../")

		itemName := "Item"
		item := model.NewWorkItem(itemName)
		wm := model.NewWorkspaceModel(
			model.NewWorkspace(model.NewColumn("Column", item)),
		)

		childID, err := wm.WorkItemZoom(wm.WorkspaceRootID(), item.ID(), 0)
		if err != nil {
			t.Fatal(err)
		}

		app := newTestApplication(t, wm)
		ts := newTestServer(t, app.routes())
		defer ts.Close()

		resp := ts.get(t, workspacesViewURL(childID))
		assertStatusCode(t, http.StatusOK, resp.StatusCode)

		want := fmt.Sprintf(`<h1>%s</h1>`, itemName)
		assertContains(t, resp.Body, want)

	})
}

func newUnknownWorkItemID(existing model.WorkItemID) model.WorkItemID {
	for {
		id := model.NewWorkItem("Not inserted").ID()
		if id != existing {
			return id
		}
	}
}

func newUnknownWorkspaceID(existing model.WorkspaceID) model.WorkspaceID {
	for {
		id := model.NewWorkspaceModel(model.NewWorkspace()).WorkspaceRootID()
		if id != existing {
			return id
		}
	}
}

func workItemAddURL(workspaceID model.WorkspaceID) string {
	return fmt.Sprintf("/workspaces/%s/work-items", workspaceID)
}

func workItemAddForm(columnIdx int, name string, revision uint64) url.Values {
	form := url.Values{}
	form.Set("revision", strconv.FormatUint(revision, 10))
	form.Set("columnIdx", strconv.Itoa(columnIdx))
	form.Set("name", name)
	return form
}

func workspacesViewURL(id model.WorkspaceID) string {
	return fmt.Sprintf("/workspaces/%s", id)
}

func workspaceIDFromViewLocation(t *testing.T, loc string) model.WorkspaceID {
	t.Helper()

	rawID, ok := strings.CutPrefix(loc, "/workspaces/")
	if !ok {
		t.Fatalf("expected workspace view location, got %q", loc)
	}

	id, err := model.ParseWorkspaceID(rawID)
	if err != nil {
		t.Fatal(err)
	}

	return id
}

func workItemDeleteURL(workspaceID model.WorkspaceID, itemID model.WorkItemID) string {
	return fmt.Sprintf("/workspaces/%s/work-items/%s", workspaceID, itemID)
}

func workItemDeleteForm(revision uint64) url.Values {
	form := url.Values{}
	form.Set("_method", "DELETE")
	form.Set("revision", strconv.FormatUint(revision, 10))
	return form
}

func workItemMoveURL(workspaceID model.WorkspaceID, itemID model.WorkItemID) string {
	return fmt.Sprintf("/workspaces/%s/work-items/%s/move", workspaceID, itemID)
}

func workItemMoveForm(direction model.MoveDirection, revision uint64) url.Values {
	form := url.Values{}
	form.Set("_method", "PATCH")
	form.Set("revision", strconv.FormatUint(revision, 10))
	form.Set("direction", string(direction))
	return form
}

func workItemZoomURL(workspaceID model.WorkspaceID, itemID model.WorkItemID) string {
	return fmt.Sprintf("/workspaces/%s/work-items/%s/zoom", workspaceID, itemID)
}

func workItemZoomForm(revision uint64) url.Values {
	form := url.Values{}
	form.Set("revision", strconv.FormatUint(revision, 10))
	return form
}
