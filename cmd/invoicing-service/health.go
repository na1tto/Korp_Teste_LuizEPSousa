package main

import (
	"net/http"
)

// getHealth godoc
//
//	@Summary		Shows the server health
//	@Description	Fetches the user feed with flags for filtering
//	@Tags			Health
//	@Accept			json
//	@Produce		json
//	@Failure		500	{object}	error
//	@Router			/health [get]
func (app *application) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	data := map[string]string{
		"status":  "ok",
		"env":     app.config.env,
		"version": version,
	}

	if err := app.jsonResponse(w, http.StatusOK, data); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}
