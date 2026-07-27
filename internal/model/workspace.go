package model

import (
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

func (ws *Workspace) add(columnIdx int, name string) error {
	if !validSliceAccess(columnIdx, len(ws.Columns)) {
		return columnIdxErr(columnIdx, len(ws.Columns))
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalidWorkItemName
	}

	ws.Columns[columnIdx].WorkItems = append(ws.Columns[columnIdx].WorkItems, NewWorkItem(name))

	return nil
}

func (ws *Workspace) delete(id string) error {
	pos, err := ws.findWorkItemPosition(id)
	if err != nil {
		return err
	}

	items := ws.Columns[pos.ColumnIdx].WorkItems
	items = slices.Delete(items, pos.ItemIdx, pos.ItemIdx+1)
	ws.Columns[pos.ColumnIdx].WorkItems = items

	return nil
}

func (ws *Workspace) moveInDirection(id string, from WorkItemPosition, direction MoveDirection) error {
	if err := ws.isValidFromPosition(from); err != nil {
		return err
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
		return err
	}

	if from == to {
		return nil
	}

	ws.moveWorkItemToPosition(from, to)
	return nil
}

func (ws *Workspace) moveToPosition(id string, from, to WorkItemPosition) error {
	if err := ws.isValidFromPosition(from); err != nil {
		return err
	}

	item := ws.Columns[from.ColumnIdx].WorkItems[from.ItemIdx]
	if item.ID != id {
		return ErrItemIDMismatch
	}

	if err := ws.isValidToPosition(to); err != nil {
		return err
	}

	if from == to {
		return nil
	}

	ws.moveWorkItemToPosition(from, to)
	return nil
}

func (ws *Workspace) findWorkItemPosition(id string) (WorkItemPosition, error) {
	for colIdx, col := range ws.Columns {
		for itemIdx, item := range col.WorkItems {
			if item.ID == id {
				return WorkItemPosition{
					ColumnIdx: colIdx,
					ItemIdx:   itemIdx,
				}, nil
			}
		}
	}

	return WorkItemPosition{}, fmt.Errorf("%w: %q", ErrWorkItemNotFound, id)
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

func (ws *Workspace) moveWorkItemToPosition(from, to WorkItemPosition) {
	if from.ColumnIdx == to.ColumnIdx {
		ws.moveWorkItemWithinColumn(from.ColumnIdx, from.ItemIdx, to.ItemIdx)
		return
	}

	ws.moveWorkItemBetweenColumns(from, to)
}

func (ws *Workspace) moveWorkItemWithinColumn(columnIdx, fromIdx, toIdx int) {
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
}

func (ws *Workspace) moveWorkItemBetweenColumns(from, to WorkItemPosition) {
	fromItems := ws.Columns[from.ColumnIdx].WorkItems
	toItems := ws.Columns[to.ColumnIdx].WorkItems

	item := fromItems[from.ItemIdx]
	fromItems = slices.Delete(fromItems, from.ItemIdx, from.ItemIdx+1)
	toItems = slices.Insert(toItems, to.ItemIdx, item)

	ws.Columns[from.ColumnIdx].WorkItems = fromItems
	ws.Columns[to.ColumnIdx].WorkItems = toItems
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
		return columnIdxErr(pos.ColumnIdx, len(ws.Columns))
	}

	if !validSliceAccess(pos.ItemIdx, len(ws.Columns[pos.ColumnIdx].WorkItems)) {
		return itemIdxErr(pos.ColumnIdx, pos.ItemIdx, len(ws.Columns[pos.ColumnIdx].WorkItems))
	}

	return nil
}

func (ws *Workspace) isValidToPosition(pos WorkItemPosition) error {
	if !validSliceAccess(pos.ColumnIdx, len(ws.Columns)) {
		return columnIdxErr(pos.ColumnIdx, len(ws.Columns))
	}

	if !validInsertionIndex(pos.ItemIdx, len(ws.Columns[pos.ColumnIdx].WorkItems)) {
		return itemInsertionIdxErr(pos.ColumnIdx, pos.ItemIdx, len(ws.Columns[pos.ColumnIdx].WorkItems))
	}

	return nil
}

func validSliceAccess(idx, length int) bool {
	return idx >= 0 && idx < length
}

func validInsertionIndex(idx, length int) bool {
	return idx >= 0 && idx <= length
}
