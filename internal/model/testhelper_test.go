package model

func workspace(columns ...Column) Workspace {
	return Workspace{
		Columns: columns,
	}
}

func column(name string, items ...WorkItem) Column {
	result := []WorkItem{}

	result = append(result, items...)

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
