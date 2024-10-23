package handler

import (
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/media-cdn/s3/client"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimLeft(r.URL.Path, "/")
	bucketName := strings.Split(path, "/")[0]
	path = strings.Replace(path, bucketName+"/", "", 1)
	if strings.HasSuffix(path, "/") {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	s3Client := client.NewS3Client()
	input := &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(path),
		Range:  aws.String(r.Header.Get("Range")),
	}
	output, err := s3Client.GetObject(r.Context(), input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Add("Content-Type", *output.ContentType)
	w.Header().Add("Content-Length", strconv.FormatInt(*output.ContentLength, 10))
	w.Header().Add("Accept-Ranges", *output.AcceptRanges)
	w.Header().Add("Last-Modified", output.LastModified.Format(http.TimeFormat))
	w.Header().Add("ETag", *output.ETag)
	w.Header().Add("Cache-Control", "max-age=31536000")
	w.Header().Add("Expires", *output.ExpiresString)
	w.Header().Add("Content-Disposition", *output.ContentDisposition)
	w.Header().Add("Content-Encoding", *output.ContentEncoding)
	w.Header().Add("Content-Range", *output.ContentRange)
	if _, err := io.Copy(w, output.Body); err != nil {
		log.Println(err)
	}
}
