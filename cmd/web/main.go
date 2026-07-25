package main

import (
	"encoding/gob"
	"flag"
	"log/slog"
	"net/http"
	"os"

	"github.com/aaronbittel/atomiq/internal/model"
	"github.com/alexedwards/scs/v2"
)

func init() {
	gob.Register(ColumnErr{})
}

type application struct {
	wm *model.WorkspaceModel
	sm *scs.SessionManager

	logger *slog.Logger
}

func main() {
	port := flag.String("port", ":3888", "web port")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	wm := &model.WorkspaceModel{
		Workspace: model.Workspace{
			Columns: []model.Column{
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
		},
	}

	app := application{
		wm:     wm,
		sm:     scs.New(),
		logger: logger,
	}

	logger.Info("server starting", "port", *port)

	err := http.ListenAndServe(*port, app.routes())
	logger.Error("server error", "err", err)
	os.Exit(1)
}
