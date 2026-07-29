package model

func workspaceWithID(id WorkspaceID, columns ...Column) Workspace {
	return Workspace{
		id:      id,
		columns: columns,
	}
}

func column(name string, items ...WorkItem) Column {
	result := []WorkItem{}

	result = append(result, items...)

	return Column{
		name:      name,
		workItems: result,
	}
}

func item(id WorkItemID, name string) WorkItem {
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
