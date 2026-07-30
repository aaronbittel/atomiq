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
	mu              sync.RWMutex
	workspaces      map[WorkspaceID]Workspace
	rootWorkspaceID WorkspaceID
	revision        uint64
}

// NewWorkspaceModel creates a WorkspaceModel that owns a deep copy of ws.
func NewWorkspaceModel(ws Workspace) *WorkspaceModel {
	return &WorkspaceModel{
		workspaces: map[WorkspaceID]Workspace{
			ws.id: ws.clone(),
		},
		rootWorkspaceID: ws.id,
	}
}

// WorkspaceView returns a detached snapshot of the current workspace.
//
// The snapshot includes the revision required for subsequent mutation requests. Its
// slices do not share backing arrays with the model, so callers may render or inspect
// it without holding the model lock.
func (wm *WorkspaceModel) WorkspaceView(workspaceID WorkspaceID) (WorkspaceView, error) {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	ws, found := wm.workspaces[workspaceID]
	if !found {
		return WorkspaceView{}, ErrWorkspaceNotFound
	}

	return WorkspaceView{
		Columns:  ws.view(),
		Revision: wm.revision,
		ID:       workspaceID,
	}, nil
}

func (wm *WorkspaceModel) WorkspaceRootID() WorkspaceID {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	return wm.rootWorkspaceID
}

func (wm *WorkspaceModel) WorkItemZoom(workspaceID WorkspaceID, itemID WorkItemID, expectedRevision uint64) (WorkspaceID, error) {
	var childID WorkspaceID

	err := wm.mutate(workspaceID, expectedRevision, func(w *Workspace) (updated bool, err error) {
		var found bool

		childID, found, err = w.getChildWorkspaceID(itemID)
		if err != nil {
			return false, err
		}

		if found {
			return false, nil
		}

		ws := defaultWorkspace()
		wm.workspaces[ws.id] = ws

		w.attachChildWorkspaceID(itemID, ws.id)

		childID = ws.id
		return true, nil
	})

	if err != nil {
		return "", err
	}

	return childID, nil
}

// WorkItemAdd trims and appends a new item to the selected column.
func (wm *WorkspaceModel) WorkItemAdd(workspaceID WorkspaceID, expectedRevision uint64, columnIdx int, name string) error {
	return wm.mutate(workspaceID, expectedRevision, func(w *Workspace) (bool, error) {
		return w.add(columnIdx, name)
	})
}

// WorkItemDelete removes the item with itemID.
func (wm *WorkspaceModel) WorkItemDelete(workspaceID WorkspaceID, expectedRevision uint64, itemID WorkItemID) error {
	return wm.mutate(workspaceID, expectedRevision, func(w *Workspace) (bool, error) {
		return w.delete(itemID)
	})
}

// WorkItemMoveDirection moves an item one visual step.
//
// Moves blocked by a workspace boundary are successful no-ops and leave the revision
// unchanged.
func (wm *WorkspaceModel) WorkItemMoveDirection(workspaceID WorkspaceID, expectedRevision uint64, itemID WorkItemID, direction MoveDirection) error {
	return wm.mutate(workspaceID, expectedRevision, func(w *Workspace) (bool, error) {
		return w.moveInDirection(itemID, direction)
	})
}

// WorkItemMovePosition moves an item to an insertion position.
//
// Within the same column, moving an item to its current insertion position or the
// following insertion position is a no-op. For [A, B, C], moving B to index 1 or 2
// leaves the column and revision unchanged.
func (wm *WorkspaceModel) WorkItemMovePosition(workspaceID WorkspaceID, expectedRevision uint64, itemID WorkItemID, insertPoint WorkItemInsertionPoint) error {
	return wm.mutate(workspaceID, expectedRevision, func(w *Workspace) (bool, error) {
		return w.moveToPosition(itemID, insertPoint)
	})
}

func defaultWorkspace() Workspace {
	return NewWorkspace(
		NewColumn("Backlog"),
		NewColumn("In Progress"),
		NewColumn("Done"),
	)
}

// mutate applies mutation when expectedRevision matches the current revision.
func (wm *WorkspaceModel) mutate(workspaceID WorkspaceID, expectedRevision uint64, mutation func(*Workspace) (bool, error)) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	ws, found := wm.workspaces[workspaceID]
	if !found {
		return ErrWorkspaceNotFound
	}

	if wm.revision != expectedRevision {
		return fmt.Errorf("%w: expected %d, actual %d", ErrRevisionConflict, expectedRevision, wm.revision)
	}

	updated, err := mutation(&ws)
	if err != nil {
		return err
	}

	if updated {
		wm.revision++
		// update workspaces map, because lookup only returns a copy
		wm.workspaces[workspaceID] = ws
	}

	return nil
}
