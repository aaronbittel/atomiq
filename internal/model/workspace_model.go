package model

import (
	"crypto/rand"
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

// NewWorkspaceModel creates a new NewWorkspaceModel.
func NewWorkspaceModel(ws Workspace) *WorkspaceModel {
	return &WorkspaceModel{
		workspace: ws.clone(),
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
func (wm *WorkspaceModel) WorkItemAdd(expectedRevision uint64, columnIdx int, name string) error {
	return wm.mutate(expectedRevision, func(w *Workspace) (bool, error) {
		return w.add(columnIdx, name)
	})
}

// WorkItemDelete removes the item with itemID.
func (wm *WorkspaceModel) WorkItemDelete(expectedRevision uint64, itemID string) error {
	return wm.mutate(expectedRevision, func(w *Workspace) (bool, error) {
		return w.delete(itemID)
	})
}

// WorkItemMoveDirection moves an item one visual step.
func (wm *WorkspaceModel) WorkItemMoveDirection(expectedRevision uint64, itemID string, direction MoveDirection) error {
	return wm.mutate(expectedRevision, func(w *Workspace) (bool, error) {
		return w.moveInDirection(itemID, direction)
	})
}

// WorkItemMovePosition moves an item to an insertion position.
//
// Within the same column, moving an item to its own insertion position or the
// next insertion position is a no-op. For [A, B, C], moving B to index 1 or 2
// leaves the column unchanged.
func (wm *WorkspaceModel) WorkItemMovePosition(expectedRevision uint64, itemID string, insertPoint WorkItemInsertionPoint) error {
	return wm.mutate(expectedRevision, func(w *Workspace) (bool, error) {
		return w.moveToPosition(itemID, insertPoint)
	})
}
func (wm *WorkspaceModel) mutate(expectedRevision uint64, mutation func(*Workspace) (bool, error)) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if wm.workspace.revision != expectedRevision {
		return fmt.Errorf(
			"%w: expected %d, actual %d",
			ErrRevisionConflict,
			expectedRevision,
			wm.workspace.revision,
		)
	}

	updated, err := mutation(&wm.workspace)
	if err != nil {
		return err
	}

	if updated {
		wm.workspace.revision++
	}

	return nil
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
