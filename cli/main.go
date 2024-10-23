package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/media-cdn/s3/client"
)

func main() {
	// register the handler
	s3Client := client.NewS3Client()
	// start the server
	port := ":8080"
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.StripSlashes)
	r.Use(middleware.CleanPath)
	r.Get("/{bucket}/*", func(w http.ResponseWriter, r *http.Request) {
		bucketName := chi.URLParam(r, "bucket")
		path := chi.URLParam(r, "*")
		output, err := s3Client.GetObject(r.Context(), bucketName, path, r.Header)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer output.Body.Close()
		w.WriteHeader(output.StatusCode)
		for key, values := range output.Header {
			for _, value := range values {
				if strings.Contains(key, "Wasabi") {
					continue
				}
				if strings.Contains(value, "Wasabi") {
					continue
				}
				w.Header().Add(key, value)
			}
		}
		if output.Header.Get("Content-Range") != "" {
			w.Header().Set("Content-Length", fmt.Sprintf("%d", output.ContentLength))
			w.Header().Set("Content-Type", "application/octet-stream")
			w.WriteHeader(http.StatusPartialContent)
		}

		if _, err := io.Copy(w, output.Body); err != nil {
			log.Println(err)
		}
	})
	if err := http.ListenAndServe(port, r); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
