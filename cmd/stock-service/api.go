package main

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
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
	r.Use(app.recoverPanic)
	r.Use(middleware.Logger)

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:4200"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Route("/v1", func(r chi.Router) {
		r.Get("/health", app.healthCheckHandler)

		r.Route("/products", func(r chi.Router) {
			r.Get("/", app.listProductsHandler)
			r.Post("/", app.createProductHandler)
			r.Post("/deduct", app.deductStockHandler)
		})
	})

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		app.notFoundResponse(w, r, errRouteNotFound)
	})
	r.MethodNotAllowed(app.methodNotAllowedResponse)

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

	app.logger.Infow("stock server has started at port", "addr", app.config.apiURL, "env", app.config.env)
	err := srv.ListenAndServe()

	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
