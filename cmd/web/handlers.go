package main

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/aaronbittel/atomiq/internal/model"
)

type moveDirectionView struct {
	Value  string
	Symbol string
}

var moveDirectionViews = []moveDirectionView{
	{Value: "up", Symbol: "↑"},
	{Value: "down", Symbol: "↓"},
	{Value: "right", Symbol: "→"},
	{Value: "left", Symbol: "←"},
}

type workspaceRenderView struct {
	Ws        model.Workspace
	ColumnErr *ColumnErr

	MoveDirections []moveDirectionView
}

type ColumnErr struct {
	Idx int
	Msg string
}

func (app *application) workspaceView(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("./ui/html/workspaceView.tmpl")
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	t.Option("missingkey=error")

	data := workspaceRenderView{
		Ws:             app.workspaceModel.WorkspaceView(),
		MoveDirections: moveDirectionViews,
	}

	if columnErr, ok := app.sessionManager.Pop(r.Context(), "columnErr").(*ColumnErr); ok {
		data.ColumnErr = columnErr
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		app.serverError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = buf.WriteTo(w)
}

func (app *application) workItemPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		app.clientError(w, http.StatusUnprocessableEntity)
		return
	}

	columnIdx, err := parseInt(r.PostForm.Get("columnIdx"))
	if err != nil {
		app.clientError(w, http.StatusUnprocessableEntity)
		return
	}

	workItemName := strings.TrimSpace(r.PostForm.Get("name"))
	if workItemName == "" {
		app.sessionManager.Put(r.Context(), "columnErr", &ColumnErr{Idx: columnIdx, Msg: "work item must not be blank"})
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if err := app.workspaceModel.WorkItemAdd(columnIdx, workItemName); err != nil {
		switch {
		case errors.Is(err, model.ErrInvalidColumn):
			app.clientError(w, http.StatusUnprocessableEntity)
		default:
			app.serverError(w, r, err)
		}
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *application) workItemDelete(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		app.clientError(w, http.StatusUnprocessableEntity)
		return
	}

	itemID := r.PathValue("id")
	if len(itemID) != model.WorkItemIDLength {
		app.clientError(w, http.StatusUnprocessableEntity)
		return
	}

	columnIdx, err := parseInt(r.PostForm.Get("columnIdx"))
	if err != nil {
		app.clientError(w, http.StatusUnprocessableEntity)
		return
	}

	itemIdx, err := parseInt(r.PostForm.Get("itemIdx"))
	if err != nil {
		app.clientError(w, http.StatusUnprocessableEntity)
		return
	}

	pos := model.WorkItemPosition{ColumnIdx: columnIdx, ItemIdx: itemIdx}

	if err := app.workspaceModel.WorkItemDelete(itemID, pos); err != nil {
		switch {
		case errors.Is(err, model.ErrInvalidPosition):
			app.clientError(w, http.StatusUnprocessableEntity)
		case errors.Is(err, model.ErrItemIDMismatch):
			app.clientError(w, http.StatusConflict)
		default:
			app.serverError(w, r, err)
		}
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *application) parseMoveFormData(r *http.Request) (model.WorkItemPosition, model.MoveDirection, error) {
	if err := r.ParseForm(); err != nil {
		return model.WorkItemPosition{}, "", err
	}

	columnIdx, err := parseInt(r.PostForm.Get("fromColumnIdx"))
	if err != nil {
		return model.WorkItemPosition{}, "", err
	}

	itemIdx, err := parseInt(r.PostForm.Get("fromItemIdx"))
	if err != nil {
		return model.WorkItemPosition{}, "", err
	}

	moveDir, err := model.ParseMoveDirection(r.PostForm.Get("direction"))
	if err != nil {
		return model.WorkItemPosition{}, "", err
	}

	from := model.WorkItemPosition{ColumnIdx: columnIdx, ItemIdx: itemIdx}
	return from, moveDir, nil
}

func (app *application) workItemMove(w http.ResponseWriter, r *http.Request) {
	srcPos, direction, err := app.parseMoveFormData(r)
	if err != nil {
		app.clientError(w, http.StatusUnprocessableEntity)
		return
	}

	workItemID := r.PathValue("id")
	if len(workItemID) != model.WorkItemIDLength {
		app.clientError(w, http.StatusUnprocessableEntity)
		return
	}

	if err := app.workspaceModel.WorkItemMoveDirection(workItemID, srcPos, direction); err != nil {
		switch {
		case errors.Is(err, model.ErrInvalidPosition) || errors.Is(err, model.ErrInvalidMoveDirection):
			app.clientError(w, http.StatusUnprocessableEntity)
		case errors.Is(err, model.ErrItemIDMismatch):
			app.clientError(w, http.StatusConflict)
		default:
			app.serverError(w, r, err)
		}
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintln(w, `{"status": "OK"}`)
}
