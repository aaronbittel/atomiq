package main

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/aaronbittel/atomiq/internal/model"
)

type workspaceRenderView struct {
	Ws        model.Workspace
	ColumnErr ColumnErr
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

	data := workspaceRenderView{Ws: app.wm.Workspace}
	if columnErr, ok := app.sm.Pop(r.Context(), "name").(ColumnErr); ok {
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
		http.Error(w, http.StatusText(http.StatusUnprocessableEntity), http.StatusUnprocessableEntity)
		return
	}

	columnIdxStr := r.PostForm.Get("columnIdx")
	if columnIdxStr == "" {
		app.clientError(w, http.StatusUnprocessableEntity)
		return
	}
	columnIdx, err := strconv.Atoi(columnIdxStr)
	if err != nil {
		app.clientError(w, http.StatusUnprocessableEntity)
		return
	}

	workItemName := strings.TrimSpace(r.PostForm.Get("name"))
	if workItemName == "" {
		app.sm.Put(r.Context(), "name", ColumnErr{Idx: columnIdx, Msg: "work item must not be blank"})
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if err := app.wm.AddWorkItem(columnIdx, workItemName); err != nil {
		app.serverError(w, r, err)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintln(w, `{"status": "OK"}`)
}
