package model

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var (
	A = item("1", "A")
	B = item("2", "B")
	C = item("3", "C")
)

func TestAdd(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		ws := workspace(column("Column", A))

		if err := ws.add(0, "  New item  "); err != nil {
			t.Fatal(err)
		}

		items := ws.Columns[0].WorkItems

		if len(items) != 2 {
			t.Fatalf("expected 2 work items, got %d", len(items))
		}

		if items[1].Name != "New item" {
			t.Fatalf("expected trimmed name %q, got %q", "New item", items[1].Name)
		}

		if len(items[1].ID) != WorkItemIDLength {
			t.Fatal("invalid work item ID")
		}
	})

	t.Run("blank name", func(t *testing.T) {
		ws := workspace(column("Column", A))

		if err := ws.add(0, "   "); !errors.Is(err, ErrInvalidWorkItemName) {
			t.Fatalf("expected error %v, got %v", ErrInvalidWorkItemName, err)
		}

		want := workspace(column("Column", A))

		if diff := cmp.Diff(want, ws); diff != "" {
			t.Errorf("workspace mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("invalid column", func(t *testing.T) {
		ws := workspace(column("Column", A))

		wantErr := ErrInvalidPosition

		if err := ws.add(1, "New item"); !errors.Is(err, wantErr) {
			t.Fatalf("expected error %v, got %v", wantErr, err)
		}
	})
}

func TestDelete(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		tests := []struct {
			name string
			ws   Workspace
			id   string
			want Workspace
		}{
			{
				name: "only item",
				ws: workspace(
					column("Column", A),
				),
				id:   A.ID,
				want: workspace(column("Column")),
			},
			{
				name: "first item",
				ws: workspace(
					column("Column", A, B),
				),
				id:   A.ID,
				want: workspace(column("Column", B)),
			},
			{
				name: "middle item",
				ws: workspace(
					column("Column", A, B, C),
				),
				id:   B.ID,
				want: workspace(column("Column", A, C)),
			},
			{
				name: "last item",
				ws: workspace(
					column("Column", A, B, C),
				),
				id:   C.ID,
				want: workspace(column("Column", A, B)),
			},
			{
				name: "multiple columns",
				ws: workspace(
					column("Column 1", A, B),
					column("Column 2", C),
				),
				id: C.ID,
				want: workspace(
					column("Column 1", A, B),
					column("Column 2"),
				),
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if err := tt.ws.delete(tt.id); err != nil {
					t.Fatal(err)
				}

				if diff := cmp.Diff(tt.want, tt.ws); diff != "" {
					t.Errorf("workspace mismatch (-want +got):\n%s", diff)
				}
			})
		}
	})

	t.Run("invalid", func(t *testing.T) {
		tests := []struct {
			name    string
			id      string
			pos     WorkItemPosition
			wantErr error
		}{
			{
				name:    "ID mismatch",
				id:      "wrong ID",
				wantErr: ErrWorkItemNotFound,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				ws := workspace(column("Column", A))

				err := ws.delete(tt.id)
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected err %v, got %v", tt.wantErr, err)
				}

				want := workspace(column("Column", A))

				if diff := cmp.Diff(want, ws); diff != "" {
					t.Errorf("workspace mismatch (-want +got):\n%s", diff)
				}
			})
		}
	})
}

func TestMoveInDirection(t *testing.T) {
	t.Run("no ops", func(t *testing.T) {
		for _, dir := range []MoveDirection{DirectionUp, DirectionDown, DirectionRight, DirectionLeft} {
			t.Run(string(dir), func(t *testing.T) {
				ws := workspace(column("Column", A))

				if err := ws.moveInDirection("1", WorkItemPosition{}, dir); err != nil {
					t.Fatal(err)
				}
				want := workspace(column("Column", A))

				if diff := cmp.Diff(want, ws); diff != "" {
					t.Errorf("workspace mismatch (-want +got):\n%s", diff)
				}
			})
		}
	})

	t.Run("valid", func(t *testing.T) {
		tests := []struct {
			name      string
			ws        Workspace
			id        string
			from      WorkItemPosition
			direction MoveDirection
			want      Workspace
		}{
			{
				name: "up",
				ws: workspace(
					column("Column", A, B),
				),
				id:        B.ID,
				from:      WorkItemPosition{ColumnIdx: 0, ItemIdx: 1},
				direction: DirectionUp,
				want: workspace(
					column("Column", B, A),
				),
			},
			{
				name: "down",
				ws: workspace(
					column("Column", A, B),
				),
				id:        A.ID,
				from:      WorkItemPosition{ColumnIdx: 0, ItemIdx: 0},
				direction: DirectionDown,
				want: workspace(
					column("Column", B, A),
				),
			},
			{
				name: "right",
				ws: workspace(
					column("Column 1", A, B),
					column("Column 2", C),
				),
				id:        A.ID,
				from:      WorkItemPosition{ColumnIdx: 0, ItemIdx: 0},
				direction: DirectionRight,
				want: workspace(
					column("Column 1", B),
					column("Column 2", C, A),
				),
			},
			{
				name: "left",
				ws: workspace(
					column("Column 1", C),
					column("Column 2", A, B),
				),
				id:        A.ID,
				from:      WorkItemPosition{ColumnIdx: 1, ItemIdx: 0},
				direction: DirectionLeft,
				want: workspace(
					column("Column 1", C, A),
					column("Column 2", B),
				),
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if err := tt.ws.moveInDirection(tt.id, tt.from, tt.direction); err != nil {
					t.Fatal(err)
				}

				if diff := cmp.Diff(tt.want, tt.ws); diff != "" {
					t.Errorf("workspace mismatch (-want +got):\n%s", diff)
				}
			})
		}
	})

	t.Run("invalid", func(t *testing.T) {
		tests := []struct {
			name      string
			ws        Workspace
			id        string
			from      WorkItemPosition
			direction MoveDirection
			wantErr   error
		}{
			{
				name: "column index",
				ws: workspace(
					column("Column", A),
				),
				id:        A.ID,
				from:      WorkItemPosition{ColumnIdx: 1},
				direction: DirectionUp,
				wantErr:   ErrInvalidPosition,
			},
			{
				name: "item index",
				ws: workspace(
					column("Column", A),
				),
				id:        A.ID,
				from:      WorkItemPosition{ItemIdx: 1},
				direction: DirectionUp,
				wantErr:   ErrInvalidPosition,
			},
			{
				name: "item ID",
				ws: workspace(
					column("Column", A),
				),
				id:        B.ID,
				from:      WorkItemPosition{},
				direction: DirectionUp,
				wantErr:   ErrItemIDMismatch,
			},
			{
				name: "move direction",
				ws: workspace(
					column("Column", A),
				),
				id:        A.ID,
				from:      WorkItemPosition{},
				direction: MoveDirection("invalid"),
				wantErr:   ErrInvalidMoveDirection,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := tt.ws.moveInDirection(tt.id, tt.from, tt.direction)
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

func TestMoveToPosition(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		tests := []struct {
			name string
			ws   Workspace
			id   string
			from WorkItemPosition
			to   WorkItemPosition
			want Workspace
		}{
			{
				name: "move between columns",
				ws: workspace(
					column("Column 1", A, B),
					column("Column 2", C),
				),
				id:   B.ID,
				from: position(0, 1),
				to:   position(1, 1),
				want: workspace(
					column("Column 1", A),
					column("Column 2", C, B),
				),
			},
			{
				name: "move earlier in same column",
				ws: workspace(
					column("Column", A, B, C),
				),
				id:   C.ID,
				from: position(0, 2),
				to:   position(0, 0),
				want: workspace(
					column("Column", C, A, B),
				),
			},
			{
				name: "move later in same column",
				ws: workspace(
					column("Column", A, B, C),
				),
				id:   A.ID,
				from: position(0, 0),
				to:   position(0, 2),
				want: workspace(
					column("Column", B, A, C),
				),
			},
			{
				name: "stay in same position",
				ws: workspace(
					column("Column", A, B, C),
				),
				id:   A.ID,
				from: position(0, 0),
				to:   position(0, 0),
				want: workspace(
					column("Column", A, B, C),
				),
			},
			{
				name: "same column move to end",
				ws: workspace(
					column("Column", A, B, C),
				),
				id:   A.ID,
				from: position(0, 0),
				to:   position(0, 3),
				want: workspace(
					column("Column", B, C, A),
				),
			},
			{
				name: "same column move to beginning",
				ws: workspace(
					column("Column", A, B, C),
				),
				id:   C.ID,
				from: position(0, 2),
				to:   position(0, 0),
				want: workspace(
					column("Column", C, A, B),
				),
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := tt.ws.moveToPosition(tt.id, tt.from, tt.to)
				if err != nil {
					t.Fatalf("WorkItemMove() unexpected error: %v", err)
				}

				if diff := cmp.Diff(tt.want, tt.ws); diff != "" {
					t.Errorf("workspace mismatch (-want +got):\n%s", diff)
				}
			})
		}
	})

	t.Run("invalid", func(t *testing.T) {
		tests := []struct {
			name    string
			ws      Workspace
			id      string
			from    WorkItemPosition
			to      WorkItemPosition
			wantErr error
		}{
			{
				name: "from column",
				ws: workspace(
					column("Column", A),
				),
				id:      A.ID,
				from:    position(9, 0),
				to:      position(0, 0),
				wantErr: ErrInvalidPosition,
			},
			{
				name: "to column",
				ws: workspace(
					column("Column", A),
				),
				id:      A.ID,
				from:    position(0, 0),
				to:      position(9, 0),
				wantErr: ErrInvalidPosition,
			},
			{
				name: "from index",
				ws: workspace(
					column("Column", A),
				),
				id:      A.ID,
				from:    position(0, 2),
				to:      position(0, 0),
				wantErr: ErrInvalidPosition,
			},
			{
				name: "to index",
				ws: workspace(
					column("Column 1", A),
					column("Column 2"),
				),
				id:      A.ID,
				from:    position(0, 0),
				to:      position(1, 2),
				wantErr: ErrInvalidPosition,
			},
			{
				name: "equal positions are still validated",
				ws: workspace(
					column("Column", A),
				),
				id:      A.ID,
				from:    position(9, 0),
				to:      position(9, 0),
				wantErr: ErrInvalidPosition,
			},
			{
				name: "ID mismatch",
				ws: workspace(
					column("Column", A),
				),
				id:      B.ID,
				from:    position(0, 0),
				to:      position(0, 0),
				wantErr: ErrItemIDMismatch,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := tt.ws.moveToPosition(tt.id, tt.from, tt.to)

				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("WorkItemMove() expected error: %v, got nil", tt.wantErr)
				}
			})
		}
	})
}
