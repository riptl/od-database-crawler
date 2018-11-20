package main

import (
	"github.com/beeker1121/goque"
	"os"
	"sync"
	"sync/atomic"
)

type BufferedQueue struct {
	dataDir string
	q       *goque.Queue
	buf     []Job
	m       sync.Mutex
}

func OpenQueue(dataDir string) (bq *BufferedQueue, err error) {
	bq = new(BufferedQueue)
	if config.JobBufferSize < 0 {
		return
	}
	bq.dataDir = dataDir
	bq.q, err = goque.OpenQueue(dataDir)
	if err != nil { return nil, err }
	return
}

func (q *BufferedQueue) Enqueue(job *Job) error {
	atomic.AddInt64(&totalQueued, 1)
	if q.directEnqueue(job) {
		return nil
	}

	var gob JobGob
	gob.ToGob(job)
	_, err := q.q.EnqueueObject(gob)
	return err
}

func (q *BufferedQueue) Dequeue() (job Job, err error) {
	if q.directDequeue(&job) {
		atomic.AddInt64(&totalQueued, -1)
		return job, nil
	}

	if config.JobBufferSize < 0 {
		err = goque.ErrEmpty
		return
	}

	var item *goque.Item
	item, err = q.q.Dequeue()
	if err != nil { return }

	atomic.AddInt64(&totalQueued, -1)

	var gob JobGob
	err = item.ToObject(&gob)
	if err != nil { return }
	gob.FromGob(&job)

	return
}

func (q *BufferedQueue) directEnqueue(job *Job) bool {
	q.m.Lock()
	defer q.m.Unlock()

	bs := config.JobBufferSize
	if len(q.buf) < bs || bs < 0 {
		q.buf = append(q.buf, *job)
		return true
	} else {
		return false
	}
}

func (q *BufferedQueue) directDequeue(job *Job) bool {
	q.m.Lock()
	defer q.m.Unlock()

	if len(q.buf) > 0 {
		*job = q.buf[0]
		q.buf = q.buf[1:]
		return true
	} else {
		return false
	}
}

// Always returns nil (But implements io.Closer)
func (q *BufferedQueue) Close() error {
	if config.JobBufferSize < 0 {
		return nil
	}

	// Close ignoring errors
	q.q.Close()

	// Delete files
	if err := os.RemoveAll(q.dataDir);
		err != nil { panic(err) }

	return nil
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
