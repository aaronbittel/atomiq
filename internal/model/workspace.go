package model

import (
	"crypto/rand"
	"fmt"
	"slices"
	"strings"
)

type WorkItemID string
type WorkspaceID string

// Workspace contains the columns and work items in one work context.
type Workspace struct {
	id      WorkspaceID
	columns []Column
}

func NewWorkspace(columns ...Column) Workspace {
	clonedColumns := make([]Column, len(columns))
	for i, col := range columns {
		clonedColumns[i] = col.clone()
	}
	return Workspace{
		id:      newWorkspaceID(),
		columns: clonedColumns,
	}
}

func (ws *Workspace) clone() Workspace {
	columns := make([]Column, len(ws.columns))
	for i, col := range ws.columns {
		columns[i] = col.clone()
	}
	return Workspace{
		id:      ws.id,
		columns: columns,
	}
}

// Column groups work items under one name.
type Column struct {
	name      string
	workItems []WorkItem
}

func NewColumn(name string, items ...WorkItem) Column {
	clonedItems := make([]WorkItem, len(items))
	for i, item := range items {
		clonedItems[i] = item.clone()
	}
	return Column{
		name:      name,
		workItems: clonedItems,
	}
}

func (c Column) clone() Column {
	items := make([]WorkItem, len(c.workItems))
	for i, item := range c.workItems {
		items[i] = item.clone()
	}
	return Column{
		name:      c.name,
		workItems: items,
	}
}

// WorkItem is one unit of work in a column.
type WorkItem struct {
	id   WorkItemID
	name string
}

func (wi WorkItem) ID() WorkItemID {
	return wi.id
}

func NewWorkItem(name string) WorkItem {
	return WorkItem{
		id:   newWorkItemID(),
		name: name,
	}
}

func (wi WorkItem) clone() WorkItem {
	return WorkItem{
		id:   wi.id,
		name: wi.name,
	}
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

func (ws *Workspace) add(columnIdx int, name string) (updated bool, err error) {
	if !validSliceAccess(columnIdx, len(ws.columns)) {
		return false, columnIdxErr(columnIdx, len(ws.columns))
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return false, ErrInvalidWorkItemName
	}

	ws.columns[columnIdx].workItems = append(ws.columns[columnIdx].workItems, NewWorkItem(name))

	return true, nil
}

func (ws *Workspace) delete(id WorkItemID) (updated bool, err error) {
	pos, err := ws.findWorkItemPosition(id)
	if err != nil {
		return false, err
	}

	items := ws.columns[pos.ColumnIdx].workItems
	items = slices.Delete(items, pos.ItemIdx, pos.ItemIdx+1)
	ws.columns[pos.ColumnIdx].workItems = items

	return true, nil
}

func (ws *Workspace) moveInDirection(id WorkItemID, direction MoveDirection) (updated bool, err error) {
	pos, err := ws.findWorkItemPosition(id)
	if err != nil {
		return false, err
	}

	to, err := ws.getToWorkItemPosition(pos, direction)
	if err != nil {
		return false, err
	}

	if pos == to {
		return false, nil
	}

	ws.moveWorkItemToPosition(pos, to)
	return true, nil
}

func (ws *Workspace) moveToPosition(id WorkItemID, insertPoint WorkItemInsertionPoint) (updated bool, err error) {
	from, err := ws.findWorkItemPosition(id)
	if err != nil {
		return false, err
	}

	if err := ws.isValidToPosition(insertPoint); err != nil {
		return false, err
	}

	to := workItemPosition(insertPoint)

	if sameEffectivePosition(from, to) {
		return false, nil
	}

	ws.moveWorkItemToPosition(from, to)
	return true, nil
}

func sameEffectivePosition(from, to workItemPosition) bool {
	if from.ColumnIdx != to.ColumnIdx {
		return false
	}

	return to.ItemIdx == from.ItemIdx || to.ItemIdx == from.ItemIdx+1
}

func (ws *Workspace) findWorkItemPosition(id WorkItemID) (workItemPosition, error) {
	for colIdx, col := range ws.columns {
		for itemIdx, item := range col.workItems {
			if item.id == id {
				return workItemPosition{
					ColumnIdx: colIdx,
					ItemIdx:   itemIdx,
				}, nil
			}
		}
	}

	return workItemPosition{}, fmt.Errorf("%w: %q", ErrWorkItemNotFound, id)
}

func (ws *Workspace) view() []ColumnView {
	columns := make([]ColumnView, len(ws.columns))
	for i, col := range ws.columns {
		columns[i].Name = col.name
		for _, wi := range col.workItems {
			columns[i].WorkItems = append(columns[i].WorkItems, WorkItemView{
				ID:   wi.id,
				Name: wi.name,
			})
		}
	}
	return columns
}

func (ws *Workspace) moveWorkItemToPosition(from, to workItemPosition) {
	if from.ColumnIdx == to.ColumnIdx {
		ws.moveWorkItemWithinColumn(from.ColumnIdx, from.ItemIdx, to.ItemIdx)
		return
	}

	ws.moveWorkItemBetweenColumns(from, to)
}

func (ws *Workspace) moveWorkItemWithinColumn(columnIdx, fromIdx, toIdx int) {
	items := ws.columns[columnIdx].workItems
	item := items[fromIdx]

	items = slices.Delete(items, fromIdx, fromIdx+1)
	if fromIdx < toIdx {
		// toIdx refers to the original column; deleting an earlier item shifts
		// the insertion position one slot left.
		toIdx -= 1
	}
	items = slices.Insert(items, toIdx, item)

	ws.columns[columnIdx].workItems = items
}

func (ws *Workspace) moveWorkItemBetweenColumns(from, to workItemPosition) {
	fromItems := ws.columns[from.ColumnIdx].workItems
	toItems := ws.columns[to.ColumnIdx].workItems

	item := fromItems[from.ItemIdx]
	fromItems = slices.Delete(fromItems, from.ItemIdx, from.ItemIdx+1)
	toItems = slices.Insert(toItems, to.ItemIdx, item)

	ws.columns[from.ColumnIdx].workItems = fromItems
	ws.columns[to.ColumnIdx].workItems = toItems
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
		items := ws.columns[from.ColumnIdx].workItems
		if from.ItemIdx == len(items)-1 {
			return from, nil
		}
		to = workItemPosition{
			ColumnIdx: from.ColumnIdx,
			ItemIdx:   from.ItemIdx + 2,
		}
	case DirectionRight:
		columns := ws.columns
		if from.ColumnIdx == len(columns)-1 {
			return from, nil
		}

		toColumnIdx := from.ColumnIdx + 1

		to = workItemPosition{
			ColumnIdx: toColumnIdx,
			ItemIdx:   len(columns[toColumnIdx].workItems),
		}
	case DirectionLeft:
		if from.ColumnIdx == 0 {
			return from, nil
		}

		columns := ws.columns
		toColumnIdx := from.ColumnIdx - 1

		to = workItemPosition{
			ColumnIdx: toColumnIdx,
			ItemIdx:   len(columns[toColumnIdx].workItems),
		}
	default:
		return workItemPosition{}, fmt.Errorf("%w: %q", ErrInvalidMoveDirection, direction)
	}

	return to, nil
}

func (ws *Workspace) isValidToPosition(pos WorkItemInsertionPoint) error {
	if !validSliceAccess(pos.ColumnIdx, len(ws.columns)) {
		return columnIdxErr(pos.ColumnIdx, len(ws.columns))
	}

	if !validInsertionIndex(pos.ItemIdx, len(ws.columns[pos.ColumnIdx].workItems)) {
		return itemInsertionIdxErr(pos.ColumnIdx, pos.ItemIdx, len(ws.columns[pos.ColumnIdx].workItems))
	}

	return nil
}

func newWorkItemID() WorkItemID {
	return WorkItemID(rand.Text()[:WorkItemIDLength])
}

func ParseWorkItemID(s string) (WorkItemID, error) {
	if len(s) != WorkItemIDLength {
		return "", ErrInvalidWorkItemIDFormat
	}
	return WorkItemID(s), nil
}

func (wid WorkItemID) String() string {
	return string(wid)
}

func newWorkspaceID() WorkspaceID {
	return WorkspaceID(rand.Text()[:WorkspaceIDLength])
}

func ParseWorkspaceID(s string) (WorkspaceID, error) {
	if len(s) != WorkspaceIDLength {
		return "", ErrInvalidWorkspaceIDFormat
	}
	return WorkspaceID(s), nil
}

func (wid WorkspaceID) String() string {
	return string(wid)
}

func validSliceAccess(idx, length int) bool {
	return idx >= 0 && idx < length
}

func validInsertionIndex(idx, length int) bool {
	return idx >= 0 && idx <= length
}
