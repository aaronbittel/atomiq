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

type workspaceRenderView struct {
	Ws        model.Workspace
	ColumnErr *ColumnErr
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

	data := workspaceRenderView{Ws: app.wm.WorkspaceView()}
	if columnErr, ok := app.sm.Pop(r.Context(), "columnErr").(*ColumnErr); ok {
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
		app.sm.Put(r.Context(), "columnErr", &ColumnErr{Idx: columnIdx, Msg: "work item must not be blank"})
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if err := app.wm.WorkItemAdd(columnIdx, workItemName); err != nil {
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

	workItemID := r.PathValue("id")
	if len(workItemID) != 8 {
		app.clientError(w, http.StatusUnprocessableEntity)
		return
	}

	columnIdx, err := parseInt(r.PostForm.Get("columnIdx"))
	if err != nil {
		app.clientError(w, http.StatusUnprocessableEntity)
		return
	}

	if err := app.wm.WorkItemDelete(columnIdx, workItemID); err != nil {
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

func healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintln(w, `{"status": "OK"}`)
}
