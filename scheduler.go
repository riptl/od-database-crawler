package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/terorie/od-database-crawler/fasturl"
	"os"
	"path"
	"sync"
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

func ScheduleTask(remotes chan<- *OD, t *Task, u *fasturl.URL) {
	globalWait.Add(1)
	now := time.Now()
	od := &OD {
		Task: *t,
		BaseUri: *u,
		Result: TaskResult {
			WebsiteId: t.WebsiteId,
			StartTime: now,
			StartTimeUnix: now.Unix(),
		},
	}
	remotes <- od
}

func (o *OD) Watch(results chan File) {
	filePath := path.Join("crawled", fmt.Sprintf("%d.json", o.Task.WebsiteId))

	// Open crawl results file
	// TODO Print errors
	err := os.MkdirAll("crawled", 0755)
	if err != nil { return }
	f, err := os.OpenFile(
		filePath,
		os.O_CREATE | os.O_RDWR | os.O_TRUNC,
		0644,
	)

	if err != nil { return }
	defer f.Close()
	defer os.Remove(filePath)

	// Wait for the file to be fully written
	var fileLock sync.Mutex
	fileLock.Lock()

	go o.Task.Collect(results, f, &fileLock)

	// Wait for all jobs on remote to finish
	o.Wait.Wait()
	close(o.WCtx.in)
	atomic.AddInt32(&activeTasks, -1)

	// Log finish

	logrus.
		WithField("url", o.BaseUri.String()).
		WithField("duration", time.Since(o.Result.StartTime)).
		Info("Crawler finished")

	// Set status code
	now := time.Now()
	o.Result.EndTimeUnix = now.Unix()
	fileCount := atomic.LoadUint64(&o.Result.FileCount)
	if fileCount == 0 {
		errorCount := atomic.LoadUint64(&o.Result.ErrorCount)
		if errorCount == 0 {
			o.Result.StatusCode = "empty"
		} else {
			o.Result.StatusCode = "directory listing failed"
		}
	} else {
		o.Result.StatusCode = "success"
	}

	// Shut down Collect()
	close(results)

	// Wait for results to sync to file
	fileLock.Lock()
	fileLock.Unlock()

	// Upload results
	err = PushResult(&o.Result, f)
	if err != nil {
		logrus.WithError(err).
			Error("Failed uploading result")
	}

	// Mark job as completely done
	globalWait.Done()
}

func (t *Task) Collect(results chan File, f *os.File, done *sync.Mutex) {
	err := t.collect(results, f)
	if err != nil {
		logrus.WithError(err).
			Error("Failed saving crawl results")
	}
	done.Unlock()
}

func (t *Task) collect(results chan File, f *os.File) error {
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
