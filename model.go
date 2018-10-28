package main

import (
	"net/url"
	"sync"
	"time"
)

type Job struct {
	OD        *OD
	Uri       url.URL
	UriStr    string
	Fails     int
	LastError error
}

type OD struct {
	Wait    sync.WaitGroup
	BaseUri url.URL
	lock    sync.Mutex
	Files   []File
	WCtx    WorkerContext
	Scanned sync.Map
}

type File struct {
	Name  string    `json:"name"`
	Size  int64     `json:"size"`
	MTime time.Time `json:"mtime"`
	Path  string    `json:"path"`
	IsDir bool      `json:"-"`
}
