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

// workItemPosition identifies the current location of an existing work item.
type workItemPosition struct {
	ColumnIdx int
	ItemIdx   int
}

// WorkItemInsertionPoint identifies a position at which a work item can be inserted.
type WorkItemInsertionPoint struct {
	ColumnIdx int
	// ItemIdx is an insertion index within the destination column.
	//
	// For a column containing [A, B, C], index 0 inserts before A, index 1
	// inserts between A and B, and index 3 appends after C.
	ItemIdx int
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

func (ws *Workspace) moveInDirection(id string, direction MoveDirection) error {
	pos, err := ws.findWorkItemPosition(id)
	if err != nil {
		return err
	}

	to, err := ws.getToWorkItemPosition(pos, direction)
	if err != nil {
		return err
	}

	if pos == to {
		return nil
	}

	ws.moveWorkItemToPosition(pos, to)
	return nil
}

func (ws *Workspace) moveToPosition(id string, insertPoint WorkItemInsertionPoint) error {
	from, err := ws.findWorkItemPosition(id)
	if err != nil {
		return err
	}

	if err := ws.isValidToPosition(insertPoint); err != nil {
		return err
	}

	to := workItemPosition{
		ColumnIdx: insertPoint.ColumnIdx,
		ItemIdx:   insertPoint.ItemIdx,
	}

	if sameEffectivePosition(from, to) {
		return nil
	}

	ws.moveWorkItemToPosition(from, to)
	return nil
}

func sameEffectivePosition(from, to workItemPosition) bool {
	if from.ColumnIdx != to.ColumnIdx {
		return false
	}

	return to.ItemIdx == from.ItemIdx || to.ItemIdx == from.ItemIdx+1
}

func (ws *Workspace) findWorkItemPosition(id string) (workItemPosition, error) {
	for colIdx, col := range ws.Columns {
		for itemIdx, item := range col.WorkItems {
			if item.ID == id {
				return workItemPosition{
					ColumnIdx: colIdx,
					ItemIdx:   itemIdx,
				}, nil
			}
		}
	}

	return workItemPosition{}, fmt.Errorf("%w: %q", ErrWorkItemNotFound, id)
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

func (ws *Workspace) moveWorkItemToPosition(from, to workItemPosition) {
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

func (ws *Workspace) moveWorkItemBetweenColumns(from, to workItemPosition) {
	fromItems := ws.Columns[from.ColumnIdx].WorkItems
	toItems := ws.Columns[to.ColumnIdx].WorkItems

	item := fromItems[from.ItemIdx]
	fromItems = slices.Delete(fromItems, from.ItemIdx, from.ItemIdx+1)
	toItems = slices.Insert(toItems, to.ItemIdx, item)

	ws.Columns[from.ColumnIdx].WorkItems = fromItems
	ws.Columns[to.ColumnIdx].WorkItems = toItems
}

func (ws *Workspace) getToWorkItemPosition(from workItemPosition, direction MoveDirection) (workItemPosition, error) {
	var to workItemPosition
	switch direction {
	case DirectionUp:
		if from.ItemIdx == 0 {
			return from, nil
		}
		to = workItemPosition{
			ColumnIdx: from.ColumnIdx,
			ItemIdx:   from.ItemIdx - 1,
		}
	case DirectionDown:
		items := ws.Columns[from.ColumnIdx].WorkItems
		if from.ItemIdx == len(items)-1 {
			return from, nil
		}
		to = workItemPosition{
			ColumnIdx: from.ColumnIdx,
			ItemIdx:   from.ItemIdx + 2,
		}
	case DirectionRight:
		columns := ws.Columns
		if from.ColumnIdx == len(columns)-1 {
			return from, nil
		}

		toColumnIdx := from.ColumnIdx + 1

		to = workItemPosition{
			ColumnIdx: toColumnIdx,
			ItemIdx:   len(columns[toColumnIdx].WorkItems),
		}
	case DirectionLeft:
		if from.ColumnIdx == 0 {
			return from, nil
		}

		columns := ws.Columns
		toColumnIdx := from.ColumnIdx - 1

		to = workItemPosition{
			ColumnIdx: toColumnIdx,
			ItemIdx:   len(columns[toColumnIdx].WorkItems),
		}
	default:
		return workItemPosition{}, fmt.Errorf("%w: %q", ErrInvalidMoveDirection, direction)
	}

	return to, nil
}

func (ws *Workspace) isValidFromPosition(pos workItemPosition) error {
	if !validSliceAccess(pos.ColumnIdx, len(ws.Columns)) {
		return columnIdxErr(pos.ColumnIdx, len(ws.Columns))
	}

	if !validSliceAccess(pos.ItemIdx, len(ws.Columns[pos.ColumnIdx].WorkItems)) {
		return itemIdxErr(pos.ColumnIdx, pos.ItemIdx, len(ws.Columns[pos.ColumnIdx].WorkItems))
	}

	return nil
}

func (ws *Workspace) isValidToPosition(pos WorkItemInsertionPoint) error {
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
