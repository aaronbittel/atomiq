package model

import (
	"fmt"
	"sync"
)

// WorkspaceModel owns and synchronizes the mutable workspace state.
//
// Mutation methods use optimistic concurrency control. The supplied expected revision
// must match the target workspace revision, otherwise they return ErrRevisionConflict.
// Each workspace has its own revision, which is incremented only when an operation
// changes that workspace; successful no-ops leave it unchanged.
type WorkspaceModel struct {
	mu              sync.RWMutex
	workspaces      map[WorkspaceID]revisionedWorkspace
	rootWorkspaceID WorkspaceID
}

// revisionedWorkspace stores one workspace with its optimistic concurrency revision.
type revisionedWorkspace struct {
	workspace        Workspace
	revision         uint64
	parentID         *WorkspaceID
	parentWorkItemID *WorkItemID
}

func newRevisionedWorkspace(ws Workspace, parentID *WorkspaceID, parentWorkItemID *WorkItemID) revisionedWorkspace {
	return revisionedWorkspace{
		workspace:        ws.clone(),
		parentID:         parentID,
		parentWorkItemID: parentWorkItemID,
	}
}

// NewWorkspaceModel creates a WorkspaceModel that owns a deep copy of ws.
func NewWorkspaceModel(root Workspace) *WorkspaceModel {
	return &WorkspaceModel{
		workspaces: map[WorkspaceID]revisionedWorkspace{
			root.id: newRevisionedWorkspace(root, nil, nil),
		},
		rootWorkspaceID: root.id,
	}
}

const rootWorkspaceTitle = "Root Workspace"

// WorkspaceView returns a detached snapshot of the current workspace.
//
// The snapshot includes the revision required for subsequent mutation requests. Its
// slices do not share backing arrays with the model, so callers may render or inspect
// it without holding the model lock.
func (wm *WorkspaceModel) WorkspaceView(workspaceID WorkspaceID) (WorkspaceView, error) {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	rws, found := wm.workspaces[workspaceID]
	if !found {
		return WorkspaceView{}, ErrWorkspaceNotFound
	}

	title := rootWorkspaceTitle

	if rws.parentID == nil && rws.parentWorkItemID != nil {
		panic("workspace has parent work item without parent workspace")
	}

	if rws.parentID != nil && rws.parentWorkItemID == nil {
		panic("workspace has parent workspace without parent work item")
	}

	if rws.parentID != nil && rws.parentWorkItemID != nil {
		parentWorkspace, found := wm.workspaces[*rws.parentID]
		if !found {
			panic("workspace parent link points to missing parent workspace")
		}

		parentWorkItem, err := parentWorkspace.workspace.findWorkItem(*rws.parentWorkItemID)
		if err != nil {
			return WorkspaceView{}, err
		}
		title = parentWorkItem.name
	}

	return WorkspaceView{
		Columns:  rws.workspace.view(),
		Revision: rws.revision,
		ID:       workspaceID,
		ParentID: cloneWorkspaceIDPtr(rws.parentID),
		Title:    title,
	}, nil
}

// WorkspaceRootID returns the ID of the top-level workspace.
func (wm *WorkspaceModel) WorkspaceRootID() WorkspaceID {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	return wm.rootWorkspaceID
}

// WorkItemZoom creates or returns the child workspace owned by itemID.
//
// Creating a child workspace updates the parent workspace revision. Returning an
// existing child workspace is a successful no-op and leaves the parent revision
// unchanged.
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
		wm.workspaces[ws.id] = newRevisionedWorkspace(ws, &workspaceID, &itemID)

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
//
// It returns the generated ID of the new item when the workspace was updated.
func (wm *WorkspaceModel) WorkItemAdd(workspaceID WorkspaceID, expectedRevision uint64, columnIdx int, name string) (WorkItemID, error) {
	var itemID WorkItemID
	err := wm.mutate(workspaceID, expectedRevision, func(w *Workspace) (updated bool, err error) {
		itemID, updated, err = w.add(columnIdx, name)
		return updated, err
	})

	if err != nil {
		return "", err
	}

	return itemID, nil
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

	rws, found := wm.workspaces[workspaceID]
	if !found {
		return ErrWorkspaceNotFound
	}

	if rws.revision != expectedRevision {
		return fmt.Errorf("%w: expected %d, actual %d", ErrRevisionConflict, expectedRevision, rws.revision)
	}

	updated, err := mutation(&rws.workspace)
	if err != nil {
		return err
	}

	if updated {
		rws.revision++
		// update workspaces map, because lookup only returns a copy
		wm.workspaces[workspaceID] = rws
	}

	return nil
}

func cloneWorkspaceIDPtr(id *WorkspaceID) *WorkspaceID {
	if id == nil {
		return nil
	}

	cloned := *id
	return &cloned
}
