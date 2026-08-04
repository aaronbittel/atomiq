package main

import (
	"net/http"
)

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()

	fileServer := http.FileServer(http.Dir("./ui/static/"))
	mux.Handle("GET /static/", http.StripPrefix("/static", fileServer))

	mux.HandleFunc("GET /healthz", http.HandlerFunc(healthz))

	mux.HandleFunc("GET /{$}", app.home)
	mux.HandleFunc("GET /workspaces/{$}", app.workspaceRoot)
	mux.HandleFunc("GET /workspaces/{workspaceID}", app.workspaceView)
	mux.HandleFunc("POST /workspaces/{workspaceID}/work-items", app.workItemPost)
	mux.HandleFunc("DELETE /workspaces/{workspaceID}/work-items/{id}", app.workItemDelete)
	mux.HandleFunc("PATCH /workspaces/{workspaceID}/work-items/{id}/move", app.workItemMove)
	mux.HandleFunc("POST /workspaces/{workspaceID}/work-items/{id}/zoom", app.workItemZoom)
	mux.HandleFunc("GET /workspaces/{workspaceID}/work-items/{id}/edit", app.workItemEditView)
	mux.HandleFunc("PATCH /workspaces/{workspaceID}/work-items/{id}/edit", app.workItemEdit)

	return app.recoverPanic(app.methodOverride(app.logRequest(app.sessionManager.LoadAndSave(mux))))
}
