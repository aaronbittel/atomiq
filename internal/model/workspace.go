package model

import (
	"crypto/rand"
	"fmt"
	"slices"
	"strings"
)

// WorkspaceID identifies a workspace.
type WorkspaceID string

// WorkspaceIDLength is the number of characters generated for workspace IDs.
const WorkspaceIDLength = 8

// newWorkspaceID returns a random workspace ID.
func newWorkspaceID() WorkspaceID {
	return WorkspaceID(rand.Text()[:WorkspaceIDLength])
}

// ParseWorkspaceID validates s and returns it as a WorkspaceID.
func ParseWorkspaceID(s string) (WorkspaceID, error) {
	if len(s) != WorkspaceIDLength {
		return "", ErrInvalidWorkspaceIDFormat
	}
	return WorkspaceID(s), nil
}

// String returns the raw workspace ID value.
func (wid WorkspaceID) String() string {
	return string(wid)
}

// MoveDirection describes a visual move in the rendered workspace.
//
// Up and down move within a column; left and right move to the end of the
// neighboring column.
type MoveDirection string

const (
	// DirectionUp moves a work item one position earlier in its column.
	DirectionUp MoveDirection = "up"
	// DirectionDown moves a work item one position later in its column.
	DirectionDown MoveDirection = "down"
	// DirectionRight moves a work item to the end of the next column.
	DirectionRight MoveDirection = "right"
	// DirectionLeft moves a work item to the end of the previous column.
	DirectionLeft MoveDirection = "left"
)

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
		return "", fmt.Errorf("%w: %q", ErrInvalidMoveDirection, s)
	}
}

// WorkItemInsertionPoint identifies where a work item can be inserted.
type WorkItemInsertionPoint struct {
	// ColumnIdx is the index of the destination column.
	ColumnIdx int
	// ItemIdx is an insertion index within the destination column.
	//
	// For a column containing [A, B, C], index 0 inserts before A, index 1
	// inserts between A and B, and index 3 appends after C.
	ItemIdx int
}

// Workspace contains the columns and work items in one work context.
type Workspace struct {
	id      WorkspaceID
	columns []Column
}

// NewWorkspace creates a workspace with a new ID and detached column copies.
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

// clone returns a deep copy of ws that preserves its ID.
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

// view returns a detached render snapshot of ws.
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

// add trims and appends a new work item to the selected column.
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

// delete removes the work item with id.
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

// moveInDirection moves a work item one visual step.
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

// moveToPosition moves a work item to an insertion point.
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

// workItemPosition identifies the current location of an existing work item.
type workItemPosition struct {
	ColumnIdx int
	ItemIdx   int
}

// findWorkItemPosition returns the current position of the work item with id.
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

// moveWorkItemToPosition moves an item from an existing position to an insertion position.
func (ws *Workspace) moveWorkItemToPosition(from, to workItemPosition) {
	if from.ColumnIdx == to.ColumnIdx {
		ws.moveWorkItemWithinColumn(from.ColumnIdx, from.ItemIdx, to.ItemIdx)
		return
	}

	ws.moveWorkItemBetweenColumns(from, to)
}

// moveWorkItemWithinColumn reorders an item in one column.
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

// moveWorkItemBetweenColumns moves an item from one column into another.
func (ws *Workspace) moveWorkItemBetweenColumns(from, to workItemPosition) {
	fromItems := ws.columns[from.ColumnIdx].workItems
	toItems := ws.columns[to.ColumnIdx].workItems

	item := fromItems[from.ItemIdx]
	fromItems = slices.Delete(fromItems, from.ItemIdx, from.ItemIdx+1)
	toItems = slices.Insert(toItems, to.ItemIdx, item)

	ws.columns[from.ColumnIdx].workItems = fromItems
	ws.columns[to.ColumnIdx].workItems = toItems
}

// getToWorkItemPosition resolves a directional move into an insertion position.
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

// isValidToPosition reports whether pos can be used as an insertion point.
func (ws *Workspace) isValidToPosition(pos WorkItemInsertionPoint) error {
	if !validSliceAccess(pos.ColumnIdx, len(ws.columns)) {
		return columnIdxErr(pos.ColumnIdx, len(ws.columns))
	}

	if !validInsertionIndex(pos.ItemIdx, len(ws.columns[pos.ColumnIdx].workItems)) {
		return itemInsertionIdxErr(pos.ColumnIdx, pos.ItemIdx, len(ws.columns[pos.ColumnIdx].workItems))
	}

	return nil
}

// sameEffectivePosition reports whether a same-column move would leave the item in place.
func sameEffectivePosition(from, to workItemPosition) bool {
	if from.ColumnIdx != to.ColumnIdx {
		return false
	}

	return to.ItemIdx == from.ItemIdx || to.ItemIdx == from.ItemIdx+1
}

// validSliceAccess reports whether idx addresses an existing slice element.
func validSliceAccess(idx, length int) bool {
	return idx >= 0 && idx < length
}

// validInsertionIndex reports whether idx can be used with slices.Insert.
func validInsertionIndex(idx, length int) bool {
	return idx >= 0 && idx <= length
}
