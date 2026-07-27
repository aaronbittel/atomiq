package model

// WorkspaceView is a detached snapshot of a Workspace.
type WorkspaceView struct {
	Columns  []ColumnView
	Revision uint64
}

// ColumnView is the rendered representation of a workspace column.
type ColumnView struct {
	Name      string
	WorkItems []WorkItemView
}

// WorkItemView is the rendered representation of a work item.
type WorkItemView struct {
	ID   string
	Name string
}
