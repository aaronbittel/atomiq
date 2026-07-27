package model

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidPosition      = errors.New("invalid position")
	ErrInvalidWorkItemName  = errors.New("invalid work item name")
	ErrItemIDMismatch       = errors.New("item ID mismatch")
	ErrInvalidMoveDirection = errors.New("invalid move direction")
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
