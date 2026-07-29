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

		updated, err := ws.add(0, "  New item  ")
		if err != nil {
			t.Fatal(err)
		}

		if !updated {
			t.Fatal("ws should have been updated")
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

		wantErr := ErrInvalidWorkItemName

		updated, err := ws.add(0, "   ")
		if !errors.Is(err, wantErr) {
			t.Fatalf("expected error %v, got %v", wantErr, err)
		}

		if updated {
			t.Fatal("expected workspace not to be updated")
		}

		want := workspace(column("Column", A))

		if diff := cmp.Diff(want, ws); diff != "" {
			t.Errorf("workspace mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("invalid column", func(t *testing.T) {
		ws := workspace(column("Column", A))

		wantErr := ErrInvalidPosition

		updated, err := ws.add(1, "New item")
		if !errors.Is(err, wantErr) {
			t.Fatalf("expected error %v, got %v", wantErr, err)
		}

		if updated {
			t.Fatal("expected workspace not to be updated")
		}
	})
}

func TestDelete(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		tests := []struct {
			name string
			ws   Workspace
			id   WorkItemID
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
				updated, err := tt.ws.delete(tt.id)
				if err != nil {
					t.Fatal(err)
				}

				if !updated {
					t.Fatalf("expected workspace to be updated")
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
			id      WorkItemID
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

				updated, err := ws.delete(tt.id)
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected err %v, got %v", tt.wantErr, err)
				}

				if updated {
					t.Fatal("expected workspace not to be updated")
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

				updated, err := ws.moveInDirection("1", dir)
				if err != nil {
					t.Fatal(err)
				}

				if updated {
					t.Fatal("expected workspace not to be updated")
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
			id        WorkItemID
			direction MoveDirection
			want      Workspace
		}{
			{
				name: "up",
				ws: workspace(
					column("Column", A, B),
				),
				id:        B.ID,
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
				direction: DirectionLeft,
				want: workspace(
					column("Column 1", C, A),
					column("Column 2", B),
				),
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				updated, err := tt.ws.moveInDirection(tt.id, tt.direction)
				if err != nil {
					t.Fatal(err)
				}

				if !updated {
					t.Fatalf("expected workspace to be updated")
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
			id        WorkItemID
			direction MoveDirection
			wantErr   error
		}{
			{
				name: "item ID",
				ws: workspace(
					column("Column", A),
				),
				id:        B.ID,
				direction: DirectionUp,
				wantErr:   ErrWorkItemNotFound,
			},
			{
				name: "move direction",
				ws: workspace(
					column("Column", A),
				),
				id:        A.ID,
				direction: MoveDirection("invalid"),
				wantErr:   ErrInvalidMoveDirection,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				updated, err := tt.ws.moveInDirection(tt.id, tt.direction)
				if err == nil {
					t.Fatalf("expected error %v, got nil", tt.wantErr)
				}

				if updated {
					t.Fatal("expected workspace not to be updated")
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
			name        string
			ws          Workspace
			id          WorkItemID
			insertPoint WorkItemInsertionPoint
			want        Workspace
			wantUpdated bool
		}{
			{
				name: "move between columns",
				ws: workspace(
					column("Column 1", A, B),
					column("Column 2", C),
				),
				id:          B.ID,
				insertPoint: insertPoint(1, 1),
				want: workspace(
					column("Column 1", A),
					column("Column 2", C, B),
				),
				wantUpdated: true,
			},
			{
				name: "move earlier in same column",
				ws: workspace(
					column("Column", A, B, C),
				),
				id:          C.ID,
				insertPoint: insertPoint(0, 0),
				want: workspace(
					column("Column", C, A, B),
				),
				wantUpdated: true,
			},
			{
				name: "move later in same column",
				ws: workspace(
					column("Column", A, B, C),
				),
				id:          A.ID,
				insertPoint: insertPoint(0, 2),
				want: workspace(
					column("Column", B, A, C),
				),
				wantUpdated: true,
			},
			{
				name: "stay in same position",
				ws: workspace(
					column("Column", A, B, C),
				),
				id:          A.ID,
				insertPoint: insertPoint(0, 0),
				want: workspace(
					column("Column", A, B, C),
				),
				wantUpdated: false,
			},
			{
				name: "same column move to end",
				ws: workspace(
					column("Column", A, B, C),
				),
				id:          A.ID,
				insertPoint: insertPoint(0, 3),
				want: workspace(
					column("Column", B, C, A),
				),
				wantUpdated: true,
			},
			{
				name: "same column move to beginning",
				ws: workspace(
					column("Column", A, B, C),
				),
				id:          C.ID,
				insertPoint: insertPoint(0, 0),
				want: workspace(
					column("Column", C, A, B),
				),
				wantUpdated: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				updated, err := tt.ws.moveToPosition(tt.id, tt.insertPoint)
				if err != nil {
					t.Fatalf("WorkItemMove() unexpected error: %v", err)
				}

				if tt.wantUpdated != updated {
					t.Fatalf("expected updated to be %v, got %v", tt.wantUpdated, updated)
				}

				if diff := cmp.Diff(tt.want, tt.ws); diff != "" {
					t.Errorf("workspace mismatch (-want +got):\n%s", diff)
				}
			})
		}
	})

	t.Run("invalid", func(t *testing.T) {
		tests := []struct {
			name        string
			ws          Workspace
			id          WorkItemID
			insertPoint WorkItemInsertionPoint
			wantErr     error
		}{
			{
				name: "to column",
				ws: workspace(
					column("Column", A),
				),
				id:          A.ID,
				insertPoint: insertPoint(9, 0),
				wantErr:     ErrInvalidPosition,
			},
			{
				name: "to index",
				ws: workspace(
					column("Column 1", A),
					column("Column 2"),
				),
				id:          A.ID,
				insertPoint: insertPoint(1, 2),
				wantErr:     ErrInvalidPosition,
			},
			{
				name: "equal positions are still validated",
				ws: workspace(
					column("Column", A),
				),
				id:          A.ID,
				insertPoint: insertPoint(9, 0),
				wantErr:     ErrInvalidPosition,
			},
			{
				name: "ID mismatch",
				ws: workspace(
					column("Column", A),
				),
				id:          B.ID,
				insertPoint: insertPoint(0, 0),
				wantErr:     ErrWorkItemNotFound,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				updated, err := tt.ws.moveToPosition(tt.id, tt.insertPoint)

				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}

				if updated {
					t.Fatal("expected workspace not to be updated")
				}
			})
		}
	})
}

func TestFindWorkItemPosition(t *testing.T) {
	tests := []struct {
		name    string
		ws      Workspace
		id      WorkItemID
		want    workItemPosition
		wantErr error
	}{
		{
			name: "first column",
			ws: workspace(
				column("One", A, B),
				column("Two", C),
			),
			id:   B.ID,
			want: position(0, 1),
		},
		{
			name: "second column",
			ws: workspace(
				column("One", A),
				column("Two", B, C),
			),
			id:   C.ID,
			want: position(1, 1),
		},
		{
			name:    "not found",
			ws:      workspace(column("One", A)),
			id:      "missing",
			wantErr: ErrWorkItemNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos, err := tt.ws.findWorkItemPosition(tt.id)

			if tt.wantErr != nil && errors.Is(err, tt.wantErr) {
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			if pos != tt.want {
				t.Errorf("expected position %v, got %v", tt.want, pos)
			}
		})
	}
}
