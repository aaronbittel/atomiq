package main

import (
	"net/http"
)

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()

	fileServer := http.FileServer(http.Dir("./ui/static/"))
	mux.Handle("GET /static/", http.StripPrefix("/static", fileServer))

	mux.HandleFunc("GET /healthz", http.HandlerFunc(healthz))

	mux.HandleFunc("GET /", http.HandlerFunc(app.viewWorkspace))

	return app.recoverPanic(app.logRequest(mux))
}
