package model

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// Workspace contains the columns and work items in one work context.
type Workspace struct {
	Columns []Column
}

// Column groups work items under one name.
type Column struct {
	Name      string
	WorkItems []WorkItem
}

// WorkItem is one unit of work in a column.
type WorkItem struct {
	ID   string
	Name string
}

var (
	ErrInvalidColumn        = errors.New("invalid column")
	ErrInvalidPosition      = errors.New("invalid work item position")
	ErrInvalidWorkItemName  = errors.New("invalid work item name")
	ErrItemIDMismatch       = errors.New("item ID mismatch")
	ErrInvalidMoveDirection = errors.New("invalid move direction")
)

func (ws *Workspace) add(columnIdx int, name string) error {
	if !validSliceAccess(columnIdx, len(ws.Columns)) {
		return ErrInvalidColumn
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalidWorkItemName
	}

	ws.Columns[columnIdx].WorkItems = append(ws.Columns[columnIdx].WorkItems, NewWorkItem(name))

	return nil
}

func (ws *Workspace) delete(id string, pos WorkItemPosition) error {
	if !validSliceAccess(pos.ColumnIdx, len(ws.Columns)) {
		return fmt.Errorf("%w: column index out of bounds", ErrInvalidPosition)
	}
	items := ws.Columns[pos.ColumnIdx].WorkItems
	if !validSliceAccess(pos.ItemIdx, len(items)) {
		return fmt.Errorf("%w: item index out of bounds", ErrInvalidPosition)
	}

	item := items[pos.ItemIdx]
	if item.ID != id {
		return ErrItemIDMismatch
	}

	items = slices.Delete(items, pos.ItemIdx, pos.ItemIdx+1)
	ws.Columns[pos.ColumnIdx].WorkItems = items

	return nil
}

func (ws *Workspace) moveInDirection(id string, from WorkItemPosition, direction MoveDirection) error {
	if err := ws.isValidFromPosition(from); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidPosition, err)
	}

	item := ws.Columns[from.ColumnIdx].WorkItems[from.ItemIdx]
	if id != item.ID {
		return fmt.Errorf("%w: expected %q, got %q", ErrItemIDMismatch, id, item.ID)
	}

	to, err := ws.getToWorkItemPosition(from, direction)
	if err != nil {
		return err
	}

	if err := ws.isValidToPosition(to); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidPosition, err)
	}

	if from == to {
		return nil
	}

	return ws.moveWorkItemToPosition(from, to)
}

func (ws *Workspace) moveToPosition(from, to WorkItemPosition) error {
	if err := ws.isValidFromPosition(from); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidPosition, err)
	}

	if err := ws.isValidToPosition(to); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidPosition, err)
	}

	if from == to {
		return nil
	}

	return ws.moveWorkItemToPosition(from, to)
}

func (ws *Workspace) clone() Workspace {
	columns := make([]Column, len(ws.Columns))
	for i, col := range ws.Columns {
		columns[i].Name = col.Name
		for _, item := range col.WorkItems {
			columns[i].WorkItems = append(columns[i].WorkItems, WorkItem{
				ID:   item.ID,
				Name: item.Name,
			})
		}
	}
	return Workspace{Columns: columns}
}

func (ws *Workspace) view() WorkspaceView {
	columnViews := make([]ColumnView, len(ws.Columns))
	for i, col := range ws.Columns {
		columnViews[i].Name = col.Name
		for _, wi := range col.WorkItems {
			columnViews[i].WorkItems = append(columnViews[i].WorkItems, WorkItemView{
				ID:   wi.ID,
				Name: wi.Name,
			})
		}
	}
	return WorkspaceView{Columns: columnViews}
}

func (ws *Workspace) moveWorkItemToPosition(from, to WorkItemPosition) error {
	if from.ColumnIdx == to.ColumnIdx {
		return ws.moveWorkItemWithinColumn(from.ColumnIdx, from.ItemIdx, to.ItemIdx)
	}

	return ws.moveWorkItemBetweenColumns(from, to)
}

func (ws *Workspace) moveWorkItemWithinColumn(columnIdx, fromIdx, toIdx int) error {
	items := ws.Columns[columnIdx].WorkItems
	item := items[fromIdx]

	items = slices.Delete(items, fromIdx, fromIdx+1)
	if fromIdx < toIdx {
		// toIdx refers to the original column; deleting an earlier item shifts
		// the insertion position one slot left.
		toIdx -= 1
	}
	items = slices.Insert(items, toIdx, item)

	ws.Columns[columnIdx].WorkItems = items

	return nil
}

func (ws *Workspace) moveWorkItemBetweenColumns(from, to WorkItemPosition) error {
	fromItems := ws.Columns[from.ColumnIdx].WorkItems
	toItems := ws.Columns[to.ColumnIdx].WorkItems

	item := fromItems[from.ItemIdx]
	fromItems = slices.Delete(fromItems, from.ItemIdx, from.ItemIdx+1)
	toItems = slices.Insert(toItems, to.ItemIdx, item)

	ws.Columns[from.ColumnIdx].WorkItems = fromItems
	ws.Columns[to.ColumnIdx].WorkItems = toItems

	return nil
}

func (ws *Workspace) getToWorkItemPosition(from WorkItemPosition, direction MoveDirection) (WorkItemPosition, error) {
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
		items := ws.Columns[from.ColumnIdx].WorkItems
		if from.ItemIdx == len(items)-1 {
			return from, nil
		}
		to = WorkItemPosition{
			ColumnIdx: from.ColumnIdx,
			ItemIdx:   from.ItemIdx + 2,
		}
	case DirectionRight:
		columns := ws.Columns
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

		columns := ws.Columns
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

func (ws *Workspace) isValidFromPosition(pos WorkItemPosition) error {
	if !validSliceAccess(pos.ColumnIdx, len(ws.Columns)) {
		return errors.New("from column index out of bounds")
	}

	if !validSliceAccess(pos.ItemIdx, len(ws.Columns[pos.ColumnIdx].WorkItems)) {
		return errors.New("from item index out of bounds")
	}

	return nil
}

func (ws *Workspace) isValidToPosition(pos WorkItemPosition) error {
	if !validSliceAccess(pos.ColumnIdx, len(ws.Columns)) {
		return errors.New("to column index out of bounds")
	}

	if !validSliceAccess(pos.ItemIdx, len(ws.Columns[pos.ColumnIdx].WorkItems)+1) {
		return errors.New("to item index out of bounds")
	}

	return nil
}

func validSliceAccess(idx, length int) bool {
	return idx >= 0 && idx < length
}
