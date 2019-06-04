package main

import (
	"context"
	"log"
	"net/http"
	"net/http/pprof"
	"os"

	"github.com/goofin/scopby/logger"
	"github.com/goofin/scopby/models"
	"github.com/goofin/scopby/rest"
	"github.com/goofin/scopby/server/dbserver"
	"github.com/zeebo/errs"
	"github.com/zeebo/mon"
	"github.com/zeebo/mon/monhandler"
	"go.uber.org/zap"
)

func main() {
	defer func() { _ = logger.Logger.Sync() }()
	defer zap.ReplaceGlobals(logger.Logger)()
	defer zap.RedirectStdLog(logger.Logger)()

	if err := run(context.Background()); err != nil {
		log.Fatalf("%+v", err)
	}
}

func run(ctx context.Context) (err error) {
	defer mon.Start().Stop(&err)

	db, err := models.Open("sqlite3", "db.sqlite3")
	if err != nil {
		return errs.Wrap(err)
	}
	defer db.Close()

	if len(os.Args) > 1 && os.Args[1] == "init" {
		if err := doInit(ctx, db); err != nil {
			return err
		}
	}

	srv := dbserver.New(db)
	res := rest.New(srv)

	var mux http.ServeMux
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	mux.Handle("/mon/", http.StripPrefix("/mon", monhandler.Handler{}))
	mux.Handle("/", res)

	return errs.Wrap(http.ListenAndServe(":8080", logger.Handler(&mux)))
}

func doInit(ctx context.Context, db *models.DB) (err error) {
	defer mon.Start().Stop(&err)

	_, err = db.ExecContext(ctx, db.Schema())
	return errs.Wrap(err)
}
