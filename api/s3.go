package handler

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/media-cdn/s3/client"
)

var s3Client = client.NewS3Client()

func Handler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimLeft(r.URL.Path, "/")
	bucketName := strings.Split(path, "/")[0]
	path = strings.Replace(path, bucketName+"/", "", 1)
	output, err := s3Client.GetObject(r.Context(), bucketName, path, r.Header)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer output.Body.Close()

	// Set all headers from S3 response first
	for key, values := range output.Header {
		for _, value := range values {
			// Skip Wasabi-specific headers if they are not desired in the final response
			if strings.Contains(key, "Wasabi") || strings.Contains(value, "Wasabi") {
				continue
			}
			w.Header().Add(key, value)
		}
	}

	// Determine the final status code
	statusCode := output.StatusCode
	if output.Header.Get("Content-Range") != "" {
		// For partial content requests, set Content-Length and Content-Type, and use 206 Partial Content status
		w.Header().Set("Content-Length", fmt.Sprintf("%d", output.ContentLength))
		// The original code explicitly set application/octet-stream for partial content.
		// If the original S3 Content-Type should be preserved, use: w.Header().Set("Content-Type", output.Header.Get("Content-Type"))
		w.Header().Set("Content-Type", "application/octet-stream")
		statusCode = http.StatusPartialContent
	}

	// Write the header with the determined status code
	w.WriteHeader(statusCode)

	// Copy the S3 object body to the HTTP response writer
	if _, err := io.Copy(w, output.Body); err != nil {
		log.Println("Error copying S3 object body to response writer:", err)
	}
}
