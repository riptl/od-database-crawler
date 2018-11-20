package main

import (
	"github.com/terorie/od-database-crawler/ds/redblackhash"
	"github.com/terorie/od-database-crawler/fasturl"
	"sync"
	"time"
)

type Task struct {
	WebsiteId uint64 `json:"website_id"`
	Url       string `json:"url"`
}

type TaskResult struct {
	StatusCode    string    `json:"status_code"`
	FileCount     uint64    `json:"file_count"`
	ErrorCount    uint64    `json:"-"`
	StartTime     time.Time `json:"-"`
	StartTimeUnix int64     `json:"start_time"`
	EndTimeUnix   int64     `json:"end_time"`
	WebsiteId     uint64    `json:"website_id"`
}

type Job struct {
	Uri       fasturl.URL
	UriStr    string
	Fails     int
	LastError error
}

type JobGob struct {
	Uri string
	Fails int
	LastError string
}

func (g *JobGob) ToGob(j *Job) {
	g.Uri = j.UriStr
	g.Fails = j.Fails
	if j.LastError != nil {
		g.LastError = j.LastError.Error()
	}
}

func (g *JobGob) FromGob(j *Job) {
	if err := j.Uri.Parse(g.Uri);
		err != nil { panic(err) }
	j.UriStr = g.Uri
	j.Fails = g.Fails
	if g.LastError != "" {
		j.LastError = errorString(g.LastError)
	}
}

type OD struct {
	Task    Task
	Result  TaskResult
	Wait    sync.WaitGroup
	BaseUri fasturl.URL
	WCtx    WorkerContext
	Scanned redblackhash.Tree
}

type File struct {
	Name  string `json:"name"`
	Size  int64  `json:"size"`
	MTime int64  `json:"mtime"`
	Path  string `json:"path"`
	IsDir bool   `json:"-"`
}

func (o *OD) LoadOrStoreKey(k *redblackhash.Key) (exists bool) {
	o.Scanned.Lock()
	defer o.Scanned.Unlock()

	exists = o.Scanned.Get(k)
	if exists { return true }

	o.Scanned.Put(k)
	return false
}

type errorString string
func (e errorString) Error() string {
	return string(e)
}
