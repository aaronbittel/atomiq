package main

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
)

type columns struct {
	Name      string
	WorkItems []string
}

type workspace struct {
	Columns []columns
}

func (app *application) viewWorkspace(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("./ui/html/workspaceView.tmpl")
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	t.Option("missingkey=error")

	ws := workspace{
		Columns: []columns{
			{
				Name:      "Backlog",
				WorkItems: []string{"Some Item", "Another Item"},
			},
			{
				Name:      "In Progress",
				WorkItems: []string{"Cool Stuff", "Atomiq", "Hyped"},
			},
			{
				Name:      "Done",
				WorkItems: []string{"Ofc something", "this is also done"},
			},
		},
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, ws); err != nil {
		app.serverError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = buf.WriteTo(w)
}

func healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintln(w, `{"status": "OK"}`)
}
