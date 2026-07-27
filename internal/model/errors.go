package model

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidPosition      = errors.New("invalid position")
	ErrInvalidWorkItemName  = errors.New("invalid work item name")
	ErrInvalidMoveDirection = errors.New("invalid move direction")
	ErrWorkItemNotFound     = errors.New("work item not found")
)

func columnIdxErr(idx, length int) error {
	return fmt.Errorf(
		"%w: column index %d out of bounds for %d columns",
		ErrInvalidPosition,
		idx,
		length,
	)
}

func itemIdxErr(columnIdx, itemIdx, length int) error {
	return fmt.Errorf(
		"%w: item index %d out of bounds for column %d with %d items",
		ErrInvalidPosition,
		itemIdx,
		columnIdx,
		length,
	)
}

func itemInsertionIdxErr(columnIdx, insertIdx, length int) error {
	return fmt.Errorf(
		"%w: item insertion index %d out of bounds for column %d with %d items",
		ErrInvalidPosition,
		insertIdx,
		columnIdx,
		length,
	)
}
