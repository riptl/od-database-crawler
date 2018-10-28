package main

import "errors"

var ErrRateLimit = errors.New("too many requests")
var ErrForbidden = errors.New("access denied")
var ErrKnown     = errors.New("already crawled")

