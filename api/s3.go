package handler

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/media-cdn/s3/client"
)

var s3Client = client.NewS3Client()

// parseRequestPath tách URL path thành bucket name và object key
func parseRequestPath(urlPath string) (bucket, path string) {
	path = strings.TrimPrefix(urlPath, "/")
	if i := strings.Index(path, "/"); i != -1 {
		return path[:i], path[i+1:]
	}
	return path, ""
}

// applyPrefixToPath áp dụng PREFIX_PATH vào request path nếu được cấu hình
// Trả về path mới đã được thêm prefix
func applyPrefixToPath(originalPath string) string {
	prefix := os.Getenv("PREFIX_PATH")
	if prefix == "" {
		return originalPath
	}

	// Chuẩn hóa prefix và path
	prefix = strings.Trim(prefix, "/")
	path := strings.TrimPrefix(originalPath, "/")

	// Kết hợp prefix và path
	return "/" + prefix + "/" + path
}

// setResponseHeaders thiết lập tất cả response headers từ S3Object
func setResponseHeaders(w http.ResponseWriter, output *client.S3Object) {
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

	// Copy metadata headers với filtering
	for key, value := range output.Metadata {
		if strings.HasPrefix(strings.ToLower(key), "wasabi") ||
			strings.HasPrefix(strings.ToLower(value), "wasabi") {
			continue
		}
		w.Header().Set("x-amz-meta-"+key, value)
	}
}

// getStatusCode xác định HTTP status code phù hợp
func getStatusCode(output *client.S3Object) int {
	if output.ContentRange != "" {
		return http.StatusPartialContent
	}
	return output.StatusCode
}

// writeResponseBody copy S3 object body với error handling
func writeResponseBody(w http.ResponseWriter, r *http.Request, body io.ReadCloser) {
	if _, err := io.Copy(w, body); err != nil {
		// Chỉ log lỗi thực sự, không log khi client disconnect
		if r.Context().Err() == nil {
			log.Printf("Error copying S3 object body: %v", err)
		}
	}
}

// Handler xử lý HTTP request cho S3 object
func Handler(w http.ResponseWriter, r *http.Request) {
	// 1. Áp dụng prefix TRƯỚC KHI parse (tuân thủ SRP)
	prefixedPath := applyPrefixToPath(r.URL.Path)

	// 2. Parse request path từ fullPath đã xử lý prefix
	bucket, path := parseRequestPath(prefixedPath)

	// 3. Fetch S3 object
	output, err := s3Client.GetObject(r.Context(), bucket, path,
		client.WithRangeHeader(r.Header.Get("Range")))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer output.Body.Close()

	// 4. Set response headers
	setResponseHeaders(w, output)

	// 5. Write status code and body
	w.WriteHeader(getStatusCode(output))
	writeResponseBody(w, r, output.Body)
}
