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
	output, err := s3Client.GetObject(r.Context(), bucketName, path, client.WithRangeHeader(r.Header.Get("Range")))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer output.Body.Close()

	// Set headers từ S3Object
	if output.ContentType != "" {
		w.Header().Set("Content-Type", output.ContentType)
	}
	if output.ContentLength > 0 {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", output.ContentLength))
	}
	if output.ETag != "" {
		w.Header().Set("ETag", output.ETag)
	}
	if output.ContentRange != "" {
		w.Header().Set("Content-Range", output.ContentRange)
	}

	// Copy metadata headers
	for key, value := range output.Metadata {
		// Skip Wasabi-specific headers if they are not desired in the final response
		if strings.Contains(key, "Wasabi") || strings.Contains(value, "Wasabi") {
			continue
		}
		w.Header().Set("x-amz-meta-"+key, value)
	}

	// Determine the final status code
	statusCode := output.StatusCode
	if output.ContentRange != "" {
		// For partial content requests, override Content-Type for compatibility
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
