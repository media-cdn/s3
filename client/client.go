package client

import (
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func NewS3Client() *s3.Client {
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
	}
	return s3.New(options)
}
