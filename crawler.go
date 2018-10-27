package main

import (
	"fmt"
	"net/url"
)

const (
	maxTimeoutRetries = 3
)

type File struct {
	Name string `json:"name"`
	Size int64 `json:"size"`
	Mtime int `json:"mtime"`
	Path string `json:"path"`
	IsDir bool `json:"-"`
}

type RemoteDir interface {
	ListDir(path string)
}

func GetRemoteDir(u url.URL) (RemoteDir, error) {
	switch u.Scheme {
	case "http", "https":
		return nil, nil //&HttpDirectory{}, nil
	default:
		return nil, fmt.Errorf("unsupported scheme: %s", u.Scheme)
	}
}

type CrawlResult struct {
	FileCount int
	Status string
}

type RemoteDirCrawler struct {
	Url string
	MaxThreads int
	// CrawledPaths
	StatusCode string
}

func (r *RemoteDirCrawler) CrawlDir(outFile string) CrawlResult {
	return CrawlResult{}
}
