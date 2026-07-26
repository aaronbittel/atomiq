package model

import (
	"crypto/rand"
	"errors"
	"fmt"
	"slices"
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

// Workspace is the board shown by the application.
type Workspace struct {
	Columns []Column
}

// Column groups work items under one name.
type Column struct {
	Name      string
	WorkItems []WorkItem
}

// WorkItem is one task card in a column.
type WorkItem struct {
	ID   string
	Name string
}

// MoveDirection describes a visual move in the rendered workspace.
// Up and down move within a column; left and right move to the end of the
// neighboring column.
type MoveDirection string

const (
	DirectionUp    MoveDirection = "up"
	DirectionDown  MoveDirection = "down"
	DirectionRight MoveDirection = "right"
	DirectionLeft  MoveDirection = "left"
)

// NewWorkspaceModel creates a new NewWorkspaceModel.
func NewWorkspaceModel(workspace Workspace) *WorkspaceModel {
	return &WorkspaceModel{
		workspace: workspace,
	}
}

// WorkspaceView returns a snapshot of the workspace.
// The returned slices do not share backing arrays with the model, so callers
// can render or inspect the snapshot without holding the model lock.
func (wm *WorkspaceModel) WorkspaceView() Workspace {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	columns := make([]Column, len(wm.workspace.Columns))
	for i, col := range wm.workspace.Columns {
		columns[i] = Column{
			Name:      col.Name,
			WorkItems: append([]WorkItem(nil), col.WorkItems...),
		}
	}

	return Workspace{Columns: columns}
}

var (
	ErrInvalidColumn        = errors.New("invalid column")
	ErrInvalidPosition      = errors.New("invalid work item position")
	ErrItemIDMismatch       = errors.New("item ID mismatch")
	ErrInvalidMoveDirection = errors.New("invalid move direction")
)

// WorkItemAdd appends a new item to the selected column.
func (wm *WorkspaceModel) WorkItemAdd(columnIdx int, workItemName string) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if !validSliceAccess(columnIdx, len(wm.workspace.Columns)) {
		return ErrInvalidColumn
	}

	wm.workspace.Columns[columnIdx].WorkItems = append(wm.workspace.Columns[columnIdx].WorkItems, NewWorkItem(workItemName))
	return nil
}

// WorkItemDelete removes all items with workItemID from the selected column.
func (wm *WorkspaceModel) WorkItemDelete(columnIdx int, workItemID string) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if !validSliceAccess(columnIdx, len(wm.workspace.Columns)) {
		return ErrInvalidColumn
	}

	column := &wm.workspace.Columns[columnIdx]
	column.WorkItems = slices.DeleteFunc(column.WorkItems, func(wi WorkItem) bool {
		return wi.ID == workItemID
	})

	return nil
}

// WorkItemMoveDirection moves an item one visual step.
//
// The from position must point at the item currently identified by itemID. This
// protects requests from acting on a stale position after the workspace changed.
func (wm *WorkspaceModel) WorkItemMoveDirection(itemID string, from WorkItemPosition, direction MoveDirection) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if err := wm.isValidFromPosition(from); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidPosition, err)
	}

	item := wm.workspace.Columns[from.ColumnIdx].WorkItems[from.ItemIdx]
	if itemID != item.ID {
		return fmt.Errorf("%w: expected %q, got %q", ErrItemIDMismatch, itemID, item.ID)
	}

	to, err := wm.getToWorkItemPosition(from, direction)
	if err != nil {
		return err
	}

	if err := wm.isValidToPosition(to); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidPosition, err)
	}

	if from == to {
		return nil
	}

	return wm.moveWorkItemToPosition(from, to)
}

func (wm *WorkspaceModel) isValidFromPosition(pos WorkItemPosition) error {
	columns := wm.workspace.Columns

	if !validSliceAccess(pos.ColumnIdx, len(columns)) {
		return errors.New("from column index out of bounds")
	}

	if !validSliceAccess(pos.ItemIdx, len(columns[pos.ColumnIdx].WorkItems)) {
		return errors.New("from item index out of bounds")
	}

	return nil
}

func (wm *WorkspaceModel) isValidToPosition(pos WorkItemPosition) error {
	columns := wm.workspace.Columns

	if !validSliceAccess(pos.ColumnIdx, len(columns)) {
		return errors.New("to column index out of bounds")
	}

	if !validSliceAccess(pos.ItemIdx, len(columns[pos.ColumnIdx].WorkItems)+1) {
		return errors.New("to item index out of bounds")
	}

	return nil
}

// WorkItemPosition addresses a column plus an item or insertion slot.
type WorkItemPosition struct {
	ColumnIdx int
	// ItemIdx is an insertion index.
	//
	// In a column [A, B, C], index 0 inserts before A, index 1 inserts between
	// A and B, and index 3 appends after C.
	ItemIdx int
}

// WorkItemMovePosition moves an item to an insertion position.
//
// The from position must point at an existing item. The to position may point
// between items or to len(column.WorkItems), which means append.
//
// Within the same column, moving an item to its own insertion position or the
// next insertion position is a no-op. For [A, B, C], moving B to index 1 or 2
// leaves the column unchanged.
func (wm *WorkspaceModel) WorkItemMovePosition(from, to WorkItemPosition) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if err := wm.isValidFromPosition(from); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidPosition, err)
	}

	if err := wm.isValidToPosition(to); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidPosition, err)
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
	items := wm.workspace.Columns[columnIdx].WorkItems
	item := items[fromIdx]

	items = slices.Delete(items, fromIdx, fromIdx+1)
	if fromIdx < toIdx {
		// toIdx refers to the original column; deleting an earlier item shifts
		// the insertion position one slot left.
		toIdx -= 1
	}
	items = slices.Insert(items, toIdx, item)

	wm.workspace.Columns[columnIdx].WorkItems = items

	return nil
}

// moveWorkItemBetweenColumns expects the mutex to be already locked
func (wm *WorkspaceModel) moveWorkItemBetweenColumns(from, to WorkItemPosition) error {
	fromItems := wm.workspace.Columns[from.ColumnIdx].WorkItems
	toItems := wm.workspace.Columns[to.ColumnIdx].WorkItems

	item := fromItems[from.ItemIdx]
	fromItems = slices.Delete(fromItems, from.ItemIdx, from.ItemIdx+1)
	toItems = slices.Insert(toItems, to.ItemIdx, item)

	wm.workspace.Columns[from.ColumnIdx].WorkItems = fromItems
	wm.workspace.Columns[to.ColumnIdx].WorkItems = toItems

	return nil
}

// NewWorkItem creates a work item with a generated short ID.
func NewWorkItem(name string) WorkItem {
	return WorkItem{
		ID:   rand.Text()[:8],
		Name: name,
	}
}

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
		items := wm.workspace.Columns[from.ColumnIdx].WorkItems
		if from.ItemIdx == len(items)-1 {
			return from, nil
		}
		to = WorkItemPosition{
			ColumnIdx: from.ColumnIdx,
			ItemIdx:   from.ItemIdx + 2,
		}
	case DirectionRight:
		columns := wm.workspace.Columns
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

		columns := wm.workspace.Columns
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
		return "", fmt.Errorf("unknown move direction %q", s)
	}
}
