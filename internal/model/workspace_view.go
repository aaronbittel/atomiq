package model

type WorkspaceView struct {
	Columns  []ColumnView
	Revision uint64
}

type ColumnView struct {
	Name      string
	WorkItems []WorkItemView
}

type WorkItemView struct {
	ID   string
	Name string
}
