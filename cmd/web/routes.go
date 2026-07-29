package main

import (
	"net/http"
)

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()

	fileServer := http.FileServer(http.Dir("./ui/static/"))
	mux.Handle("GET /static/", http.StripPrefix("/static", fileServer))

	mux.HandleFunc("GET /healthz", http.HandlerFunc(healthz))

	mux.HandleFunc("GET /{$}", app.workspaceView)
	mux.HandleFunc("POST /work-item", app.workItemPost)
	mux.HandleFunc("DELETE /work-item/{id}", app.workItemDelete)
	mux.HandleFunc("PATCH /work-item/{id}/move", app.workItemMove)

	return app.recoverPanic(app.methodOverride(app.logRequest(app.sessionManager.LoadAndSave(mux))))
}
