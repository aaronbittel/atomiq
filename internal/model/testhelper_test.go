package model_test

import "github.com/aaronbittel/atomiq/internal/model"

func workspace(columns ...model.Column) model.Workspace {
	return model.Workspace{
		Columns: columns,
	}
}

func column(name string, items ...model.WorkItem) model.Column {
	var result []model.WorkItem

	for _, item := range items {
		result = append(result, item)
	}

	return model.Column{
		Name:      name,
		WorkItems: result,
	}
}

func item(id, name string) model.WorkItem {
	return model.WorkItem{
		ID:   id,
		Name: name,
	}
}

func position(columnIdx, itemIdx int) model.WorkItemPosition {
	return model.WorkItemPosition{
		ColumnIdx: columnIdx,
		ItemIdx:   itemIdx,
	}
}

func workspaceView(columns ...model.ColumnView) model.WorkspaceView {
	return model.WorkspaceView{
		Columns: columns,
	}
}

func columnView(name string, items ...model.WorkItemView) model.ColumnView {
	var result []model.WorkItemView

	for _, item := range items {
		result = append(result, item)
	}

	return model.ColumnView{
		Name:      name,
		WorkItems: result,
	}
}

func itemView(item model.WorkItem) model.WorkItemView {
	return model.WorkItemView{
		ID:   item.ID,
		Name: item.Name,
	}
}
