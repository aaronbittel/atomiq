package model

import (
	"crypto/rand"
	"errors"
	"fmt"
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

type MoveDirection string

const (
	DirectionUp    MoveDirection = "up"
	DirectionDown  MoveDirection = "down"
	DirectionRight MoveDirection = "right"
	DirectionLeft  MoveDirection = "left"
)

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

var (
	ErrInvalidColumn   = errors.New("invalid column")
	ErrInvalidPosition = errors.New("invalid work item position")
)

func (wm *WorkspaceModel) WorkItemAdd(columnIdx int, workItemName string) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if !validSliceAccess(columnIdx, len(wm.Workspace.Columns)) {
		return ErrInvalidColumn
	}

	wm.Workspace.Columns[columnIdx].WorkItems = append(wm.Workspace.Columns[columnIdx].WorkItems, NewWorkItem(workItemName))
	return nil
}

func (wm *WorkspaceModel) WorkItemDelete(columnIdx int, workItemID string) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if !validSliceAccess(columnIdx, len(wm.Workspace.Columns)) {
		return ErrInvalidColumn
	}

	column := &wm.Workspace.Columns[columnIdx]
	column.WorkItems = slices.DeleteFunc(column.WorkItems, func(wi WorkItem) bool {
		return wi.ID == workItemID
	})

	return nil
}

var ErrItemIDMismatch = errors.New("item ID mismatch")

func (wm *WorkspaceModel) WorkItemMoveDirection(itemID string, from WorkItemPosition, direction MoveDirection) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if err := wm.isValidFromPosition(from); err != nil {
		return err
	}

	item := wm.Workspace.Columns[from.ColumnIdx].WorkItems[from.ItemIdx]
	if itemID != item.ID {
		return fmt.Errorf("%w: expected %q, got %q", ErrItemIDMismatch, itemID, item.ID)
	}

	to, err := wm.getToWorkItemPosition(from, direction)
	if err != nil {
		return err
	}

	if err := wm.isValidToPosition(to); err != nil {
		return err
	}

	if from == to {
		return nil
	}

	return wm.moveWorkItemToPosition(from, to)
}

func (wm *WorkspaceModel) isValidFromPosition(pos WorkItemPosition) error {
	columns := wm.Workspace.Columns

	if !validSliceAccess(pos.ColumnIdx, len(columns)) {
		return fmt.Errorf("%w: from column index", ErrInvalidPosition)
	}

	if !validSliceAccess(pos.ItemIdx, len(columns[pos.ColumnIdx].WorkItems)) {
		return fmt.Errorf("%w: from item index", ErrInvalidPosition)
	}

	return nil
}

func (wm *WorkspaceModel) isValidToPosition(pos WorkItemPosition) error {
	columns := wm.Workspace.Columns

	if !validSliceAccess(pos.ColumnIdx, len(columns)) {
		return fmt.Errorf("%w: to column index", ErrInvalidPosition)
	}

	if !validSliceAccess(pos.ItemIdx, len(columns[pos.ColumnIdx].WorkItems)+1) {
		return fmt.Errorf("%w: to item index", ErrInvalidPosition)
	}

	return nil
}

type WorkItemPosition struct {
	ColumnIdx int
	ItemIdx   int
}

func (wm *WorkspaceModel) WorkItemMovePosition(from, to WorkItemPosition) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if err := wm.isValidFromPosition(from); err != nil {
		return err
	}

	if err := wm.isValidToPosition(to); err != nil {
		return err
	}

	if from == to {
		return nil
	}

	return wm.moveWorkItemToPosition(from, to)
}

// moveWorkItemToPosition expects the mutex to be already locked
func (wm *WorkspaceModel) moveWorkItemToPosition(from, to WorkItemPosition) error {
	if from.ColumnIdx == to.ColumnIdx {
		return wm.moveWorkItemWithinColumn(from.ColumnIdx, from.ItemIdx, to.ItemIdx)
	}

	return wm.moveWorkItemBetweenColumns(from, to)
}

// moveWorkItemWithinColumn expects the mutex to be already locked
func (wm *WorkspaceModel) moveWorkItemWithinColumn(columnIdx, fromIdx, toIdx int) error {
	items := wm.Workspace.Columns[columnIdx].WorkItems
	item := items[fromIdx]

	items = slices.Delete(items, fromIdx, fromIdx+1)
	if fromIdx < toIdx {
		toIdx -= 1
	}
	items = slices.Insert(items, toIdx, item)

	wm.Workspace.Columns[columnIdx].WorkItems = items

	return nil
}

// moveWorkItemBetweenColumns expects the mutex to be already locked
func (wm *WorkspaceModel) moveWorkItemBetweenColumns(from, to WorkItemPosition) error {
	fromItems := wm.Workspace.Columns[from.ColumnIdx].WorkItems
	toItems := wm.Workspace.Columns[to.ColumnIdx].WorkItems

	item := fromItems[from.ItemIdx]
	fromItems = slices.Delete(fromItems, from.ItemIdx, from.ItemIdx+1)
	toItems = slices.Insert(toItems, to.ItemIdx, item)

	wm.Workspace.Columns[from.ColumnIdx].WorkItems = fromItems
	wm.Workspace.Columns[to.ColumnIdx].WorkItems = toItems

	return nil
}

func NewWorkItem(name string) WorkItem {
	return WorkItem{
		ID:   rand.Text()[:8],
		Name: name,
	}
}

var ErrInvalidMoveDirection = errors.New("invalid move direction")

// getToWorkItemPosition expects the mutex to be already locked
func (wm *WorkspaceModel) getToWorkItemPosition(from WorkItemPosition, direction MoveDirection) (WorkItemPosition, error) {
	var to WorkItemPosition
	switch direction {
	case DirectionUp:
		if from.ItemIdx == 0 {
			return from, nil
		}
		to = WorkItemPosition{
			ColumnIdx: from.ColumnIdx,
			ItemIdx:   from.ItemIdx - 1,
		}
	case DirectionDown:
		items := wm.Workspace.Columns[from.ColumnIdx].WorkItems
		if from.ItemIdx == len(items)-1 {
			return from, nil
		}
		to = WorkItemPosition{
			ColumnIdx: from.ColumnIdx,
			ItemIdx:   from.ItemIdx + 2,
		}
	case DirectionRight:
		columns := wm.Workspace.Columns
		if from.ColumnIdx == len(columns)-1 {
			return from, nil
		}

		toColumnIdx := from.ColumnIdx + 1

		to = WorkItemPosition{
			ColumnIdx: toColumnIdx,
			ItemIdx:   len(columns[toColumnIdx].WorkItems),
		}
	case DirectionLeft:
		if from.ColumnIdx == 0 {
			return from, nil
		}

		columns := wm.Workspace.Columns
		toColumnIdx := from.ColumnIdx - 1

		to = WorkItemPosition{
			ColumnIdx: toColumnIdx,
			ItemIdx:   len(columns[toColumnIdx].WorkItems),
		}
	default:
		return WorkItemPosition{}, fmt.Errorf("%w: %q", ErrInvalidMoveDirection, direction)
	}

	return to, nil
}

func validSliceAccess(idx, length int) bool {
	return idx >= 0 && idx < length
}

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
		return "", fmt.Errorf("unknown move direction %q", s)
	}
}
