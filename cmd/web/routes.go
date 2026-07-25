package main

import (
	"net/http"
)

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()

	fileServer := http.FileServer(http.Dir("./ui/static/"))
	mux.Handle("GET /static/", http.StripPrefix("/static", fileServer))

	mux.HandleFunc("GET /healthz", http.HandlerFunc(healthz))

	mux.HandleFunc("GET /{$}", http.HandlerFunc(app.workspaceView))
	mux.HandleFunc("POST /work-item", http.HandlerFunc(app.workItemPost))
	mux.HandleFunc("DELETE /work-item/{id}", http.HandlerFunc(app.workItemDelete))

	return app.recoverPanic(app.methodOverride(app.logRequest(app.sm.LoadAndSave(mux))))
}
