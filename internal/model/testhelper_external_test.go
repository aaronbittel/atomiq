package model_test

import "github.com/aaronbittel/atomiq/internal/model"

func workspace(columns ...model.Column) model.Workspace {
	return model.Workspace{
		Columns: columns,
	}
}

func column(name string, items ...model.WorkItem) model.Column {
	result := []model.WorkItem{}

	result = append(result, items...)

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

func workspaceView(revision uint64, columns ...model.ColumnView) model.WorkspaceView {
	return model.WorkspaceView{
		Revision: revision,
		Columns:  columns,
	}
}

func columnView(name string, items ...model.WorkItemView) model.ColumnView {
	var result []model.WorkItemView

	result = append(result, items...)

	return model.ColumnView{
		Name:      name,
		WorkItems: result,
	}
}

func itemView(item model.WorkItem) model.WorkItemView {
	return model.WorkItemView(item)
}
