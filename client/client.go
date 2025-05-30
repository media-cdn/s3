package client

import (
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Client struct {
	pc         *s3.PresignClient
	httpClient *http.Client // Add a shared HTTP client
}

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
	presignClient := s3.NewPresignClient(client)
	return &S3Client{
		pc:         presignClient,
		httpClient: &http.Client{}, // Initialize the shared HTTP client
	}
}

func (c *S3Client) GetObject(
	ctx context.Context,
	bucketName string,
	path string,
	header http.Header,
) (*http.Response, error) {
	if bucketName == "" {
		return nil, ErrInvalidBucket
	}
	if strings.HasSuffix(path, "/") {
		return nil, ErrInvalidPath
	}
	input := &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(path),
		Range:  aws.String(header.Get("Range")),
	}
	presignedGetRequest, err := c.pc.PresignGetObject(ctx, input)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodGet, presignedGetRequest.URL, nil)
	if err != nil {
		return nil, err
	}
	// Chỉ sử dụng URL đã được ký từ SDK, không thêm headers bổ sung
	// để tránh lỗi "headers not signed"
	return c.httpClient.Do(req) // Use the shared HTTP client
}
