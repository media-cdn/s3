package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	// register the handler
	s3Handler := newS3Handler()

	// start the server
	port := ":8080"
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.StripSlashes)
	r.Use(middleware.CleanPath)
	r.Get("/{bucket}/*", s3Handler.servingContent)
	if err := http.ListenAndServe(port, r); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}

func newS3Client() *s3.Client {
	return s3.New(s3.Options{
		BaseEndpoint: aws.String(os.Getenv("ENDPOINT")),
		Region:       os.Getenv("REGION"),
		Credentials: aws.NewCredentialsCache(
			credentials.NewStaticCredentialsProvider(
				os.Getenv("KEY"),
				os.Getenv("SECRET"),
				"",
			),
		),
	})
}

type s3Handler struct {
	s3Client *s3.Client
}

func newS3Handler() *s3Handler {
	return &s3Handler{
		s3Client: newS3Client(),
	}
}

// The servingContent function is the HTTP handler that takes a request, extracts the bucket name, object path from the parameters,
// generates a signed URL for the private S3 object, and then response file content to client.
func (h *s3Handler) servingContent(w http.ResponseWriter, r *http.Request) {
	bucket := chi.URLParam(r, "bucket")
	path := chi.URLParam(r, "*")
	if strings.HasSuffix(path, "/") {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	// generate presigned URL
	presignClient := s3.NewPresignClient(h.s3Client)
	object := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(path),
		Range:  aws.String(r.Header.Get("Range")),
	}
	presignedGetRequest, err := presignClient.PresignGetObject(r.Context(), object)
	if err != nil {
		log.Printf("failed to presign request: %v", err)
		http.Error(
			w,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
		return
	}

	// new http reqest
	req, err := http.NewRequest(http.MethodGet, presignedGetRequest.URL, nil)
	if err != nil {
		log.Printf("failed to create request: %v", err)
		http.Error(
			w,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
		return
	}

	// set request headers
	for key, values := range r.Header {
		for _, value := range values {
			// ignore header: If-Modified-Since, If-None-Match to prevent status 304
			if key == "If-Modified-Since" || key == "If-None-Match" {
				continue
			}

			req.Header.Add(key, value)
		}
	}
	// set header no cache to prevent status 304
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")

	httpClient := http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("failed to get response: %v", err)
		http.Error(
			w,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
		return
	}
	defer resp.Body.Close()

	// check status code
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		log.Printf("failed to get response: %v", err)
		http.Error(
			w,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
		return
	}

	// set response headers
	for key, values := range resp.Header {
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

	// check if the response contains partial content
	// for streaming content like video, audio, etc.
	if resp.Header.Get("Content-Range") != "" {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", resp.ContentLength))
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusPartialContent)
	}

	// write response body to client

	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Printf("failed to write response: %v", err)
		http.Error(
			w,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
		return
	}
}
