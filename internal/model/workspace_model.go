package model

import (
	"fmt"
	"sync"
)

// WorkspaceModel owns and synchronizes the mutable workspace state.
//
// Mutation methods use optimistic concurrency control. The supplied expected revision
// must match the current workspace revision, otherwise they return ErrRevisionConflict.
// The revision is incremented only when an operation changes the workspace; successful
// no-ops leave it unchanged.
type WorkspaceModel struct {
	mu        sync.RWMutex
	workspace Workspace
	revision  uint64
}

// MoveDirection describes a visual move in the rendered workspace.
// Up and down move within a column; left and right move to the end of the
// neighboring column.
type MoveDirection string

const (
	// WorkItemIDLength is the number of characters generated for work item IDs.
	WorkItemIDLength = 8
	// WorkspaceIDLength is the number of characters generated for workspace IDs.
	WorkspaceIDLength = 8

	DirectionUp    MoveDirection = "up"
	DirectionDown  MoveDirection = "down"
	DirectionRight MoveDirection = "right"
	DirectionLeft  MoveDirection = "left"
)

// NewWorkspaceModel creates a WorkspaceModel that owns a deep copy of ws.
func NewWorkspaceModel(ws Workspace) *WorkspaceModel {
	return &WorkspaceModel{
		workspace: ws.clone(),
	}
}

// WorkspaceView returns a detached snapshot of the current workspace, including the
// revision required for subsequent mutation requests.
//
// The returned slices do not share backing arrays with the model, so callers may render
// or inspect the snapshot without holding the model lock.
func (wm *WorkspaceModel) WorkspaceView() WorkspaceView {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	return WorkspaceView{
		Columns:  wm.workspace.view(),
		Revision: wm.revision,
	}
}

// WorkItemAdd trims and appends a new item to the selected column.
//
// It returns ErrRevisionConflict when expectedRevision does not match the current
// revision. A successful addition increments the revision.
func (wm *WorkspaceModel) WorkItemAdd(expectedRevision uint64, columnIdx int, name string) error {
	return wm.mutate(expectedRevision, func(w *Workspace) (bool, error) {
		return w.add(columnIdx, name)
	})
}

// WorkItemDelete removes the item with itemID.
//
// It returns ErrRevisionConflict when expectedRevision does not match the current
// revision. A successful deletion increments the revision.
func (wm *WorkspaceModel) WorkItemDelete(expectedRevision uint64, itemID WorkItemID) error {
	return wm.mutate(expectedRevision, func(w *Workspace) (bool, error) {
		return w.delete(itemID)
	})
}

// WorkItemMoveDirection moves an item one visual step.
//
// Moves that are blocked by a workspace boundary are successful no-ops and leave the
// revision unchanged.
func (wm *WorkspaceModel) WorkItemMoveDirection(expectedRevision uint64, itemID WorkItemID, direction MoveDirection) error {
	return wm.mutate(expectedRevision, func(w *Workspace) (bool, error) {
		return w.moveInDirection(itemID, direction)
	})
}

// WorkItemMovePosition moves an item to an insertion position.
//
// Within the same column, moving an item to its current insertion position or the
// following insertion position is a no-op. For [A, B, C], moving B to index 1 or 2
// leaves the column and revision unchanged.
func (wm *WorkspaceModel) WorkItemMovePosition(expectedRevision uint64, itemID WorkItemID, insertPoint WorkItemInsertionPoint) error {
	return wm.mutate(expectedRevision, func(w *Workspace) (bool, error) {
		return w.moveToPosition(itemID, insertPoint)
	})
}
func (wm *WorkspaceModel) mutate(expectedRevision uint64, mutation func(*Workspace) (bool, error)) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if wm.revision != expectedRevision {
		return fmt.Errorf("%w: expected %d, actual %d", ErrRevisionConflict, expectedRevision, wm.revision)
	}

	updated, err := mutation(&wm.workspace)
	if err != nil {
		return err
	}

	if updated {
		wm.revision++
	}

	return nil
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
