package model

import "crypto/rand"

// WorkItemID identifies a work item.
type WorkItemID string

// WorkItemIDLength is the number of characters generated for work item IDs.
const WorkItemIDLength = 8

// newWorkItemID returns a random work item ID.
func newWorkItemID() WorkItemID {
	return WorkItemID(rand.Text()[:WorkItemIDLength])
}

// ParseWorkItemID validates s and returns it as a WorkItemID.
func ParseWorkItemID(s string) (WorkItemID, error) {
	if len(s) != WorkItemIDLength {
		return "", ErrInvalidWorkItemIDFormat
	}
	return WorkItemID(s), nil
}

// String returns the raw work item ID value.
func (wid WorkItemID) String() string {
	return string(wid)
}

// WorkItem is one unit of work in a column.
type WorkItem struct {
	id   WorkItemID
	name string
}

// NewWorkItem creates a work item with a new ID.
func NewWorkItem(name string) WorkItem {
	return WorkItem{
		id:   newWorkItemID(),
		name: name,
	}
}

// ID returns the work item ID.
func (wi WorkItem) ID() WorkItemID {
	return wi.id
}

// clone returns a copy of wi.
func (wi WorkItem) clone() WorkItem {
	return WorkItem{
		id:   wi.id,
		name: wi.name,
	}
}
