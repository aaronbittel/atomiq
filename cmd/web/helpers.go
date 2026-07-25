package main

import (
	"net/http"
	"runtime/debug"
)

func (app *application) serverError(w http.ResponseWriter, r *http.Request, err error) {
	var (
		stack  = debug.Stack()
		method = r.Method
		uri    = r.URL.Path
	)

	app.logger.Error("server error", "err", err, "method", method, "uri", uri, "stack", stack)

	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func (app *application) clientError(w http.ResponseWriter, status int) {
	http.Error(w, http.StatusText(status), status)
}
