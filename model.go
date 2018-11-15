package main

import (
	"github.com/terorie/oddb-go/ds/redblackhash"
	"github.com/terorie/oddb-go/fasturl"
	"sync"
	"time"
)

type Job struct {
	OD        *OD
	Uri       fasturl.URL
	UriStr    string
	Fails     int
	LastError error
}

type OD struct {
	Task    *Task
	Wait    sync.WaitGroup
	BaseUri fasturl.URL
	WCtx    WorkerContext
	Scanned redblackhash.Tree

	lock    sync.Mutex
}

type File struct {
	Name  string    `json:"name"`
	Size  int64     `json:"size"`
	MTime time.Time `json:"mtime"`
	Path  string    `json:"path"`
	IsDir bool      `json:"-"`
}

func (o *OD) LoadOrStoreKey(k *redblackhash.Key) (exists bool) {
	o.lock.Lock()
	defer o.lock.Unlock()

	exists = o.Scanned.Get(k)
	if exists { return true }

	o.Scanned.Put(k)
	return false
}
