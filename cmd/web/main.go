package main

import (
	"flag"
	"log/slog"
	"net/http"
	"os"
)

type application struct {
	logger *slog.Logger
}

func main() {
	port := flag.String("port", ":3888", "web port")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	app := application{
		logger: logger,
	}

	logger.Info("server starting", "port", *port)

	err := http.ListenAndServe(*port, app.routes())
	logger.Error("server error", "err", err)
	os.Exit(1)
}
