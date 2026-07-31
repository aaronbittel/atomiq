package model_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/aaronbittel/atomiq/internal/model"
	"github.com/google/go-cmp/cmp"
)

const (
	itemAName = "A"
	itemBName = "B"
)

var (
	itemA = model.NewWorkItem(itemAName)
	itemB = model.NewWorkItem(itemBName)
	viewA = model.WorkItemView{ID: itemA.ID(), Name: itemAName}
	viewB = model.WorkItemView{ID: itemB.ID(), Name: itemBName}
)

func TestWorkspaceView(t *testing.T) {
	t.Run("view returns snapshot", func(t *testing.T) {
		wm := model.NewWorkspaceModel(model.NewWorkspace(model.NewColumn("Column", itemA)))

		view, err := wm.WorkspaceView(wm.WorkspaceRootID())
		if err != nil {
			t.Fatal(err)
		}
		view.Columns[0].Name = "Mutated"
		view.Columns[0].WorkItems[0].Name = "Mutated"

		got, err := wm.WorkspaceView(wm.WorkspaceRootID())
		if err != nil {
			t.Fatal(err)
		}
		want := workspaceRootView(wm.WorkspaceRootID(), 0, columnView("Column", viewA))

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("workspace view mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("view parent id is snapshot", func(t *testing.T) {
		item := model.NewWorkItem("Item")
		wm := model.NewWorkspaceModel(model.NewWorkspace(model.NewColumn("Column", item)))

		childID, err := wm.WorkItemZoom(wm.WorkspaceRootID(), item.ID(), 0)
		if err != nil {
			t.Fatal(err)
		}

		view, err := wm.WorkspaceView(childID)
		if err != nil {
			t.Fatal(err)
		}
		if view.ParentID == nil {
			t.Fatal("expected parent id")
		}

		*view.ParentID = "mutated"

		assertDefaultChildWorkspace(t, wm, childID, wm.WorkspaceRootID(), 0)
	})

	t.Run("child workspace has parentID", func(t *testing.T) {
		item := model.NewWorkItem("Item")
		wm := model.NewWorkspaceModel(model.NewWorkspace(model.NewColumn("Column", item)))

		childID, err := wm.WorkItemZoom(wm.WorkspaceRootID(), item.ID(), 0)
		if err != nil {
			t.Fatal(err)
		}

		assertDefaultChildWorkspace(t, wm, childID, wm.WorkspaceRootID(), 0)
	})

	t.Run("child of child", func(t *testing.T) {
		item := model.NewWorkItem("Item")

		wm := model.NewWorkspaceModel(model.NewWorkspace(model.NewColumn("Column", item)))

		childID, err := wm.WorkItemZoom(wm.WorkspaceRootID(), item.ID(), 0)
		if err != nil {
			t.Fatal(err)
		}

		childItemID, err := wm.WorkItemAdd(childID, 0, 0, "Child Item")
		if err != nil {
			t.Fatal(err)
		}

		childChildID, err := wm.WorkItemZoom(childID, childItemID, 1)
		if err != nil {
			t.Fatal(err)
		}

		assertDefaultChildWorkspace(t, wm, childChildID, childID, 0)
	})
}

func TestWorkItemAdd(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		wm := model.NewWorkspaceModel(model.NewWorkspace(model.NewColumn("Column")))

		itemID, err := wm.WorkItemAdd(wm.WorkspaceRootID(), 0, 0, "New item")
		if err != nil {
			t.Fatal(err)
		}

		want := workspaceRootView(wm.WorkspaceRootID(), 1,
			columnView("Column", model.WorkItemView{
				ID:   itemID,
				Name: "New item",
			}))

		got, err := wm.WorkspaceView(wm.WorkspaceRootID())
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("workspace view mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("invalid column index", func(t *testing.T) {
		wm := model.NewWorkspaceModel(model.NewWorkspace(model.NewColumn("Column")))

		wantErr := model.ErrInvalidPosition

		if _, err := wm.WorkItemAdd(wm.WorkspaceRootID(), 0, 1, "invalid column"); !errors.Is(err, wantErr) {
			t.Fatalf("expected %v, got %v", wantErr, err)
		}
	})
}

func TestWorkItemDelete(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		wm := model.NewWorkspaceModel(model.NewWorkspace(model.NewColumn("Column", itemA)))

		if err := wm.WorkItemDelete(wm.WorkspaceRootID(), 0, itemA.ID()); err != nil {
			t.Fatal(err)
		}

		want := workspaceRootView(wm.WorkspaceRootID(), 1, columnView("Column"))
		got, err := wm.WorkspaceView(wm.WorkspaceRootID())
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("workspace view mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("invalid item id", func(t *testing.T) {
		wm := model.NewWorkspaceModel(model.NewWorkspace(model.NewColumn("Column", itemA)))

		wantErr := model.ErrWorkItemNotFound
		err := wm.WorkItemDelete(wm.WorkspaceRootID(), 0, itemB.ID())

		if !errors.Is(err, wantErr) {
			t.Fatalf("expected %v, got %v", wantErr, err)
		}
	})
}

func TestWorkItemMoveDirection(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		wm := model.NewWorkspaceModel(model.NewWorkspace(model.NewColumn("Column", itemA, itemB)))

		if err := wm.WorkItemMoveDirection(wm.WorkspaceRootID(), 0, itemB.ID(), model.DirectionUp); err != nil {
			t.Fatal(err)
		}

		want := workspaceRootView(wm.WorkspaceRootID(), 1, columnView("Column", viewB, viewA))
		got, err := wm.WorkspaceView(wm.WorkspaceRootID())
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("workspace view mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("no-op move does not update revision", func(t *testing.T) {
		wm := model.NewWorkspaceModel(model.NewWorkspace(model.NewColumn("Column", itemA)))

		if err := wm.WorkItemMoveDirection(wm.WorkspaceRootID(), 0, itemA.ID(), model.DirectionUp); err != nil {
			t.Fatal(err)
		}

		want := workspaceRootView(wm.WorkspaceRootID(), 0, columnView("Column", model.WorkItemView{ID: itemA.ID(), Name: itemAName}))
		got, err := wm.WorkspaceView(wm.WorkspaceRootID())
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("workspace view mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("revision conflict", func(t *testing.T) {
		wm := model.NewWorkspaceModel(model.NewWorkspace(model.NewColumn("Column", itemA)))

		wantErr := model.ErrRevisionConflict
		if err := wm.WorkItemMoveDirection(wm.WorkspaceRootID(), 1, itemA.ID(), model.DirectionUp); !errors.Is(err, wantErr) {
			t.Fatalf("expected err %v, got %v", wantErr, err)
		}

		want := workspaceRootView(wm.WorkspaceRootID(), 0, columnView("Column", viewA))
		got, err := wm.WorkspaceView(wm.WorkspaceRootID())
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("workspace view mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestWorkItemMovePosition(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		wm := model.NewWorkspaceModel(model.NewWorkspace(model.NewColumn("Column", itemA, itemB)))

		err := wm.WorkItemMovePosition(wm.WorkspaceRootID(), 0, itemB.ID(), model.WorkItemInsertionPoint{
			ColumnIdx: 0,
			ItemIdx:   0,
		})
		if err != nil {
			t.Fatal(err)
		}

		want := workspaceRootView(wm.WorkspaceRootID(), 1, columnView("Column", viewB, viewA))
		got, err := wm.WorkspaceView(wm.WorkspaceRootID())
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("workspace view mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("no-op move does not update revision", func(t *testing.T) {
		wm := model.NewWorkspaceModel(model.NewWorkspace(model.NewColumn("Column", itemA, itemB)))

		err := wm.WorkItemMovePosition(wm.WorkspaceRootID(), 0, itemB.ID(), model.WorkItemInsertionPoint{
			ColumnIdx: 0,
			ItemIdx:   1,
		})
		if err != nil {
			t.Fatal(err)
		}

		want := workspaceRootView(wm.WorkspaceRootID(), 0, columnView("Column", viewA, viewB))
		got, err := wm.WorkspaceView(wm.WorkspaceRootID())
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("workspace view mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("move to following insertion index does not update revision", func(t *testing.T) {
		wm := model.NewWorkspaceModel(model.NewWorkspace(model.NewColumn("Column", itemA, itemB)))

		err := wm.WorkItemMovePosition(wm.WorkspaceRootID(), 0, itemB.ID(), model.WorkItemInsertionPoint{
			ColumnIdx: 0,
			ItemIdx:   2,
		})
		if err != nil {
			t.Fatal(err)
		}

		want := workspaceRootView(wm.WorkspaceRootID(), 0, columnView("Column", viewA, viewB))
		got, err := wm.WorkspaceView(wm.WorkspaceRootID())
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("workspace view mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("revision conflict", func(t *testing.T) {
		wm := model.NewWorkspaceModel(model.NewWorkspace(model.NewColumn("Column", itemA, itemB)))

		wantErr := model.ErrRevisionConflict
		err := wm.WorkItemMovePosition(wm.WorkspaceRootID(), 1, itemB.ID(), model.WorkItemInsertionPoint{
			ColumnIdx: 0,
			ItemIdx:   0,
		})

		if !errors.Is(err, wantErr) {
			t.Fatalf("expected err %v, got %v", wantErr, err)
		}

		want := workspaceRootView(wm.WorkspaceRootID(), 0, columnView("Column", viewA, viewB))
		got, err := wm.WorkspaceView(wm.WorkspaceRootID())
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("workspace view mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestWorkItemZoom(t *testing.T) {
	t.Run("first zoom creates default workspace with revision 0", func(t *testing.T) {
		wm := model.NewWorkspaceModel(model.NewWorkspace(
			model.NewColumn("Backlog", itemA),
		))

		childID, err := wm.WorkItemZoom(wm.WorkspaceRootID(), itemA.ID(), 0)
		if err != nil {
			t.Fatal(err)
		}

		assertDefaultChildWorkspace(t, wm, childID, wm.WorkspaceRootID(), 0)
	})

	t.Run("returns the same child workspace id", func(t *testing.T) {
		wm := model.NewWorkspaceModel(model.NewWorkspace(
			model.NewColumn("Backlog", itemA),
		))

		first, err := wm.WorkItemZoom(wm.WorkspaceRootID(), itemA.ID(), 0)
		if err != nil {
			t.Fatal(err)
		}

		second, err := wm.WorkItemZoom(wm.WorkspaceRootID(), itemA.ID(), 1)
		if err != nil {
			t.Fatal(err)
		}

		if first != second {
			t.Fatalf("zoom calls returned different child workspace ids: %q and %q", first, second)
		}
	})

	t.Run("existing child workspace does not increment revision", func(t *testing.T) {
		wm := model.NewWorkspaceModel(model.NewWorkspace(
			model.NewColumn("Backlog", itemA),
		))

		if _, err := wm.WorkItemZoom(wm.WorkspaceRootID(), itemA.ID(), 0); err != nil {
			t.Fatal(err)
		}

		if _, err := wm.WorkItemZoom(wm.WorkspaceRootID(), itemA.ID(), 1); err != nil {
			t.Fatal(err)
		}

		if _, err := wm.WorkItemZoom(wm.WorkspaceRootID(), itemA.ID(), 1); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("work item id not found", func(t *testing.T) {
		wm := model.NewWorkspaceModel(model.NewWorkspace(
			model.NewColumn("Backlog", itemA),
		))

		wantErr := model.ErrWorkItemNotFound

		if _, err := wm.WorkItemZoom(wm.WorkspaceRootID(), itemB.ID(), 0); !errors.Is(err, wantErr) {
			t.Fatalf("expected error %v, got %v", wantErr, err)
		}
	})

	t.Run("updating child workspace does not modify parent revision", func(t *testing.T) {
		wm := model.NewWorkspaceModel(model.NewWorkspace(
			model.NewColumn("Backlog", itemA),
		))

		childID, err := wm.WorkItemZoom(wm.WorkspaceRootID(), itemA.ID(), 0)
		if err != nil {
			t.Fatal(err)
		}

		itemID, err := wm.WorkItemAdd(childID, 0, 0, "New Child Item")
		if err != nil {
			t.Fatal(err)
		}

		wantRootView := workspaceRootView(wm.WorkspaceRootID(), 1, columnView("Backlog", model.WorkItemView{
			ID:   itemA.ID(),
			Name: itemAName,
		}))

		rootView, err := wm.WorkspaceView(wm.WorkspaceRootID())
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(wantRootView, rootView); diff != "" {
			t.Errorf("workspace view mismatch (-want +got):\n%s", diff)
		}

		wantChildView := workspaceChildView(childID, wm.WorkspaceRootID(), 1,
			columnView("Backlog", model.WorkItemView{
				ID:   itemID,
				Name: "New Child Item",
			}),
			columnView("In Progress"),
			columnView("Done"),
		)

		childView, err := wm.WorkspaceView(childID)
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(wantChildView, childView); diff != "" {
			t.Errorf("workspace view mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestSequentialMutationsUseLatestRevision(t *testing.T) {
	wm := model.NewWorkspaceModel(model.NewWorkspace(model.NewColumn("Column", itemA, itemB)))

	if err := wm.WorkItemMoveDirection(wm.WorkspaceRootID(), 0, itemA.ID(), model.DirectionDown); err != nil {
		t.Fatal(err)
	}

	want := workspaceRootView(wm.WorkspaceRootID(), 1, columnView("Column", viewB, viewA))
	got, err := wm.WorkspaceView(wm.WorkspaceRootID())
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("workspace view mismatch (-want +got):\n%s", diff)
	}

	if err := wm.WorkItemDelete(wm.WorkspaceRootID(), 1, itemB.ID()); err != nil {
		t.Fatal(err)
	}

	want = workspaceRootView(wm.WorkspaceRootID(), 2, columnView("Column", viewA))
	got, err = wm.WorkspaceView(wm.WorkspaceRootID())
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("workspace view mismatch (-want +got):\n%s", diff)
	}
}

func TestWorkspaceViewHasNoUnexpectedReferenceFields(t *testing.T) {
	viewType := reflect.TypeOf(model.WorkspaceView{})

	allowedReferenceFields := map[string]bool{
		"Columns":  true, // explicitly tested by "view returns snapshot"
		"ParentID": true, // explicitly tested by "view parent id is snapshot"
	}

	for i := range viewType.NumField() {
		field := viewType.Field(i)

		switch field.Type.Kind() {
		case reflect.Pointer, reflect.Slice, reflect.Map:
			if !allowedReferenceFields[field.Name] {
				t.Fatalf("WorkspaceView.%s is a reference field; add a snapshot test or allow it here", field.Name)
			}
		}
	}
}

func assertDefaultChildWorkspace(t *testing.T, wm *model.WorkspaceModel, id, parentID model.WorkspaceID, revision uint64) {
	want := workspaceChildView(id, parentID, revision,
		columnView("Backlog"),
		columnView("In Progress"),
		columnView("Done"),
	)

	got, err := wm.WorkspaceView(id)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("default workspace view mismatch (-want +got):\n%s", diff)
	}
}
