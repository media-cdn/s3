package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	handler "github.com/media-cdn/s3/api"
	// Re-import the client package
)

func main() {
	// start the server
	port := ":8080"
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.StripSlashes)
	r.Use(middleware.CleanPath)
	r.Get("/{bucket}/*", handler.Handler)
	if err := http.ListenAndServe(port, r); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
