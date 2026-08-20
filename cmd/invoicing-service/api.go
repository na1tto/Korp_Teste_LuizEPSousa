package main

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/naitto/korperp-challenge/internal/store"
	"go.uber.org/zap"
)

type application struct {
	config serverConfig
	store  store.Storage
	logger *zap.SugaredLogger
}

type serverConfig struct {
	addr   string
	db     dbConfig
	env    string
	apiURL string
}

type dbConfig struct {
	addr         string
	maxOpenConns int
	maxIdleConns int
	maxIdleTime  string
}

func (app *application) mount() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Timeout(60 * time.Second))

	r.Route("/v1", func(r chi.Router) {
		r.Get("/health", app.healthCheckHandler)
	})

	return r
}

func (app *application) run(mux http.Handler) error {

	srv := &http.Server{
		Addr:         app.config.addr,
		Handler:      mux,
		WriteTimeout: time.Second * 30,
		ReadTimeout:  time.Second * 10,
		IdleTimeout:  time.Minute,
	}

	app.logger.Infow("invoicing server has started at port", "addr", app.config.apiURL, "env", app.config.env)
	err := srv.ListenAndServe()

	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	if err != nil {
		return err
	}

	app.logger.Infow("invoicing server has stopped", "addr", app.config.addr, "env", app.config.env)

	return nil
}
