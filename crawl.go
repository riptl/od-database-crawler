package main

import (
	"time"
)

type File struct {
	Name string `json:"name"`
	Size int64 `json:"size"`
	MTime time.Time `json:"mtime"`
	Path string `json:"path"`
	IsDir bool `json:"-"`
}

type CrawlResult struct {
	FileCount int
	Status string
}
