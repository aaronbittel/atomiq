package model_test

import (
	"errors"
	"testing"

	"github.com/aaronbittel/atomiq/internal/model"
	"github.com/google/go-cmp/cmp"
)

var (
	A = item("1", "A")
	B = item("2", "B")
)

func TestWorkspaceView(t *testing.T) {
	t.Run("view returns snapshot", func(t *testing.T) {
		wm := model.NewWorkspaceModel(workspace(column("Column", A)))

		view := wm.WorkspaceView()
		view.Columns[0].Name = "Mutated"
		view.Columns[0].WorkItems[0].Name = "Mutated"

		got := wm.WorkspaceView()
		want := workspaceView(columnView("Column", itemView(A)))

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("workspace view mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestWorkItemAdd(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		wm := model.NewWorkspaceModel(workspace(column("Column")))

		if err := wm.WorkItemAdd(0, "New item"); err != nil {
			t.Fatal(err)
		}

		got := wm.WorkspaceView()
		if len(got.Columns[0].WorkItems) != 1 {
			t.Fatalf("expected one item, got %d", len(got.Columns[0].WorkItems))
		}
		if got.Columns[0].WorkItems[0].Name != "New item" {
			t.Fatalf("expected trimmed item name")
		}
	})

	t.Run("invalid column index", func(t *testing.T) {
		wm := model.NewWorkspaceModel(workspace(column("Column")))

		wantErr := model.ErrInvalidPosition

		if err := wm.WorkItemAdd(1, "invalid column"); !errors.Is(err, wantErr) {
			t.Fatalf("expected %v, got %v", wantErr, err)
		}
	})
}

func TestWorkItemDelete(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		wm := model.NewWorkspaceModel(workspace(column("Column", A)))

		if err := wm.WorkItemDelete(A.ID, model.WorkItemPosition{}); err != nil {
			t.Fatal(err)
		}

		want := workspaceView(columnView("Column"))
		got := wm.WorkspaceView()

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("workspace view mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("invalid item id", func(t *testing.T) {
		wm := model.NewWorkspaceModel(workspace(column("Column", A)))

		err := wm.WorkItemDelete(B.ID, model.WorkItemPosition{})
		if !errors.Is(err, model.ErrItemIDMismatch) {
			t.Fatalf("expected %v, got %v", model.ErrItemIDMismatch, err)
		}
	})
}

func TestWorkItemMoveDirection(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		wm := model.NewWorkspaceModel(workspace(column("Column", A, B)))

		if err := wm.WorkItemMoveDirection(B.ID, model.WorkItemPosition{ColumnIdx: 0, ItemIdx: 1}, model.DirectionUp); err != nil {
			t.Fatal(err)
		}

		want := workspaceView(columnView("Column", itemView(B), itemView(A)))
		got := wm.WorkspaceView()

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("workspace view mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("invalid from position", func(t *testing.T) {
		wm := model.NewWorkspaceModel(workspace(column("Column", A, B)))

		err := wm.WorkItemMoveDirection(B.ID, model.WorkItemPosition{ColumnIdx: 1, ItemIdx: 1}, model.DirectionUp)
		if !errors.Is(err, model.ErrInvalidPosition) {
			t.Fatal(err)
		}

		want := workspaceView(columnView("Column", itemView(A), itemView(B)))
		got := wm.WorkspaceView()

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("workspace view mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestNewWorkspaceModel(t *testing.T) {
	t.Run("workspace model is cloned", func(t *testing.T) {
		initial := workspace(column("Column", A))
		wm := model.NewWorkspaceModel(initial)

		initial.Columns[0].Name = "Mutated"
		initial.Columns[0].WorkItems[0].Name = "Mutated"

		want := workspaceView(columnView("Column", itemView(A)))
		got := wm.WorkspaceView()

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("workspace view mismatch (-want +got):\n%s", diff)
		}
	})
}
