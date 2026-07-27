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
		want := workspaceView(0, columnView("Column", itemView(A)))

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("workspace view mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestWorkItemAdd(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		wm := model.NewWorkspaceModel(workspace(column("Column")))

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
		wm := model.NewWorkspaceModel(workspace(column("Column")))

		wantErr := model.ErrInvalidPosition

		if err := wm.WorkItemAdd(0, 1, "invalid column"); !errors.Is(err, wantErr) {
			t.Fatalf("expected %v, got %v", wantErr, err)
		}
	})
}

func TestWorkItemDelete(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		wm := model.NewWorkspaceModel(workspace(column("Column", A)))

		if err := wm.WorkItemDelete(0, A.ID); err != nil {
			t.Fatal(err)
		}

		want := workspaceView(1, columnView("Column"))
		got := wm.WorkspaceView()

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("workspace view mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("invalid item id", func(t *testing.T) {
		wm := model.NewWorkspaceModel(workspace(column("Column", A)))

		wantErr := model.ErrWorkItemNotFound
		err := wm.WorkItemDelete(0, B.ID)

		if !errors.Is(err, wantErr) {
			t.Fatalf("expected %v, got %v", wantErr, err)
		}
	})
}

func TestWorkItemMoveDirection(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		wm := model.NewWorkspaceModel(workspace(column("Column", A, B)))

		if err := wm.WorkItemMoveDirection(0, B.ID, model.DirectionUp); err != nil {
			t.Fatal(err)
		}

		want := workspaceView(1, columnView("Column", itemView(B), itemView(A)))
		got := wm.WorkspaceView()

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("workspace view mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("no-op move does not update revision", func(t *testing.T) {
		wm := model.NewWorkspaceModel(workspace(column("Column", A)))

		if err := wm.WorkItemMoveDirection(0, A.ID, model.DirectionUp); err != nil {
			t.Fatal(err)
		}

		want := workspaceView(0, columnView("Column", itemView(A)))
		got := wm.WorkspaceView()

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("workspace view mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("revision conflict", func(t *testing.T) {
		wm := model.NewWorkspaceModel(workspace(column("Column", A)))

		wantErr := model.ErrRevisionConflict
		if err := wm.WorkItemMoveDirection(1, A.ID, model.DirectionUp); !errors.Is(err, wantErr) {
			t.Fatalf("expected err %v, got %v", wantErr, err)
		}

		want := workspaceView(0, columnView("Column", itemView(A)))
		got := wm.WorkspaceView()

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("workspace view mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestWorkItemMovePosition(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		wm := model.NewWorkspaceModel(workspace(column("Column", A, B)))

		err := wm.WorkItemMovePosition(0, B.ID, model.WorkItemInsertionPoint{
			ColumnIdx: 0,
			ItemIdx:   0,
		})
		if err != nil {
			t.Fatal(err)
		}

		want := workspaceView(1, columnView("Column", itemView(B), itemView(A)))
		got := wm.WorkspaceView()

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("workspace view mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("no-op move does not update revision", func(t *testing.T) {
		wm := model.NewWorkspaceModel(workspace(column("Column", A, B)))

		err := wm.WorkItemMovePosition(0, B.ID, model.WorkItemInsertionPoint{
			ColumnIdx: 0,
			ItemIdx:   1,
		})
		if err != nil {
			t.Fatal(err)
		}

		want := workspaceView(0, columnView("Column", itemView(A), itemView(B)))
		got := wm.WorkspaceView()

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("workspace view mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("no-op move does not update revision 2", func(t *testing.T) {
		wm := model.NewWorkspaceModel(workspace(column("Column", A, B)))

		err := wm.WorkItemMovePosition(0, B.ID, model.WorkItemInsertionPoint{
			ColumnIdx: 0,
			ItemIdx:   2,
		})
		if err != nil {
			t.Fatal(err)
		}

		want := workspaceView(0, columnView("Column", itemView(A), itemView(B)))
		got := wm.WorkspaceView()

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("workspace view mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("revision conflict", func(t *testing.T) {
		wm := model.NewWorkspaceModel(workspace(column("Column", A, B)))

		wantErr := model.ErrRevisionConflict
		err := wm.WorkItemMovePosition(1, B.ID, model.WorkItemInsertionPoint{
			ColumnIdx: 0,
			ItemIdx:   0,
		})

		if !errors.Is(err, wantErr) {
			t.Fatalf("expected err %v, got %v", wantErr, err)
		}

		want := workspaceView(0, columnView("Column", itemView(A), itemView(B)))
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

		want := workspaceView(0, columnView("Column", itemView(A)))
		got := wm.WorkspaceView()

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("workspace view mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestSequentialMutationsUseLatestRevision(t *testing.T) {
	wm := model.NewWorkspaceModel(workspace(column("Column", A, B)))

	if err := wm.WorkItemMoveDirection(0, A.ID, model.DirectionDown); err != nil {
		t.Fatal(err)
	}

	want := workspaceView(1, columnView("Column", itemView(B), itemView(A)))
	got := wm.WorkspaceView()

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("workspace view mismatch (-want +got):\n%s", diff)
	}

	if err := wm.WorkItemDelete(1, B.ID); err != nil {
		t.Fatal(err)
	}

	want = workspaceView(2, columnView("Column", itemView(A)))
	got = wm.WorkspaceView()

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("workspace view mismatch (-want +got):\n%s", diff)
	}
}
