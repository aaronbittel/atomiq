package model

// WorkspaceView is a detached snapshot of a Workspace.
type WorkspaceView struct {
	// Columns is the ordered list of columns in the workspace.
	Columns []ColumnView
	// Revision is the version required for the next mutation request.
	Revision uint64
	// ID is the id of the active workspace.
	ID WorkspaceID
}

// ColumnView is the rendered representation of a workspace column.
type ColumnView struct {
	// Name is the column title.
	Name string
	// WorkItems is the ordered list of items in the column.
	WorkItems []WorkItemView
}

// WorkItemView is the rendered representation of a work item.
type WorkItemView struct {
	// ID identifies the work item for future mutation requests.
	ID WorkItemID
	// Name is the work item title.
	Name string
}
