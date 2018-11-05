package main

import (
	"context"
	"github.com/sirupsen/logrus"
	"sync/atomic"
)

var activeTasks int32
var totalBuffered int64

func Schedule(c context.Context, remotes <-chan *OD) {
	go Stats(c)

	for {
		select {
		case <-c.Done():
			return

		case remote := <-remotes:
			logrus.WithField("url", remote.BaseUri.String()).
				Info("Starting crawler")

			// Spawn workers
			remote.WCtx.in, remote.WCtx.out = makeJobBuffer(c)
			for i := 0; i < config.Workers; i++ {
				go remote.WCtx.Worker()
			}

			// Enqueue initial job
			atomic.AddInt32(&activeTasks, 1)
			remote.WCtx.queueJob(Job{
				OD:     remote,
				Uri:    remote.BaseUri,
				UriStr: remote.BaseUri.String(),
				Fails:  0,
			})
			globalWait.Done()

			// Upload result when ready
			go remote.Watch()
		}
	}
}

func (r *OD) Watch() {
	// Wait for all jobs on remote to finish
	r.Wait.Wait()
	close(r.WCtx.in)
	atomic.AddInt32(&activeTasks, -1)

	logrus.WithField("url", r.BaseUri.String()).
		Info("Crawler finished")
}

func makeJobBuffer(c context.Context) (chan<- Job, <-chan Job) {
	in  := make(chan Job)
	out := make(chan Job)
	go bufferJobs(c, in, out)
	return in, out
}

func bufferJobs(c context.Context, in chan Job, out chan Job) {
	defer close(out)
	var inQueue []Job
	outCh := func() chan Job {
		if len(inQueue) == 0 {
			return nil
		}
		return out
	}
	for len(inQueue) > 0 || in != nil {
		if len(inQueue) == 0 {
			select {
			case v, ok := <-in:
				if !ok {
					in = nil
				} else {
					atomic.AddInt64(&totalBuffered, 1)
					inQueue = append(inQueue, v)
				}
			case <-c.Done():
				return
			}
		} else {
			select {
			case v, ok := <-in:
				if !ok {
					in = nil
				} else {
					atomic.AddInt64(&totalBuffered, 1)
					inQueue = append(inQueue, v)
				}
			case outCh() <- inQueue[0]:
				atomic.AddInt64(&totalBuffered, -1)
				inQueue = inQueue[1:]
			case <-c.Done():
				return
			}
		}
	}
}
