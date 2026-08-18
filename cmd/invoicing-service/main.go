package main

import (
	_ "github.com/joho/godotenv/autoload"
	"github.com/naitto/korperp-challenge/internal/env"
	"go.uber.org/zap"
)

const version = "0.0.1"

func main() {
	cfg := serverConfig{
		addr:   env.GetString("INVOICING_ADDR", ":3000"),
		env:    env.GetString("ENV", "DEVELOPMENT"),
		apiURL: env.GetString("INVOICING_EXTERNAL_URL", "localhost:3000"),
	}

	logger := zap.Must(zap.NewProduction()).Sugar()
	defer logger.Sync()

	app := &application{
		config: cfg,
		logger: logger,
	}

	mux := app.mount()
	logger.Fatal(app.run(mux))
}
