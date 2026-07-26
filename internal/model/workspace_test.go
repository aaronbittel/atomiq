package model_test

import (
	"errors"
	"testing"

	"github.com/aaronbittel/atomiq/internal/model"
	"github.com/google/go-cmp/cmp"
)

func TestWorkItemMoveDirection(t *testing.T) {
	var (
		A = item("1", "A")
		B = item("2", "B")
		C = item("3", "C")
	)

	t.Run("no ops", func(t *testing.T) {
		for _, dir := range []model.MoveDirection{model.DirectionUp, model.DirectionDown, model.DirectionRight, model.DirectionLeft} {
			t.Run(string(dir), func(t *testing.T) {
				wm := model.WorkspaceModel{Workspace: workspace(column(A))}

				want := cloneWorkspace(wm.Workspace)

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
					column(A, B),
				),
				itemID:    B.ID,
				from:      model.WorkItemPosition{ColumnIdx: 0, ItemIdx: 1},
				direction: model.DirectionUp,
				want: workspace(
					column(B, A),
				),
			},
			{
				name: "down",
				initial: workspace(
					column(A, B),
				),
				itemID:    A.ID,
				from:      model.WorkItemPosition{ColumnIdx: 0, ItemIdx: 0},
				direction: model.DirectionDown,
				want: workspace(
					column(B, A),
				),
			},
			{
				name: "right",
				initial: workspace(
					column(A, B),
					column(C),
				),
				itemID:    A.ID,
				from:      model.WorkItemPosition{ColumnIdx: 0, ItemIdx: 0},
				direction: model.DirectionRight,
				want: workspace(
					column(B),
					column(C, A),
				),
			},
			{
				name: "left",
				initial: workspace(
					column(C),
					column(A, B),
				),
				itemID:    A.ID,
				from:      model.WorkItemPosition{ColumnIdx: 1, ItemIdx: 0},
				direction: model.DirectionLeft,
				want: workspace(
					column(C, A),
					column(B),
				),
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				wm := model.WorkspaceModel{Workspace: tt.initial}

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
					column(A),
				),
				itemID:    A.ID,
				from:      model.WorkItemPosition{ColumnIdx: 1},
				direction: model.DirectionUp,
				wantErr:   model.ErrInvalidPosition,
			},
			{
				name: "item index",
				initial: workspace(
					column(A),
				),
				itemID:    A.ID,
				from:      model.WorkItemPosition{ItemIdx: 1},
				direction: model.DirectionUp,
				wantErr:   model.ErrInvalidPosition,
			},
			{
				name: "item ID",
				initial: workspace(
					column(A),
				),
				itemID:    B.ID,
				from:      model.WorkItemPosition{},
				direction: model.DirectionUp,
				wantErr:   model.ErrItemIDMismatch,
			},
			{
				name: "move direction",
				initial: workspace(
					column(A),
				),
				itemID:    A.ID,
				from:      model.WorkItemPosition{},
				direction: model.MoveDirection("invalid"),
				wantErr:   model.ErrInvalidMoveDirection,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				wm := model.WorkspaceModel{Workspace: tt.initial}

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
	var (
		A = item("1", "A")
		B = item("2", "B")
		C = item("3", "C")
	)

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
					column(A, B),
					column(C),
				),
				from: position(0, 1),
				to:   position(1, 1),
				want: workspace(
					column(A),
					column(C, B),
				),
			},
			{
				name: "move earlier in same column",
				initial: workspace(
					column(A, B, C),
				),
				from: position(0, 2),
				to:   position(0, 0),
				want: workspace(
					column(C, A, B),
				),
			},
			{
				name: "move later in same column",
				initial: workspace(
					column(A, B, C),
				),
				from: position(0, 0),
				to:   position(0, 2),
				want: workspace(
					column(B, C, A),
				),
			},
			{
				name: "stay in same position",
				initial: workspace(
					column(A, B, C),
				),
				from: position(0, 0),
				to:   position(0, 0),
				want: workspace(
					column(A, B, C),
				),
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				wm := &model.WorkspaceModel{Workspace: tt.initial}

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
					column(A),
				),
				from:    position(9, 0),
				to:      position(0, 0),
				wantErr: model.ErrInvalidPosition,
			},
			{
				name: "to column",
				initial: workspace(
					column(A),
				),
				from:    position(0, 0),
				to:      position(9, 0),
				wantErr: model.ErrInvalidPosition,
			},
			{
				name: "from index",
				initial: workspace(
					column(A),
				),
				from:    position(0, 2),
				to:      position(0, 0),
				wantErr: model.ErrInvalidPosition,
			},
			{
				name: "to index",
				initial: workspace(
					column(A),
					column(),
				),
				from:    position(0, 0),
				to:      position(1, 2),
				wantErr: model.ErrInvalidPosition,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				wm := &model.WorkspaceModel{
					Workspace: cloneWorkspace(tt.initial),
				}

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

func column(items ...model.WorkItem) model.Column {
	var result []model.WorkItem

	for _, item := range items {
		result = append(result, item)
	}

	return model.Column{
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
