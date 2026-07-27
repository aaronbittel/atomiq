package main

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strconv"

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
	Ws        model.WorkspaceView
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

	revision, err := strconv.ParseUint(r.PostForm.Get("revision"), 10, 64)
	if err != nil {
		app.clientError(w, http.StatusUnprocessableEntity)
		return
	}

	workItemName := r.PostForm.Get("name")

	if err := app.workspaceModel.WorkItemAdd(revision, columnIdx, workItemName); err != nil {
		switch {
		case errors.Is(err, model.ErrInvalidPosition):
			app.clientError(w, http.StatusUnprocessableEntity)
		case errors.Is(err, model.ErrInvalidWorkItemName):
			app.sessionManager.Put(r.Context(), "columnErr", &ColumnErr{Idx: columnIdx, Msg: "work item must not be blank"})
			http.Redirect(w, r, "/", http.StatusSeeOther)
		case errors.Is(err, model.ErrRevisionConflict):
			app.clientError(w, http.StatusConflict)
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

	revision, err := strconv.ParseUint(r.PostForm.Get("revision"), 10, 64)
	if err != nil {
		app.clientError(w, http.StatusUnprocessableEntity)
		return
	}

	if err := app.workspaceModel.WorkItemDelete(revision, itemID); err != nil {
		switch {
		case errors.Is(err, model.ErrWorkItemNotFound):
			app.clientError(w, http.StatusNotFound)
		case errors.Is(err, model.ErrRevisionConflict):
			app.clientError(w, http.StatusConflict)
		default:
			app.serverError(w, r, err)
		}
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *application) workItemMove(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		app.clientError(w, http.StatusUnprocessableEntity)
		return
	}

	direction, err := model.ParseMoveDirection(r.PostForm.Get("direction"))
	if err != nil {
		app.clientError(w, http.StatusUnprocessableEntity)
		return
	}

	workItemID := r.PathValue("id")
	if len(workItemID) != model.WorkItemIDLength {
		app.clientError(w, http.StatusUnprocessableEntity)
		return
	}

	revision, err := strconv.ParseUint(r.PostForm.Get("revision"), 10, 64)
	if err != nil {
		app.clientError(w, http.StatusUnprocessableEntity)
		return
	}

	if err := app.workspaceModel.WorkItemMoveDirection(revision, workItemID, direction); err != nil {
		switch {
		case errors.Is(err, model.ErrWorkItemNotFound):
			app.clientError(w, http.StatusNotFound)
		case errors.Is(err, model.ErrRevisionConflict):
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
