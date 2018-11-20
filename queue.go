package main

import (
	"github.com/beeker1121/goque"
	"sync"
	"sync/atomic"
)

const (
	threshold = 5000
)

type BufferedQueue struct {
	q *goque.Queue
	outBuf []Job
	m sync.Mutex
}

func OpenQueue(dataDir string) (bq *BufferedQueue, err error) {
	bq = new(BufferedQueue)
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

	if len(q.outBuf) < threshold {
		q.outBuf = append(q.outBuf, *job)
		return true
	} else {
		return false
	}
}

func (q *BufferedQueue) directDequeue(job *Job) bool {
	q.m.Lock()
	defer q.m.Unlock()

	if len(q.outBuf) > 0 {
		*job = q.outBuf[0]
		q.outBuf = q.outBuf[1:]
		return true
	} else {
		return false
	}
}

func (q *BufferedQueue) Close() error {
	return q.q.Close()
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
