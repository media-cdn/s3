package client

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func TestS3Client_Interface(t *testing.T) {
	// Test interface implementation
	var _ Downloader = (*S3Client)(nil)
}

func TestWithRangeHeader(t *testing.T) {
	tests := []struct {
		name        string
		rangeHeader string
		expectRange bool
	}{
		{
			name:        "empty range header",
			rangeHeader: "",
			expectRange: false,
		},
		{
			name:        "valid range header",
			rangeHeader: "bytes=0-1023",
			expectRange: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := &s3.GetObjectInput{}
			opt := WithRangeHeader(tt.rangeHeader)
			opt(input)

			if tt.expectRange && input.Range == nil {
				t.Error("expected Range to be set")
			}
			if !tt.expectRange && input.Range != nil {
				t.Error("expected Range to be nil")
			}
		})
	}
}

func TestWithRange(t *testing.T) {
	input := &s3.GetObjectInput{}
	opt := WithRange(0, 1023)
	opt(input)

	if input.Range == nil {
		t.Fatal("expected Range to be set")
	}

	expected := "bytes=0-1023"
	if *input.Range != expected {
		t.Errorf("expected Range=%q, got %q", expected, *input.Range)
	}
}
