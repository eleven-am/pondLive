package server_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/eleven-am/pondlive/go/internal/server"
)

// This example shows how to build a complete application with server.
func ExampleApp() {
	ctx := context.Background()
	app := server.NewApp(ctx)

	wsRoute := "/ws"
	app.Handle(wsRoute, app.PondManager().HTTPHandler())

	api := app.NewGroup("/api")

	api.HandleFunc("POST /nav", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			SessionID string `json:"sid"`
			Path      string `json:"path"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			json.NewEncoder(w).Encode(map[string]any{
				"success": false,
				"error":   err.Error(),
			})
			return
		}

		json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"patches": []any{},
		})
	})

	api.HandleFunc("GET /users/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		json.NewEncoder(w).Encode(map[string]any{
			"id":   id,
			"name": "John Doe",
		})
	})

	app.Handle("/static/", http.StripPrefix("/static/", http.FileServerFS(nil)))

	app.HandleFunc("GET /{$}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Root page")
	}))

	app.HandleFunc("GET /{path...}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.PathValue("path")
		fmt.Fprintf(w, "Page: %s", path)
	}))

	http.ListenAndServe(":3000", app.Handler())
}
