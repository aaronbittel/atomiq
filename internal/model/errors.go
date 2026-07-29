package model

import (
	"errors"
	"fmt"
)

var (
	// ErrInvalidPosition indicates that a column or item index is out of bounds.
	ErrInvalidPosition = errors.New("invalid position")
	// ErrInvalidWorkItemName indicates that a work item name is blank after trimming.
	ErrInvalidWorkItemName = errors.New("invalid work item name")
	// ErrInvalidMoveDirection indicates that a move direction is not recognized.
	ErrInvalidMoveDirection = errors.New("invalid move direction")
	// ErrWorkItemNotFound indicates that no item exists for the supplied ID.
	ErrWorkItemNotFound = errors.New("work item not found")
	// ErrRevisionConflict indicates that a mutation used a stale workspace revision.
	ErrRevisionConflict = errors.New("workspace revision conflict")
	// ErrInvalidWorkItemIDFormat indicates that a work item ID has an invalid shape.
	ErrInvalidWorkItemIDFormat = errors.New("invalid work item id format")
	// ErrInvalidWorkspaceIDFormat indicates that a workspace ID has an invalid shape.
	ErrInvalidWorkspaceIDFormat = errors.New("invalid workspace id format")
)

// columnIdxErr returns an ErrInvalidPosition for a column index access.
func columnIdxErr(idx, length int) error {
	return fmt.Errorf(
		"%w: column index %d out of bounds for %d columns",
		ErrInvalidPosition,
		idx,
		length,
	)
}

// itemInsertionIdxErr returns an ErrInvalidPosition for an item insertion index.
func itemInsertionIdxErr(columnIdx, insertIdx, length int) error {
	return fmt.Errorf(
		"%w: item insertion index %d out of bounds for column %d with %d items",
		ErrInvalidPosition,
		insertIdx,
		columnIdx,
		length,
	)
}
