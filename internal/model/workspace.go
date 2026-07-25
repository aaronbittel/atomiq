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

func (wm *WorkspaceModel) WorkItemAdd(columnIdx int, workItemName string) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if columnIdx < 0 || columnIdx >= len(wm.Workspace.Columns) {
		return errors.New("illegal column access")
	}

	wm.Workspace.Columns[columnIdx].WorkItems = append(wm.Workspace.Columns[columnIdx].WorkItems, NewWorkItem(workItemName))
	return nil
}

func (wm *WorkspaceModel) WorkItemDelete(columnIdx int, workItemID string) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if columnIdx < 0 || columnIdx >= len(wm.Workspace.Columns) {
		return errors.New("illegal column access")
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
