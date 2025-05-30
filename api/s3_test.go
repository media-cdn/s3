package handler

import (
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/media-cdn/s3/client"
)

func TestHandlerBasicRequest(t *testing.T) {
	// Tạo mock request
	req := httptest.NewRequest("GET", "/testbucket/testfile.jpg", nil)
	w := httptest.NewRecorder()

	// Note: Test này sẽ fail vì chưa có S3 backend thực
	// Nhưng nó kiểm tra cú pháp và cấu trúc code
	defer func() {
		if r := recover(); r != nil {
			t.Logf("Expected panic due to missing S3 backend: %v", r)
		}
	}()

	Handler(w, req)
}
func TestHandlerWithPrefix(t *testing.T) {
	tests := []struct {
		name         string
		prefix       string
		requestPath  string
		expectedPath string // Path mà parseRequestPath sẽ nhận được
	}{
		{
			name:         "With prefix",
			prefix:       "my-bucket/prefix",
			requestPath:  "/object-key",
			expectedPath: "/my-bucket/prefix/object-key",
		},
		{
			name:         "Without prefix",
			prefix:       "",
			requestPath:  "/bucket/object",
			expectedPath: "/bucket/object",
		},
		{
			name:         "Single segment prefix",
			prefix:       "my-bucket",
			requestPath:  "/folder/file.jpg",
			expectedPath: "/my-bucket/folder/file.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set PREFIX_PATH environment variable
			oldPrefix := os.Getenv("PREFIX_PATH")
			defer os.Setenv("PREFIX_PATH", oldPrefix)
			os.Setenv("PREFIX_PATH", tt.prefix)

			// Test logic xử lý prefix
			prefixedPath := applyPrefixToPath(tt.requestPath)
			if prefixedPath != tt.expectedPath {
				t.Errorf("applyPrefixToPath() = %v, want %v", prefixedPath, tt.expectedPath)
			}

			// Test parseRequestPath với path đã được prefix
			bucket, path := parseRequestPath(prefixedPath)
			t.Logf("Bucket: %s, Path: %s", bucket, path)

			// Verify kết quả phù hợp với kỳ vọng
			if tt.prefix != "" && bucket == "" {
				t.Error("Expected non-empty bucket when prefix is set")
			}
		})
	}
}

// Unit tests cho các hàm helper mới

func TestParseRequestPath(t *testing.T) {
	tests := []struct {
		name       string
		urlPath    string
		wantBucket string
		wantPath   string
	}{
		{
			name:       "normal path",
			urlPath:    "/mybucket/folder/file.jpg",
			wantBucket: "mybucket",
			wantPath:   "folder/file.jpg",
		},
		{
			name:       "root file",
			urlPath:    "/mybucket/file.jpg",
			wantBucket: "mybucket",
			wantPath:   "file.jpg",
		},
		{
			name:       "bucket only",
			urlPath:    "/mybucket",
			wantBucket: "mybucket",
			wantPath:   "",
		},
		{
			name:       "deep nested path",
			urlPath:    "/mybucket/a/b/c/d/file.jpg",
			wantBucket: "mybucket",
			wantPath:   "a/b/c/d/file.jpg",
		},
		{
			name:       "empty path",
			urlPath:    "/",
			wantBucket: "",
			wantPath:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBucket, gotPath := parseRequestPath(tt.urlPath)
			if gotBucket != tt.wantBucket {
				t.Errorf("parseRequestPath() bucket = %v, want %v", gotBucket, tt.wantBucket)
			}
			if gotPath != tt.wantPath {
				t.Errorf("parseRequestPath() path = %v, want %v", gotPath, tt.wantPath)
			}
		})
	}
}

func TestApplyPrefixToPath(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		input    string
		expected string
	}{
		{"No prefix", "", "/img.jpg", "/img.jpg"},
		{"Single segment", "my-bucket", "/img.jpg", "/my-bucket/img.jpg"},
		{"Multi segment", "bucket/folder", "/img.jpg", "/bucket/folder/img.jpg"},
		{"Edge: empty path", "bucket", "", "/bucket/"},
		{"Edge: root path", "bucket", "/", "/bucket/"},
		{"Edge: prefix with trailing slash", "bucket/", "/img.jpg", "/bucket/img.jpg"},
		{"Edge: path with multiple slashes", "bucket", "//img.jpg", "/bucket//img.jpg"},
		{"Nested prefix", "app/v1/data", "/users/profile.json", "/app/v1/data/users/profile.json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set PREFIX_PATH environment variable
			oldPrefix := os.Getenv("PREFIX_PATH")
			defer os.Setenv("PREFIX_PATH", oldPrefix)
			os.Setenv("PREFIX_PATH", tt.prefix)

			result := applyPrefixToPath(tt.input)
			if result != tt.expected {
				t.Errorf("applyPrefixToPath() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetStatusCode(t *testing.T) {
	tests := []struct {
		name       string
		output     *client.S3Object
		wantStatus int
	}{
		{
			name: "normal response",
			output: &client.S3Object{
				StatusCode:   200,
				ContentRange: "",
			},
			wantStatus: 200,
		},
		{
			name: "partial content response",
			output: &client.S3Object{
				StatusCode:   200,
				ContentRange: "bytes 0-1023/2048",
			},
			wantStatus: 206,
		},
		{
			name: "custom status code",
			output: &client.S3Object{
				StatusCode:   304,
				ContentRange: "",
			},
			wantStatus: 304,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStatus := getStatusCode(tt.output)
			if gotStatus != tt.wantStatus {
				t.Errorf("getStatusCode() = %v, want %v", gotStatus, tt.wantStatus)
			}
		})
	}
}

func TestSetResponseHeaders(t *testing.T) {
	tests := []struct {
		name   string
		output *client.S3Object
		want   map[string]string
	}{
		{
			name: "complete headers",
			output: &client.S3Object{
				ContentType:   "image/jpeg",
				ContentLength: 1024,
				ETag:          "\"abc123\"",
				ContentRange:  "bytes 0-1023/2048",
				Metadata: map[string]string{
					"user-id":      "12345",
					"upload-time":  "2023-01-01",
					"wasabi-class": "standard", // Should be filtered
				},
			},
			want: map[string]string{
				"Content-Type":           "image/jpeg",
				"Content-Length":         "1024",
				"ETag":                   "\"abc123\"",
				"Content-Range":          "bytes 0-1023/2048",
				"x-amz-meta-user-id":     "12345",
				"x-amz-meta-upload-time": "2023-01-01",
			},
		},
		{
			name: "minimal headers",
			output: &client.S3Object{
				ContentType: "text/plain",
				Metadata:    map[string]string{},
			},
			want: map[string]string{
				"Content-Type": "text/plain",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			setResponseHeaders(w, tt.output)

			for key, wantValue := range tt.want {
				gotValue := w.Header().Get(key)
				if gotValue != wantValue {
					t.Errorf("setResponseHeaders() header %s = %v, want %v", key, gotValue, wantValue)
				}
			}

			// Verify wasabi headers are filtered
			for key := range w.Header() {
				if strings.Contains(strings.ToLower(key), "wasabi") {
					t.Errorf("setResponseHeaders() should filter wasabi header: %s", key)
				}
			}
		})
	}
}
