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
	gob.Register(&ColumnErr{})
}

type application struct {
	workspaceModel *model.WorkspaceModel
	sessionManager *scs.SessionManager

	logger *slog.Logger
}

func main() {
	port := flag.String("port", ":3888", "web port")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	ws := model.NewWorkspace(
		model.NewColumn(
			"Backlog",
			model.NewWorkItem("Some Item"),
			model.NewWorkItem("Another Item"),
		),
		model.NewColumn(
			"In Progress",
			model.NewWorkItem("Cool Stuff"),
			model.NewWorkItem("Atomiq"),
			model.NewWorkItem("Hyped"),
		),
		model.NewColumn(
			"Done",
			model.NewWorkItem("Ofc something"),
			model.NewWorkItem("this is also done"),
		),
	)

	app := application{
		workspaceModel: model.NewWorkspaceModel(ws),
		sessionManager: scs.New(),
		logger:         logger,
	}

	logger.Info("server starting", "port", *port)

	err := http.ListenAndServe(*port, app.routes())
	logger.Error("server error", "err", err)
	os.Exit(1)
}
