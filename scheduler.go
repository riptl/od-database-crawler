package main

import (
	"context"
	"net/url"
	"sync/atomic"
)

type Job struct {
	Remote *RemoteDir
	Uri url.URL
	UriStr string
	Fails int
	LastError error
}

var activeTasks int32

func Schedule(c context.Context, remotes <-chan *RemoteDir) {
	in, out := makeJobBuffer()
	wCtx := WorkerContext{
		in: in,
		out: out,
	}
	for i := 0; i < config.Workers; i++ {
		go wCtx.Worker()
	}
	go Stats(c)

	for {
		select {
		case <-c.Done():
			close(in)
			return

		case remote := <-remotes:
			// Enqueue initial job
			atomic.AddInt32(&activeTasks, 1)
			wCtx.queueJob(Job{
				Remote: remote,
				Uri: remote.BaseUri,
				UriStr: remote.BaseUri.String(),
				Fails: 0,
			})
			globalWait.Done()
			// Upload result when ready
			go remote.Watch()
		}
	}
}

func (r *RemoteDir) Watch() {
	// Wait for all jobs on remote to finish
	r.Wait.Wait()
	atomic.AddInt32(&activeTasks, -1)
}

func makeJobBuffer() (chan<- Job, <-chan Job) {
	in  := make(chan Job)
	out := make(chan Job)
	go bufferJobs(in, out)
	return in, out
}

func bufferJobs(in chan Job, out chan Job) {
	var inQueue []Job
	outCh := func() chan Job {
		if len(inQueue) == 0 {
			return nil
		}
		return out
	}
	for len(inQueue) > 0 || in != nil {
		if len(inQueue) == 0 {
			v, ok := <-in
			if !ok {
				in = nil
			} else {
				inQueue = append(inQueue, v)
			}
		} else {
			select {
			case v, ok := <-in:
				if !ok {
					in = nil
				} else {
					inQueue = append(inQueue, v)
				}
			case outCh() <- inQueue[0]:
				inQueue = inQueue[1:]
			}
		}
	}
	close(out)
}
