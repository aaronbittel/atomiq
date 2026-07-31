package model_test

import "github.com/aaronbittel/atomiq/internal/model"

func workspaceRootView(
	id model.WorkspaceID,
	revision uint64,
	columns ...model.ColumnView,
) model.WorkspaceView {
	return workspaceView(id, nil, revision, model.RootWorkspaceTitle, columns...)
}

func workspaceChildView(
	id model.WorkspaceID,
	parentID model.WorkspaceID,
	revision uint64,
	title string,
	columns ...model.ColumnView,
) model.WorkspaceView {
	return workspaceView(id, &parentID, revision, title, columns...)
}

func workspaceView(
	id model.WorkspaceID,
	parentID *model.WorkspaceID,
	revision uint64,
	title string,
	columns ...model.ColumnView,
) model.WorkspaceView {
	return model.WorkspaceView{
		ID:       id,
		ParentID: parentID,
		Revision: revision,
		Title:    title,
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
