package main

import (
	"fmt"
	"net/http"
	"time"
)

func (app *application) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			now    = time.Now()
			method = r.Method
			proto  = r.Proto
			uri    = r.URL.Path
		)

		defer func() {
			app.logger.Info("request received", "method", method, "uri", uri, "proto", proto, "took", time.Since(now))
		}()

		next.ServeHTTP(w, r)
	})
}

func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			pv := recover()
			if pv != nil {
				app.serverError(w, r, fmt.Errorf("%v", pv))
				return
			}
		}()

		next.ServeHTTP(w, r)
	})
}
