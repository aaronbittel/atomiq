package model

import (
	"errors"
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
	WorkItems []string
}

func (wm *WorkspaceModel) AddWorkItem(columnIdx int, workItemName string) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if columnIdx < 0 || columnIdx >= len(wm.Workspace.Columns) {
		return errors.New("illegal column access")
	}

	wm.Workspace.Columns[columnIdx].WorkItems = append(wm.Workspace.Columns[columnIdx].WorkItems, workItemName)
	return nil
}
