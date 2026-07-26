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

				want := cloneWorkspace(workspace)

				if err := wm.WorkItemMoveDirection("1", model.WorkItemPosition{}, dir); err != nil {
					t.Fatal(err)
				}

				got := wm.WorkspaceView()

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
			want      model.Workspace
		}{
			{
				name: "up",
				initial: workspace(
					column("Column", A, B),
				),
				itemID:    B.ID,
				from:      model.WorkItemPosition{ColumnIdx: 0, ItemIdx: 1},
				direction: model.DirectionUp,
				want: workspace(
					column("Column", B, A),
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
				want: workspace(
					column("Column", B, A),
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
				want: workspace(
					column("Column 1", B),
					column("Column 2", C, A),
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
				want: workspace(
					column("Column 1", C, A),
					column("Column 2", B),
				),
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				wm := model.NewWorkspaceModel(tt.initial)

				if err := wm.WorkItemMoveDirection(tt.itemID, tt.from, tt.direction); err != nil {
					t.Fatal(err)
				}

				got := wm.WorkspaceView()

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
			want    model.Workspace
		}{
			{
				name: "move between columns",
				initial: workspace(
					column("Column 1", A, B),
					column("Column 2", C),
				),
				from: position(0, 1),
				to:   position(1, 1),
				want: workspace(
					column("Column 1", A),
					column("Column 2", C, B),
				),
			},
			{
				name: "move earlier in same column",
				initial: workspace(
					column("Column", A, B, C),
				),
				from: position(0, 2),
				to:   position(0, 0),
				want: workspace(
					column("Column", C, A, B),
				),
			},
			{
				name: "move later in same column",
				initial: workspace(
					column("Column", A, B, C),
				),
				from: position(0, 0),
				to:   position(0, 2),
				want: workspace(
					column("Column", B, A, C),
				),
			},
			{
				name: "stay in same position",
				initial: workspace(
					column("Column", A, B, C),
				),
				from: position(0, 0),
				to:   position(0, 0),
				want: workspace(
					column("Column", A, B, C),
				),
			},
			{
				name: "same column move to end",
				initial: workspace(
					column("Column", A, B, C),
				),
				from: position(0, 0),
				to:   position(0, 3),
				want: workspace(
					column("Column", B, C, A),
				),
			},
			{
				name: "same column move to beginning",
				initial: workspace(
					column("Column", A, B, C),
				),
				from: position(0, 2),
				to:   position(0, 0),
				want: workspace(
					column("Column", C, A, B),
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

				got := wm.WorkspaceView()

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

func workspace(columns ...model.Column) model.Workspace {
	return model.Workspace{
		Columns: columns,
	}
}

func column(name string, items ...model.WorkItem) model.Column {
	var result []model.WorkItem

	for _, item := range items {
		result = append(result, item)
	}

	return model.Column{
		Name:      name,
		WorkItems: result,
	}
}

func item(id, name string) model.WorkItem {
	return model.WorkItem{
		ID:   id,
		Name: name,
	}
}

func position(columnIdx, itemIdx int) model.WorkItemPosition {
	return model.WorkItemPosition{
		ColumnIdx: columnIdx,
		ItemIdx:   itemIdx,
	}
}

func cloneWorkspace(src model.Workspace) model.Workspace {
	dst := src
	dst.Columns = make([]model.Column, len(src.Columns))

	for i, column := range src.Columns {
		dst.Columns[i] = column
		dst.Columns[i].WorkItems = append(
			[]model.WorkItem(nil),
			column.WorkItems...,
		)
	}

	return dst
}

func TestWorkItemDelete(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		tests := []struct {
			name    string
			initial model.Workspace
			itemID  string
			pos     model.WorkItemPosition
			want    model.Workspace
		}{
			{
				name: "only item",
				initial: workspace(
					column("Column", A),
				),
				itemID: A.ID,
				pos:    model.WorkItemPosition{},
				want:   workspace(column("Column")),
			},
			{
				name: "first item",
				initial: workspace(
					column("Column", A, B),
				),
				itemID: A.ID,
				pos:    model.WorkItemPosition{},
				want:   workspace(column("Column", B)),
			},
			{
				name: "middle item",
				initial: workspace(
					column("Column", A, B, C),
				),
				itemID: B.ID,
				pos:    model.WorkItemPosition{ColumnIdx: 0, ItemIdx: 1},
				want:   workspace(column("Column", A, C)),
			},
			{
				name: "last item",
				initial: workspace(
					column("Column", A, B, C),
				),
				itemID: C.ID,
				pos:    model.WorkItemPosition{ColumnIdx: 0, ItemIdx: 2},
				want:   workspace(column("Column", A, B)),
			},
			{
				name: "multiple columns",
				initial: workspace(
					column("Column 1", A, B),
					column("Column 2", C),
				),
				itemID: C.ID,
				pos:    model.WorkItemPosition{ColumnIdx: 1, ItemIdx: 0},
				want: workspace(
					column("Column 1", A, B),
					column("Column 2"),
				),
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				wm := model.NewWorkspaceModel(tt.initial)

				if err := wm.WorkItemDelete(tt.itemID, tt.pos); err != nil {
					t.Fatal(err)
				}

				got := wm.WorkspaceView()

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
			itemID  string
			pos     model.WorkItemPosition
			wantErr error
		}{
			{
				name:    "column index",
				initial: workspace(column("Column", A)),
				itemID:  A.ID,
				pos:     model.WorkItemPosition{ColumnIdx: 1},
				wantErr: model.ErrInvalidPosition,
			},
			{
				name:    "item index",
				initial: workspace(column("Column", A)),
				itemID:  A.ID,
				pos:     model.WorkItemPosition{ColumnIdx: 0, ItemIdx: 1},
				wantErr: model.ErrInvalidPosition,
			},
			{
				name:    "id mismatach",
				initial: workspace(column("Column", A)),
				itemID:  "wrong ID",
				pos:     model.WorkItemPosition{},
				wantErr: model.ErrItemIDMismatch,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				wm := model.NewWorkspaceModel(tt.initial)
				want := cloneWorkspace(tt.initial)

				err := wm.WorkItemDelete(tt.itemID, tt.pos)
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected err %v, got %v", tt.wantErr, err)
				}

				got := wm.WorkspaceView()

				if diff := cmp.Diff(want, got); diff != "" {
					t.Errorf("workspace mismatch (-want +got):\n%s", diff)
				}
			})
		}
	})
}
