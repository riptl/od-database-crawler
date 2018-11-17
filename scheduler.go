package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/terorie/od-database-crawler/fasturl"
	"os"
	"path"
	"sync/atomic"
	"time"
)

var activeTasks int32
var totalBuffered int64

func Schedule(c context.Context, remotes <-chan *OD) {
	go Stats(c)

	for remote := range remotes {
		logrus.WithField("url", remote.BaseUri.String()).
			Info("Starting crawler")

		// Collect results
		results := make(chan File)

		// Spawn workers
		remote.WCtx.in, remote.WCtx.out = makeJobBuffer(c)
		for i := 0; i < config.Workers; i++ {
			go remote.WCtx.Worker(results)
		}

		// Enqueue initial job
		atomic.AddInt32(&activeTasks, 1)
		remote.WCtx.queueJob(Job{
			OD:     remote,
			Uri:    remote.BaseUri,
			UriStr: remote.BaseUri.String(),
			Fails:  0,
		})

		// Upload result when ready
		go remote.Watch(results)

		// Sleep if max number of tasks are active
		for atomic.LoadInt32(&activeTasks) > config.Tasks {
			select {
			case <-c.Done():
				return
			case <-time.After(time.Second):
				continue
			}
		}
	}
}

func (r *OD) Watch(results chan File) {
	go r.Task.Collect(results)

	// Wait for all jobs on remote to finish
	r.Wait.Wait()
	close(r.WCtx.in)
	atomic.AddInt32(&activeTasks, -1)

	logrus.WithField("url", r.BaseUri.String()).
		Info("Crawler finished")

	globalWait.Done()

	close(results)
}

func (t *Task) Collect(results chan File) {
	err := t.collect(results)
	if err != nil {
		logrus.WithError(err).
			Error("Failed saving crawl results")
	}
}

func (t *Task) collect(results chan File) error {
	err := os.MkdirAll("crawled", 0755)
	if err != nil { return err }

	f, err := os.OpenFile(
		path.Join("crawled", fmt.Sprintf("%d.json", t.WebsiteId)),
		os.O_CREATE | os.O_WRONLY | os.O_TRUNC,
		0755,
	)
	if err != nil { return err }
	defer f.Close()

	for result := range results {
		result.Path = fasturl.PathUnescape(result.Path)
		result.Name = fasturl.PathUnescape(result.Name)
		resJson, err := json.Marshal(result)
		if err != nil { panic(err) }
		_, err = f.Write(resJson)
		if err != nil { return err }
		_, err = f.Write([]byte{'\n'})
		if err != nil { return err }
	}

	return nil
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
