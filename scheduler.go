package main

import (
	"context"
	"sync/atomic"
	"time"
)

var activeTasks int32

func Schedule(c context.Context, remotes <-chan *OD) {
	go Stats(c)

	for {
		select {
		case <-c.Done():
			return

		case remote := <-remotes:
			for atomic.LoadInt32(&activeTasks) > config.Tasks {
				select {
				case <-time.After(time.Second):
					break
				case <-c.Done():
					return
				}
			}

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
					inQueue = append(inQueue, v)
				}
			case outCh() <- inQueue[0]:
				inQueue = inQueue[1:]
			case <-c.Done():
				return
			}
		}
	}
}
