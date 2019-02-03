package main

import (
	"errors"
	"fmt"
	"github.com/valyala/fasthttp"
	"net"
)

var ErrRateLimit = errors.New("too many requests")
var ErrKnown     = errors.New("already crawled")

type HttpError struct {
	code int
}

func (e HttpError) Error() string {
	return fmt.Sprintf("http status %d", e.code)
}

func shouldRetry(err error) bool {
	// HTTP errors
	if httpErr, ok := err.(*HttpError); ok {
		switch httpErr.code {
		case fasthttp.StatusTooManyRequests:
			return true
		default:
			// Don't retry HTTP error codes
			return false
		}
	}

	if dnsError, ok := err.(*net.DNSError); ok {
		// Don't retry permanent DNS errors
		return dnsError.IsTemporary
	}

	if netErr, ok := err.(*net.OpError); ok {
		// Don't retry permanent network errors
		return netErr.Temporary()
	}

	// Retry by default
	return true
}
