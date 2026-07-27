package model

import (
	"crypto/rand"
	"errors"
	"fmt"
	"sync"
)

// WorkspaceModel owns the mutable workspace state.
//
// Use its methods instead of mutating Workspace directly when handling
// concurrent requests.
type WorkspaceModel struct {
	mu        sync.RWMutex
	workspace Workspace
}

// MoveDirection describes a visual move in the rendered workspace.
// Up and down move within a column; left and right move to the end of the
// neighboring column.
type MoveDirection string

const (
	// WorkItemIDLength is the number of characters generated for work item IDs.
	WorkItemIDLength = 8

	DirectionUp    MoveDirection = "up"
	DirectionDown  MoveDirection = "down"
	DirectionRight MoveDirection = "right"
	DirectionLeft  MoveDirection = "left"
)

var (
	ErrInvalidColumn        = errors.New("invalid column")
	ErrInvalidPosition      = errors.New("invalid work item position")
	ErrInvalidWorkItemName  = errors.New("invalid work item name")
	ErrItemIDMismatch       = errors.New("item ID mismatch")
	ErrInvalidMoveDirection = errors.New("invalid move direction")
)

// NewWorkspaceModel creates a new NewWorkspaceModel.
func NewWorkspaceModel(workspace Workspace) *WorkspaceModel {
	// TODO: clone workspace
	return &WorkspaceModel{
		workspace: workspace,
	}
}

// WorkspaceView returns a snapshot of the current workspace.
// The returned slices do not share backing arrays with the model, so callers
// can render or inspect the snapshot without holding the model lock.
func (wm *WorkspaceModel) WorkspaceView() WorkspaceView {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	return wm.workspace.view()
}

// WorkItemAdd trims and appends a new item to the selected column.
func (wm *WorkspaceModel) WorkItemAdd(columnIdx int, name string) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	return wm.workspace.add(columnIdx, name)
}

// WorkItemDelete removes the item at pos.
//
// The position must point at an existing item whose ID matches itemID. Invalid
// positions return ErrInvalidPosition; stale positions that point at a different
// item return ErrItemIDMismatch.
func (wm *WorkspaceModel) WorkItemDelete(id string, pos WorkItemPosition) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	return wm.workspace.delete(id, pos)
}

// WorkItemMoveDirection moves an item one visual step.
//
// The from position must point at the item currently identified by itemID. This
// protects requests from acting on a stale position after the workspace changed.
func (wm *WorkspaceModel) WorkItemMoveDirection(id string, from WorkItemPosition, direction MoveDirection) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	return wm.workspace.moveInDirection(id, from, direction)
}

// wm.mu must be locked.
func (wm *WorkspaceModel) isValidFromPosition(pos WorkItemPosition) error {
	columns := wm.workspace.Columns

	if !validSliceAccess(pos.ColumnIdx, len(columns)) {
		return errors.New("from column index out of bounds")
	}

	if !validSliceAccess(pos.ItemIdx, len(columns[pos.ColumnIdx].WorkItems)) {
		return errors.New("from item index out of bounds")
	}

	return nil
}

// WorkItemPosition addresses a column plus an item or insertion slot.
type WorkItemPosition struct {
	ColumnIdx int
	// ItemIdx is an insertion index.
	//
	// In a column [A, B, C], index 0 inserts before A, index 1 inserts between
	// A and B, and index 3 appends after C.
	ItemIdx int
}

// WorkItemMovePosition moves an item to an insertion position.
//
// The from position must point at an existing item. The to position may point
// between items or to len(column.WorkItems), which means append.
//
// Within the same column, moving an item to its own insertion position or the
// next insertion position is a no-op. For [A, B, C], moving B to index 1 or 2
// leaves the column unchanged.
func (wm *WorkspaceModel) WorkItemMovePosition(from, to WorkItemPosition) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	return wm.workspace.moveToPosition(from, to)
}

// NewWorkItem creates a work item with a generated short ID.
func NewWorkItem(name string) WorkItem {
	return WorkItem{
		ID:   rand.Text()[:WorkItemIDLength],
		Name: name,
	}
}

// ParseMoveDirection converts form input into a MoveDirection.
func ParseMoveDirection(s string) (MoveDirection, error) {
	switch s {
	case "up":
		return DirectionUp, nil
	case "down":
		return DirectionDown, nil
	case "right":
		return DirectionRight, nil
	case "left":
		return DirectionLeft, nil
	default:
		return "", fmt.Errorf("%w: %q", ErrInvalidMoveDirection, s)
	}
}
