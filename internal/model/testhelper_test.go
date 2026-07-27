package model

func workspace(columns ...Column) Workspace {
	return Workspace{
		Columns: columns,
	}
}

func column(name string, items ...WorkItem) Column {
	result := []WorkItem{}

	for _, item := range items {
		result = append(result, item)
	}

	return Column{
		Name:      name,
		WorkItems: result,
	}
}

func item(id, name string) WorkItem {
	return WorkItem{
		ID:   id,
		Name: name,
	}
}

func position(columnIdx, itemIdx int) workItemPosition {
	return workItemPosition{
		ColumnIdx: columnIdx,
		ItemIdx:   itemIdx,
	}
}

func insertPoint(columnIdx, itemIdx int) WorkItemInsertionPoint {
	return WorkItemInsertionPoint{
		ColumnIdx: columnIdx,
		ItemIdx:   itemIdx,
	}
}

func workspaceView(columns ...ColumnView) WorkspaceView {
	return WorkspaceView{
		Columns: columns,
	}
}

func columnView(name string, items ...WorkItemView) ColumnView {
	var result []WorkItemView

	for _, item := range items {
		result = append(result, item)
	}

	return ColumnView{
		Name:      name,
		WorkItems: result,
	}
}

func itemView(item WorkItem) WorkItemView {
	return WorkItemView{
		ID:   item.ID,
		Name: item.Name,
	}
}
