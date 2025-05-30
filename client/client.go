package client

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Object đại diện cho object tải từ S3
type S3Object struct {
	Body          io.ReadCloser     // Stream dữ liệu
	ContentType   string            // Loại nội dung
	ContentLength int64             // Kích thước nội dung
	ContentRange  string            // Range cho partial content
	ETag          string            // Entity tag
	Metadata      map[string]string // Metadata bổ sung
	StatusCode    int               // HTTP status code
}

// Downloader định nghĩa interface tải object
type Downloader interface {
	// GetObject tải object với các tuỳ chọn
	GetObject(ctx context.Context, bucket, key string, opts ...DownloadOption) (*S3Object, error)

	// HeadObject lấy metadata object
	HeadObject(ctx context.Context, bucket, key string) (*S3Object, error)

	// Close đóng kết nối
	Close() error
}

// DownloadOption tuỳ chỉnh hành vi tải
type DownloadOption func(*s3.GetObjectInput)

// WithRange tuỳ chọn tải phần nội dung cụ thể
func WithRange(start, end int64) DownloadOption {
	return func(input *s3.GetObjectInput) {
		input.Range = aws.String(fmt.Sprintf("bytes=%d-%d", start, end))
	}
}

// WithRangeHeader tuỳ chọn sử dụng Range header từ HTTP request
func WithRangeHeader(rangeHeader string) DownloadOption {
	return func(input *s3.GetObjectInput) {
		if rangeHeader != "" {
			input.Range = aws.String(rangeHeader)
		}
	}
}

// S3Client triển khai Downloader cho AWS S3
type S3Client struct {
	client *s3.Client
}

// NewS3Client tạo S3Client mới
func NewS3Client() *S3Client {
	options := s3.Options{
		BaseEndpoint: aws.String(os.Getenv("ENDPOINT")),
		Region:       os.Getenv("REGION"),
		Credentials: aws.NewCredentialsCache(
			credentials.NewStaticCredentialsProvider(
				os.Getenv("KEY"),
				os.Getenv("SECRET"),
				"",
			),
		),
		UsePathStyle: os.Getenv("USE_PATH_STYLE") == "true",
	}
	client := s3.New(options)
	return &S3Client{
		client: client,
	}
}

// GetObject triển khai Downloader interface
func (c *S3Client) GetObject(
	ctx context.Context,
	bucket, key string,
	opts ...DownloadOption,
) (*S3Object, error) {
	if bucket == "" {
		return nil, ErrInvalidBucket
	}
	if strings.HasSuffix(key, "/") {
		return nil, ErrInvalidPath
	}

	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	// Áp dụng các tuỳ chọn
	for _, opt := range opts {
		opt(input)
	}

	output, err := c.client.GetObject(ctx, input)
	if err != nil {
		return nil, err
	}

	// Xác định status code
	statusCode := 200
	if input.Range != nil {
		statusCode = 206 // Partial Content
	}

	return &S3Object{
		Body:          output.Body,
		ContentType:   aws.ToString(output.ContentType),
		ContentLength: aws.ToInt64(output.ContentLength),
		ContentRange:  aws.ToString(output.ContentRange),
		ETag:          aws.ToString(output.ETag),
		Metadata:      output.Metadata,
		StatusCode:    statusCode,
	}, nil
}

// HeadObject lấy metadata object không tải nội dung
func (c *S3Client) HeadObject(
	ctx context.Context,
	bucket, key string,
) (*S3Object, error) {
	if bucket == "" {
		return nil, ErrInvalidBucket
	}
	if strings.HasSuffix(key, "/") {
		return nil, ErrInvalidPath
	}

	input := &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	output, err := c.client.HeadObject(ctx, input)
	if err != nil {
		return nil, err
	}

	return &S3Object{
		Body:          nil, // HeadObject không có body
		ContentType:   aws.ToString(output.ContentType),
		ContentLength: aws.ToInt64(output.ContentLength),
		ETag:          aws.ToString(output.ETag),
		Metadata:      output.Metadata,
		StatusCode:    200,
	}, nil
}

// Close đóng kết nối (placeholder cho future implementation)
func (c *S3Client) Close() error {
	// S3 client không cần đóng kết nối explicit
	return nil
}
