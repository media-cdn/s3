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
}
