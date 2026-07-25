package model

import (
	"crypto/rand"
	"errors"
	"slices"
	"sync"
)

type WorkspaceModel struct {
	mu        sync.RWMutex
	Workspace Workspace
}

type Workspace struct {
	Columns []Column
}

type Column struct {
	Name      string
	WorkItems []WorkItem
}

type WorkItem struct {
	ID   string
	Name string
}

func (wm *WorkspaceModel) WorkspaceView() Workspace {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	columns := make([]Column, len(wm.Workspace.Columns))
	for i, col := range wm.Workspace.Columns {
		columns[i] = Column{
			Name:      col.Name,
			WorkItems: append([]WorkItem(nil), col.WorkItems...),
		}
	}

	return Workspace{Columns: columns}
}

var ErrInvalidColumn = errors.New("invalid column")

func (wm *WorkspaceModel) WorkItemAdd(columnIdx int, workItemName string) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if validSliceAccess(columnIdx, len(wm.Workspace.Columns)) {
		return ErrInvalidColumn
	}

	wm.Workspace.Columns[columnIdx].WorkItems = append(wm.Workspace.Columns[columnIdx].WorkItems, NewWorkItem(workItemName))
	return nil
}

func (wm *WorkspaceModel) WorkItemDelete(columnIdx int, workItemID string) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if validSliceAccess(columnIdx, len(wm.Workspace.Columns)) {
		return ErrInvalidColumn
	}

	column := &wm.Workspace.Columns[columnIdx]
	column.WorkItems = slices.DeleteFunc(column.WorkItems, func(wi WorkItem) bool {
		return wi.ID == workItemID
	})

	return nil
}

func NewWorkItem(name string) WorkItem {
	return WorkItem{
		ID:   rand.Text()[:8],
		Name: name,
	}
}

func validSliceAccess(idx, length int) bool {
	return idx < 0 || idx >= length
}
