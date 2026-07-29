package model_test

import "github.com/aaronbittel/atomiq/internal/model"

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
