package model_test

import (
	"errors"
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

		view := wm.WorkspaceView()
		view.Columns[0].Name = "Mutated"
		view.Columns[0].WorkItems[0].Name = "Mutated"

		got := wm.WorkspaceView()
		want := workspaceView(0, columnView("Column", viewA))

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("workspace view mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestWorkItemAdd(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		wm := model.NewWorkspaceModel(model.NewWorkspace(model.NewColumn("Column")))

		if err := wm.WorkItemAdd(0, 0, "New item"); err != nil {
			t.Fatal(err)
		}

		got := wm.WorkspaceView()

		if got.Revision != 1 {
			t.Fatalf("expected revision to be 1, got %d", got.Revision)
		}

		if len(got.Columns[0].WorkItems) != 1 {
			t.Fatalf("expected one item, got %d", len(got.Columns[0].WorkItems))
		}
		if got.Columns[0].WorkItems[0].Name != "New item" {
			t.Fatalf("expected trimmed item name")
		}
	})

	t.Run("invalid column index", func(t *testing.T) {
		wm := model.NewWorkspaceModel(model.NewWorkspace(model.NewColumn("Column")))

		wantErr := model.ErrInvalidPosition

		if err := wm.WorkItemAdd(0, 1, "invalid column"); !errors.Is(err, wantErr) {
			t.Fatalf("expected %v, got %v", wantErr, err)
		}
	})
}

func TestWorkItemDelete(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		wm := model.NewWorkspaceModel(model.NewWorkspace(model.NewColumn("Column", itemA)))

		if err := wm.WorkItemDelete(0, itemA.ID()); err != nil {
			t.Fatal(err)
		}

		want := workspaceView(1, columnView("Column"))
		got := wm.WorkspaceView()

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("workspace view mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("invalid item id", func(t *testing.T) {
		wm := model.NewWorkspaceModel(model.NewWorkspace(model.NewColumn("Column", itemA)))

		wantErr := model.ErrWorkItemNotFound
		err := wm.WorkItemDelete(0, itemB.ID())

		if !errors.Is(err, wantErr) {
			t.Fatalf("expected %v, got %v", wantErr, err)
		}
	})
}

func TestWorkItemMoveDirection(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		wm := model.NewWorkspaceModel(model.NewWorkspace(model.NewColumn("Column", itemA, itemB)))

		if err := wm.WorkItemMoveDirection(0, itemB.ID(), model.DirectionUp); err != nil {
			t.Fatal(err)
		}

		want := workspaceView(1, columnView("Column", viewB, viewA))
		got := wm.WorkspaceView()

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("workspace view mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("no-op move does not update revision", func(t *testing.T) {
		wm := model.NewWorkspaceModel(model.NewWorkspace(model.NewColumn("Column", itemA)))

		if err := wm.WorkItemMoveDirection(0, itemA.ID(), model.DirectionUp); err != nil {
			t.Fatal(err)
		}

		want := workspaceView(0, columnView("Column", model.WorkItemView{ID: itemA.ID(), Name: itemAName}))
		got := wm.WorkspaceView()

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("workspace view mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("revision conflict", func(t *testing.T) {
		wm := model.NewWorkspaceModel(model.NewWorkspace(model.NewColumn("Column", itemA)))

		wantErr := model.ErrRevisionConflict
		if err := wm.WorkItemMoveDirection(1, itemA.ID(), model.DirectionUp); !errors.Is(err, wantErr) {
			t.Fatalf("expected err %v, got %v", wantErr, err)
		}

		want := workspaceView(0, columnView("Column", viewA))
		got := wm.WorkspaceView()

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("workspace view mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestWorkItemMovePosition(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		wm := model.NewWorkspaceModel(model.NewWorkspace(model.NewColumn("Column", itemA, itemB)))

		err := wm.WorkItemMovePosition(0, itemB.ID(), model.WorkItemInsertionPoint{
			ColumnIdx: 0,
			ItemIdx:   0,
		})
		if err != nil {
			t.Fatal(err)
		}

		want := workspaceView(1, columnView("Column", viewB, viewA))
		got := wm.WorkspaceView()

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("workspace view mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("no-op move does not update revision", func(t *testing.T) {
		wm := model.NewWorkspaceModel(model.NewWorkspace(model.NewColumn("Column", itemA, itemB)))

		err := wm.WorkItemMovePosition(0, itemB.ID(), model.WorkItemInsertionPoint{
			ColumnIdx: 0,
			ItemIdx:   1,
		})
		if err != nil {
			t.Fatal(err)
		}

		want := workspaceView(0, columnView("Column", viewA, viewB))
		got := wm.WorkspaceView()

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("workspace view mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("no-op move does not update revision 2", func(t *testing.T) {
		wm := model.NewWorkspaceModel(model.NewWorkspace(model.NewColumn("Column", itemA, itemB)))

		err := wm.WorkItemMovePosition(0, itemB.ID(), model.WorkItemInsertionPoint{
			ColumnIdx: 0,
			ItemIdx:   2,
		})
		if err != nil {
			t.Fatal(err)
		}

		want := workspaceView(0, columnView("Column", viewA, viewB))
		got := wm.WorkspaceView()

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("workspace view mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("revision conflict", func(t *testing.T) {
		wm := model.NewWorkspaceModel(model.NewWorkspace(model.NewColumn("Column", itemA, itemB)))

		wantErr := model.ErrRevisionConflict
		err := wm.WorkItemMovePosition(1, itemB.ID(), model.WorkItemInsertionPoint{
			ColumnIdx: 0,
			ItemIdx:   0,
		})

		if !errors.Is(err, wantErr) {
			t.Fatalf("expected err %v, got %v", wantErr, err)
		}

		want := workspaceView(0, columnView("Column", viewA, viewB))
		got := wm.WorkspaceView()

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("workspace view mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestSequentialMutationsUseLatestRevision(t *testing.T) {
	wm := model.NewWorkspaceModel(model.NewWorkspace(model.NewColumn("Column", itemA, itemB)))

	if err := wm.WorkItemMoveDirection(0, itemA.ID(), model.DirectionDown); err != nil {
		t.Fatal(err)
	}

	want := workspaceView(1, columnView("Column", viewB, viewA))
	got := wm.WorkspaceView()

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("workspace view mismatch (-want +got):\n%s", diff)
	}

	if err := wm.WorkItemDelete(1, itemB.ID()); err != nil {
		t.Fatal(err)
	}

	want = workspaceView(2, columnView("Column", viewA))
	got = wm.WorkspaceView()

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("workspace view mismatch (-want +got):\n%s", diff)
	}
}
