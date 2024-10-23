package client

import "errors"

var ErrInvalidPath = errors.New("invalid path")

var ErrInvalidBucket = errors.New("invalid bucket")

var ErrStatusCode = errors.New("request failed")
