package main

import (
	"fmt"
	"html/template"
	"net/http"
)

func (app *application) viewWorkspace(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("./ui/html/viewWorkspace.tmpl")
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	type Columns struct {
		Name      string
		WorkItems []string
	}

	type workspace struct {
		Columns []Columns
	}

	ws := workspace{
		Columns: []Columns{
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

	if err := t.Execute(w, ws); err != nil {
		app.serverError(w, r, err)
		return
	}
}

func healthz(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, `{"status": "OK"}`)
}
