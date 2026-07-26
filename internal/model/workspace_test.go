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
	C = item("3", "C")
)

func TestWorkItemMoveDirection(t *testing.T) {
	t.Run("no ops", func(t *testing.T) {
		for _, dir := range []model.MoveDirection{model.DirectionUp, model.DirectionDown, model.DirectionRight, model.DirectionLeft} {
			t.Run(string(dir), func(t *testing.T) {
				workspace := workspace(column("Column", A))
				wm := model.NewWorkspaceModel(workspace)

				want := workspaceView(columnView("Column", itemView(A)))

				if err := wm.WorkItemMoveDirection("1", model.WorkItemPosition{}, dir); err != nil {
					t.Fatal(err)
				}

				got := wm.CurrentWorkspaceView()

				if diff := cmp.Diff(want, got); diff != "" {
					t.Errorf("workspace mismatch (-want +got):\n%s", diff)
				}
			})
		}
	})

	t.Run("valid", func(t *testing.T) {
		tests := []struct {
			name      string
			initial   model.Workspace
			itemID    string
			from      model.WorkItemPosition
			direction model.MoveDirection
			want      model.WorkspaceView
		}{
			{
				name: "up",
				initial: workspace(
					column("Column", A, B),
				),
				itemID:    B.ID,
				from:      model.WorkItemPosition{ColumnIdx: 0, ItemIdx: 1},
				direction: model.DirectionUp,
				want: workspaceView(
					columnView("Column", itemView(B), itemView(A)),
				),
			},
			{
				name: "down",
				initial: workspace(
					column("Column", A, B),
				),
				itemID:    A.ID,
				from:      model.WorkItemPosition{ColumnIdx: 0, ItemIdx: 0},
				direction: model.DirectionDown,
				want: workspaceView(
					columnView("Column", itemView(B), itemView(A)),
				),
			},
			{
				name: "right",
				initial: workspace(
					column("Column 1", A, B),
					column("Column 2", C),
				),
				itemID:    A.ID,
				from:      model.WorkItemPosition{ColumnIdx: 0, ItemIdx: 0},
				direction: model.DirectionRight,
				want: workspaceView(
					columnView("Column 1", itemView(B)),
					columnView("Column 2", itemView(C), itemView(A)),
				),
			},
			{
				name: "left",
				initial: workspace(
					column("Column 1", C),
					column("Column 2", A, B),
				),
				itemID:    A.ID,
				from:      model.WorkItemPosition{ColumnIdx: 1, ItemIdx: 0},
				direction: model.DirectionLeft,
				want: workspaceView(
					columnView("Column 1", itemView(C), itemView(A)),
					columnView("Column 2", itemView(B)),
				),
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				wm := model.NewWorkspaceModel(tt.initial)

				if err := wm.WorkItemMoveDirection(tt.itemID, tt.from, tt.direction); err != nil {
					t.Fatal(err)
				}

				got := wm.CurrentWorkspaceView()

				if diff := cmp.Diff(tt.want, got); diff != "" {
					t.Errorf("workspace mismatch (-want +got):\n%s", diff)
				}
			})
		}
	})

	t.Run("invalid", func(t *testing.T) {
		tests := []struct {
			name      string
			initial   model.Workspace
			itemID    string
			from      model.WorkItemPosition
			direction model.MoveDirection
			wantErr   error
		}{
			{
				name: "column index",
				initial: workspace(
					column("Column", A),
				),
				itemID:    A.ID,
				from:      model.WorkItemPosition{ColumnIdx: 1},
				direction: model.DirectionUp,
				wantErr:   model.ErrInvalidPosition,
			},
			{
				name: "item index",
				initial: workspace(
					column("Column", A),
				),
				itemID:    A.ID,
				from:      model.WorkItemPosition{ItemIdx: 1},
				direction: model.DirectionUp,
				wantErr:   model.ErrInvalidPosition,
			},
			{
				name: "item ID",
				initial: workspace(
					column("Column", A),
				),
				itemID:    B.ID,
				from:      model.WorkItemPosition{},
				direction: model.DirectionUp,
				wantErr:   model.ErrItemIDMismatch,
			},
			{
				name: "move direction",
				initial: workspace(
					column("Column", A),
				),
				itemID:    A.ID,
				from:      model.WorkItemPosition{},
				direction: model.MoveDirection("invalid"),
				wantErr:   model.ErrInvalidMoveDirection,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				wm := model.NewWorkspaceModel(tt.initial)

				err := wm.WorkItemMoveDirection(tt.itemID, tt.from, tt.direction)
				if err == nil {
					t.Fatalf("expected error %v, got nil", tt.wantErr)
				}

				if !errors.Is(err, tt.wantErr) {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
			})
		}
	})
}

func TestWorkItemMovePosition(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		tests := []struct {
			name    string
			initial model.Workspace
			from    model.WorkItemPosition
			to      model.WorkItemPosition
			want    model.WorkspaceView
		}{
			{
				name: "move between columns",
				initial: workspace(
					column("Column 1", A, B),
					column("Column 2", C),
				),
				from: position(0, 1),
				to:   position(1, 1),
				want: workspaceView(
					columnView("Column 1", itemView(A)),
					columnView("Column 2", itemView(C), itemView(B)),
				),
			},
			{
				name: "move earlier in same column",
				initial: workspace(
					column("Column", A, B, C),
				),
				from: position(0, 2),
				to:   position(0, 0),
				want: workspaceView(
					columnView("Column", itemView(C), itemView(A), itemView(B)),
				),
			},
			{
				name: "move later in same column",
				initial: workspace(
					column("Column", A, B, C),
				),
				from: position(0, 0),
				to:   position(0, 2),
				want: workspaceView(
					columnView("Column", itemView(B), itemView(A), itemView(C)),
				),
			},
			{
				name: "stay in same position",
				initial: workspace(
					column("Column", A, B, C),
				),
				from: position(0, 0),
				to:   position(0, 0),
				want: workspaceView(
					columnView("Column", itemView(A), itemView(B), itemView(C)),
				),
			},
			{
				name: "same column move to end",
				initial: workspace(
					column("Column", A, B, C),
				),
				from: position(0, 0),
				to:   position(0, 3),
				want: workspaceView(
					columnView("Column", itemView(B), itemView(C), itemView(A)),
				),
			},
			{
				name: "same column move to beginning",
				initial: workspace(
					column("Column", A, B, C),
				),
				from: position(0, 2),
				to:   position(0, 0),
				want: workspaceView(
					columnView("Column", itemView(C), itemView(A), itemView(B)),
				),
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				wm := model.NewWorkspaceModel(tt.initial)

				err := wm.WorkItemMovePosition(tt.from, tt.to)
				if err != nil {
					t.Fatalf("WorkItemMove() unexpected error: %v", err)
				}

				got := wm.CurrentWorkspaceView()

				if diff := cmp.Diff(tt.want, got); diff != "" {
					t.Errorf("workspace mismatch (-want +got):\n%s", diff)
				}
			})
		}
	})

	t.Run("invalid", func(t *testing.T) {
		tests := []struct {
			name    string
			initial model.Workspace
			from    model.WorkItemPosition
			to      model.WorkItemPosition
			wantErr error
		}{
			{
				name: "from column",
				initial: workspace(
					column("Column", A),
				),
				from:    position(9, 0),
				to:      position(0, 0),
				wantErr: model.ErrInvalidPosition,
			},
			{
				name: "to column",
				initial: workspace(
					column("Column", A),
				),
				from:    position(0, 0),
				to:      position(9, 0),
				wantErr: model.ErrInvalidPosition,
			},
			{
				name: "from index",
				initial: workspace(
					column("Column", A),
				),
				from:    position(0, 2),
				to:      position(0, 0),
				wantErr: model.ErrInvalidPosition,
			},
			{
				name: "to index",
				initial: workspace(
					column("Column 1", A),
					column("Column 2"),
				),
				from:    position(0, 0),
				to:      position(1, 2),
				wantErr: model.ErrInvalidPosition,
			},
			// TODO: find better name, is this tests useful now after refactoring?
			{
				name: "from==to, but illegal access",
				initial: workspace(
					column("Column", A),
				),
				from:    position(9, 0),
				to:      position(9, 0),
				wantErr: model.ErrInvalidPosition,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				wm := model.NewWorkspaceModel(tt.initial)

				err := wm.WorkItemMovePosition(tt.from, tt.to)

				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("WorkItemMove() expected error: %v, got nil", tt.wantErr)
				}
			})
		}
	})
}

func TestWorkItemDelete(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		tests := []struct {
			name    string
			initial model.Workspace
			itemID  string
			pos     model.WorkItemPosition
			want    model.WorkspaceView
		}{
			{
				name: "only item",
				initial: workspace(
					column("Column", A),
				),
				itemID: A.ID,
				pos:    model.WorkItemPosition{},
				want:   workspaceView(columnView("Column")),
			},
			{
				name: "first item",
				initial: workspace(
					column("Column", A, B),
				),
				itemID: A.ID,
				pos:    model.WorkItemPosition{},
				want:   workspaceView(columnView("Column", itemView(B))),
			},
			{
				name: "middle item",
				initial: workspace(
					column("Column", A, B, C),
				),
				itemID: B.ID,
				pos:    model.WorkItemPosition{ColumnIdx: 0, ItemIdx: 1},
				want:   workspaceView(columnView("Column", itemView(A), itemView(C))),
			},
			{
				name: "last item",
				initial: workspace(
					column("Column", A, B, C),
				),
				itemID: C.ID,
				pos:    model.WorkItemPosition{ColumnIdx: 0, ItemIdx: 2},
				want:   workspaceView(columnView("Column", itemView(A), itemView(B))),
			},
			{
				name: "multiple columns",
				initial: workspace(
					column("Column 1", A, B),
					column("Column 2", C),
				),
				itemID: C.ID,
				pos:    model.WorkItemPosition{ColumnIdx: 1, ItemIdx: 0},
				want: workspaceView(
					columnView("Column 1", itemView(A), itemView(B)),
					columnView("Column 2"),
				),
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				wm := model.NewWorkspaceModel(tt.initial)

				if err := wm.WorkItemDelete(tt.itemID, tt.pos); err != nil {
					t.Fatal(err)
				}

				got := wm.CurrentWorkspaceView()

				if diff := cmp.Diff(tt.want, got); diff != "" {
					t.Errorf("workspace mismatch (-want +got):\n%s", diff)
				}
			})
		}
	})

	t.Run("invalid", func(t *testing.T) {
		tests := []struct {
			name    string
			itemID  string
			pos     model.WorkItemPosition
			wantErr error
		}{
			{
				name:    "column index",
				itemID:  A.ID,
				pos:     model.WorkItemPosition{ColumnIdx: 1},
				wantErr: model.ErrInvalidPosition,
			},
			{
				name:    "item index",
				itemID:  A.ID,
				pos:     model.WorkItemPosition{ColumnIdx: 0, ItemIdx: 1},
				wantErr: model.ErrInvalidPosition,
			},
			{
				name:    "id mismatach",
				itemID:  "wrong ID",
				pos:     model.WorkItemPosition{},
				wantErr: model.ErrItemIDMismatch,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				initial := workspace(column("Column", A))
				wm := model.NewWorkspaceModel(initial)

				err := wm.WorkItemDelete(tt.itemID, tt.pos)
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected err %v, got %v", tt.wantErr, err)
				}

				want := workspaceView(columnView("Column", itemView(A)))
				got := wm.CurrentWorkspaceView()

				if diff := cmp.Diff(want, got); diff != "" {
					t.Errorf("workspace mismatch (-want +got):\n%s", diff)
				}
			})
		}
	})
}

func TestWorkItemAdd(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		workspace := workspace(column("Column", A))
		wm := model.NewWorkspaceModel(workspace)

		if err := wm.WorkItemAdd(0, "  New item  "); err != nil {
			t.Fatal(err)
		}

		got := wm.CurrentWorkspaceView()
		items := got.Columns[0].WorkItems

		if len(items) != 2 {
			t.Fatalf("expected 2 work items, got %d", len(items))
		}

		if items[1].Name != "New item" {
			t.Fatalf("expected trimmed name %q, got %q", "New item", items[1].Name)
		}

		if len(items[1].ID) != model.WorkItemIDLength {
			t.Fatal("invalid work item ID")
		}
	})

	t.Run("blank name", func(t *testing.T) {
		initial := workspace(column("Column", A))
		wm := model.NewWorkspaceModel(initial)

		err := wm.WorkItemAdd(0, "   ")
		if !errors.Is(err, model.ErrInvalidWorkItemName) {
			t.Fatalf("expected error %v, got %v", model.ErrInvalidWorkItemName, err)
		}

		want := workspaceView(columnView("Column", itemView(A)))
		got := wm.CurrentWorkspaceView()

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("workspace mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("invalid column", func(t *testing.T) {
		wm := model.NewWorkspaceModel(workspace(column("Column", A)))

		err := wm.WorkItemAdd(1, "New item")
		if !errors.Is(err, model.ErrInvalidColumn) {
			t.Fatalf("expected error %v, got %v", model.ErrInvalidColumn, err)
		}
	})
}
