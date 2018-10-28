package main

import (
	"context"
	"github.com/sirupsen/logrus"
	"net/url"
	"sync/atomic"
	"time"
)

type Job struct {
	Remote *RemoteDir
	Uri url.URL
	UriStr string
	Fails int
}

func Schedule(c context.Context, remotes <-chan *RemoteDir) {
	in, out := makeJobBuffer()
	wCtx := WorkerContext{ in, out }
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
			queueJob(in, Job{
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
}

func Stats(c context.Context) {
	var startedLast uint64 = 0
	ticker := time.NewTicker(time.Second).C
	for {
		select {
		case <-ticker:
			startedNow := atomic.LoadUint64(&totalStarted)
			logrus.WithFields(logrus.Fields{
				"per_second": startedNow - startedLast,
				"done":    atomic.LoadUint64(&totalDone),
				"retries": atomic.LoadUint64(&totalRetries),
				"aborted": atomic.LoadUint64(&totalAborted),
			}).Info("Stats")

			startedLast = startedNow

		case <-c.Done():
			return
		}
	}
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
