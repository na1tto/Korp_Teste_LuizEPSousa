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
		addr:   env.GetString("INVOICING_ADDR", ":3001"),
		env:    env.GetString("ENV", "DEVELOPMENT"),
		apiURL: env.GetString("INVOICING_EXTERNAL_URL", "localhost:3000"),
		db: dbConfig{
			addr:         env.GetString("DB_ADDR", "postgres://admin:adminpassword@localhost:5432/korp_db?sslmode=disable"),
			maxOpenConns: env.GetInt("DB_MAX_OPEN_CONNS", 30),
			maxIdleConns: env.GetInt("DB_MAX_IDLE_CONNS", 30),
			maxIdleTime:  env.GetString("DB_MAX_IDLE_TIME", "15m"),
		},
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

	store := store.NewStorage(db)

	app := &application{
		config: cfg,
		store:  store,
		logger: logger,
	}

	mux := app.mount()
	logger.Fatal(app.run(mux))
}
