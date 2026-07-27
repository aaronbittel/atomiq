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
func (wm *WorkspaceModel) WorkItemAdd(columnIdx int, name string) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	return wm.workspace.add(columnIdx, name)
}

// WorkItemDelete removes the item with itemID.
func (wm *WorkspaceModel) WorkItemDelete(itemID string) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	return wm.workspace.delete(itemID)
}

// WorkItemMoveDirection moves an item one visual step.
func (wm *WorkspaceModel) WorkItemMoveDirection(itemID string, direction MoveDirection) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	return wm.workspace.moveInDirection(itemID, direction)
}

// WorkItemMovePosition moves an item to an insertion position.
//
// Within the same column, moving an item to its own insertion position or the
// next insertion position is a no-op. For [A, B, C], moving B to index 1 or 2
// leaves the column unchanged.
func (wm *WorkspaceModel) WorkItemMovePosition(itemID string, to WorkItemInsertionPoint) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	return wm.workspace.moveToPosition(itemID, to)
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
