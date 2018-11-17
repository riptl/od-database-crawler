package main

import (
	"errors"
	"fmt"
)

var ErrRateLimit = errors.New("too many requests")
var ErrKnown     = errors.New("already crawled")

type HttpError struct {
	code int
}

func (e HttpError) Error() string {
	return fmt.Sprintf("http status %d", e.code)
}
