package main

import (
	_ "github.com/joho/godotenv/autoload"
	"github.com/naitto/korperp-challenge/internal/db"
	"github.com/naitto/korperp-challenge/internal/env"
	"github.com/naitto/korperp-challenge/internal/store"
	"go.uber.org/zap"
)

const version = "0.0.1"

func main() {
	cfg := serverConfig{
		addr:   env.GetString("INVOICING_ADDR", ":3002"),
		env:    env.GetString("ENV", "DEVELOPMENT"),
		apiURL: env.GetString("INVOICING_EXTERNAL_URL", "localhost:3002"),
		db: dbConfig{
			addr:         env.GetString("DB_ADDR", "postgres://admin:adminpassword@localhost:5432/korp_db?sslmode=disable"),
			maxOpenConns: env.GetInt("DB_MAX_OPEN_CONNS", 30),
			maxIdleConns: env.GetInt("DB_MAX_IDLE_CONNS", 30),
			maxIdleTime:  env.GetString("DB_MAX_IDLE_TIME", "15m"),
		},
		stockClientBaseURL: env.GetString("STOCK_EXTERNAL_URL", "http://localhost:3001"),
	}

	logger := zap.Must(zap.NewProduction()).Sugar()
	defer logger.Sync()

	db, err := db.New(
		cfg.db.addr,
		cfg.db.maxOpenConns,
		cfg.db.maxIdleConns,
		cfg.db.maxIdleTime,
	)

	if err != nil {
		logger.Fatal(err)
	}
	defer db.Close()

	store := store.NewStorage(db)

	stockClient := NewStockclient(cfg.stockClientBaseURL)

	app := &application{
		config:      cfg,
		store:       store,
		stockClient: stockClient,
		logger:      logger,
	}

	mux := app.mount()
	if err := app.run(mux); err != nil {
		logger.Fatal(err)
	}
}
