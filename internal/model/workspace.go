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

	item, err := wm.workItemGetUnlocked(from.ColumnIdx, from.ItemIdx)
	if err != nil {
		return err
	}

	if itemID != item.ID {
		return fmt.Errorf("%w: expected %q, got %q", ErrItemIDMismatch, itemID, item.ID)
	}

	to, err := wm.getToWorkItemPositionUnlocked(from, direction)
	if err != nil {
		return err
	}

	if from == to {
		return nil
	}

	return wm.moveWorkItemToPositionUnlocked(from, to)
}

// workItemGetUnlocked expects the mutex to be already locked
func (wm *WorkspaceModel) workItemGetUnlocked(columnIdx, itemIdx int) (WorkItem, error) {
	columns := wm.Workspace.Columns

	if !validSliceAccess(columnIdx, len(columns)) {
		return WorkItem{}, fmt.Errorf("%w: column index", ErrInvalidPosition)
	}

	if !validSliceAccess(itemIdx, len(columns[columnIdx].WorkItems)) {
		return WorkItem{}, fmt.Errorf("%w: index index", ErrInvalidPosition)
	}

	return wm.Workspace.Columns[columnIdx].WorkItems[itemIdx], nil
}

type WorkItemPosition struct {
	ColumnIdx int
	ItemIdx   int
}

func (wm *WorkspaceModel) WorkItemMovePosition(from, to WorkItemPosition) error {
	if from == to {
		return nil
	}

	wm.mu.Lock()
	defer wm.mu.Unlock()

	return wm.moveWorkItemToPositionUnlocked(from, to)
}

// moveWorkItemToPositionUnlocked expects the mutex to be already locked
func (wm *WorkspaceModel) moveWorkItemToPositionUnlocked(from, to WorkItemPosition) error {
	if from.ColumnIdx == to.ColumnIdx {
		return wm.moveWorkItemWithinColumnUnlocked(from.ColumnIdx, from.ItemIdx, to.ItemIdx)
	}

	return wm.moveWorkItemBetweenColumnsUnlocked(from, to)
}

// moveWorkItemWithinColumnUnlocked expects the mutex to be already locked
func (wm *WorkspaceModel) moveWorkItemWithinColumnUnlocked(columnIdx, fromIdx, toIdx int) error {
	if !validSliceAccess(columnIdx, len(wm.Workspace.Columns)) {
		return fmt.Errorf("%w: source column %d", ErrInvalidPosition, columnIdx)
	}
	items := wm.Workspace.Columns[columnIdx].WorkItems
	if !validSliceAccess(fromIdx, len(items)) {
		return fmt.Errorf("%w: source index %d", ErrInvalidPosition, fromIdx)
	}
	if !validSliceAccess(toIdx, len(items)) {
		return fmt.Errorf("%w: source index %d", ErrInvalidPosition, toIdx)
	}

	item := items[fromIdx]

	items = slices.Delete(items, fromIdx, fromIdx+1)
	items = slices.Insert(items, toIdx, item)

	wm.Workspace.Columns[columnIdx].WorkItems = items

	return nil
}

// moveWorkItemBetweenColumnsUnlocked expects the mutex to be already locked
func (wm *WorkspaceModel) moveWorkItemBetweenColumnsUnlocked(from, to WorkItemPosition) error {
	if !validSliceAccess(from.ColumnIdx, len(wm.Workspace.Columns)) {
		return fmt.Errorf("%w: source column %d", ErrInvalidPosition, from.ColumnIdx)
	}
	fromItems := wm.Workspace.Columns[from.ColumnIdx].WorkItems
	if !validSliceAccess(from.ItemIdx, len(fromItems)) {
		return fmt.Errorf("%w: source index %d", ErrInvalidPosition, from.ItemIdx)
	}
	if !validSliceAccess(to.ColumnIdx, len(wm.Workspace.Columns)) {
		return fmt.Errorf("%w: destination column %d", ErrInvalidPosition, to.ColumnIdx)
	}
	toItems := wm.Workspace.Columns[to.ColumnIdx].WorkItems
	if to.ItemIdx < 0 || to.ItemIdx > len(toItems) {
		return fmt.Errorf("%w: destination index %d", ErrInvalidPosition, to.ItemIdx)
	}

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

// getToWorkItemPositionUnlocked expects the mutex to be already locked
func (wm *WorkspaceModel) getToWorkItemPositionUnlocked(from WorkItemPosition, direction MoveDirection) (WorkItemPosition, error) {
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
			ItemIdx:   from.ItemIdx + 1,
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
