# Refactor S3Client Interface

## Tổng quan
Đã thực hiện refactor toàn bộ interface của S3Client để thay thế cơ chế presigned URL bằng việc tải file trực tiếp từ S3.

## Thay đổi chính

### 1. Interface mới
```go
type Downloader interface {
    GetObject(ctx context.Context, bucket, key string, opts ...DownloadOption) (*S3Object, error)
    HeadObject(ctx context.Context, bucket, key string) (*S3Object, error)
    Close() error
}
```

### 2. Struct S3Object
```go
type S3Object struct {
    Body          io.ReadCloser     // Stream dữ liệu
    ContentType   string            // Loại nội dung
    ContentLength int64             // Kích thước nội dung
    ContentRange  string            // Range cho partial content
    ETag          string            // Entity tag
    Metadata      map[string]string // Metadata bổ sung
    StatusCode    int               // HTTP status code
}
```

### 3. Options Pattern
```go
type DownloadOption func(*s3.GetObjectInput)

func WithRange(start, end int64) DownloadOption
func WithRangeHeader(rangeHeader string) DownloadOption
```

## Lợi ích

### 1. Hiệu suất
- **Trước**: Tạo presigned URL → HTTP request riêng → Thêm latency
- **Sau**: Gọi trực tiếp S3 API → Giảm latency

### 2. Bảo mật  
- Không cần expose presigned URL
- Kiểm soát truy cập tốt hơn

### 3. Linh hoạt
- Options pattern cho các tuỳ chọn download
- Dễ mở rộng với chức năng mới

### 4. Maintainability
- Interface rõ ràng, tách biệt concerns
- Dễ test và mock

## Migration Guide

### API Handler
**Trước**:
```go
output, err := s3Client.GetObject(r.Context(), bucketName, path, r.Header)
// output là *http.Response
```

**Sau**:
```go
output, err := s3Client.GetObject(r.Context(), bucketName, path, 
    client.WithRangeHeader(r.Header.Get("Range")))
// output là *client.S3Object
```

### Header Handling
**Trước**:
```go
for key, values := range output.Header {
    // Copy headers từ http.Response
}
```

**Sau**:
```go
if output.ContentType != "" {
    w.Header().Set("Content-Type", output.ContentType)
}
if output.ContentLength > 0 {
    w.Header().Set("Content-Length", fmt.Sprintf("%d", output.ContentLength))
}
// Xử lý từng field riêng biệt
```

## Breaking Changes
- Thay đổi signature hàm `GetObject`
- Thay đổi kiểu trả về từ `*http.Response` sang `*S3Object`
- Headers không còn được copy tự động

## Tests
Đã thêm unit tests để verify:
- Interface implementation
- Options pattern functionality
- Range header processing

## Files đã thay đổi
- `client/client.go`: Refactor toàn bộ interface
- `api/s3.go`: Cập nhật để sử dụng interface mới
- `client/client_test.go`: Thêm unit tests
