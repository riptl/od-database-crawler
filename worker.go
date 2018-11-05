package main

import (
	"github.com/sirupsen/logrus"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

var globalWait sync.WaitGroup

type WorkerContext struct {
	in chan<- Job
	out <-chan Job
	lastRateLimit time.Time
	numRateLimits int
}

func (w WorkerContext) Worker() {
	for job := range w.out {
		w.step(job)
	}
}

func (w WorkerContext) step(job Job) {
	defer w.finishJob(&job)

	var f File

	newJobs, err := DoJob(&job, &f)
	atomic.AddUint64(&totalStarted, 1)
	if err == ErrKnown {
		return
	}

	if err != nil {
		job.Fails++

		if err == ErrForbidden {
			// Don't attempt crawling again
			return
		}

		if job.Fails > config.Retries {
			atomic.AddUint64(&totalAborted, 1)
			logrus.WithField("url", job.UriStr).
				Errorf("Giving up after %d fails", job.Fails)
		} else {
			atomic.AddUint64(&totalRetries, 1)
			if err == ErrRateLimit {
				w.lastRateLimit = time.Now()
				w.numRateLimits++
			}
			w.queueJob(job)
		}
		return
	}

	atomic.AddUint64(&totalDone, 1)
	for _, job := range newJobs {
		w.queueJob(job)
	}

	job.OD.Files = append(job.OD.Files, f)
}

func DoJob(job *Job, f *File) (newJobs []Job, err error) {
	if len(job.Uri.Path) == 0 { return }
	if job.Uri.Path[len(job.Uri.Path)-1] == '/' {
		// Load directory
		links, err := GetDir(job, f)
		if err != nil {
			logrus.WithError(err).
				WithField("url", job.UriStr).
				Error("Failed getting dir")
			return nil, err
		}

		// Hash directory
		hash := f.HashDir(links)

		// Skip symlinked dirs
		if job.OD.LoadOrStoreKey(&hash) {
			return nil, ErrKnown
		}

		for _, link := range links {
			// Skip already queued links
			//if _, old := job.OD.Scanned.LoadOrStore(link, true); old {
			//	continue
			//}
			job.OD.Wait.Add(1)
			newJobs = append(newJobs, Job{
				OD:     job.OD,
				Uri:    link,
				UriStr: link.String(),
				Fails:  0,
			})
		}
		if config.Verbose {
			logrus.WithFields(logrus.Fields{
				"url":   job.UriStr,
				"files": len(links),
			}).Debug("Listed")
		}
	} else {
		// Load file
		err := GetFile(job.Uri, f)
		if err != nil {
			logrus.WithError(err).
				WithField("url", job.UriStr).
				Error("Failed getting file")
			return nil, err
		}
	}
	return
}

func (w WorkerContext) queueJob(job Job) {
	job.OD.Wait.Add(1)
	globalWait.Add(1)

	if w.numRateLimits > 0 {
		if time.Since(w.lastRateLimit) > 5 * time.Second {
			w.numRateLimits = 0
		} else {
			time.Sleep(time.Duration(math.Sqrt(float64(50 * w.numRateLimits))) *
				100 * time.Millisecond)
			w.in <- job
		}
	} else {
		w.in <- job
	}
}

func (w WorkerContext) finishJob(job *Job) {
	job.OD.Wait.Done()
	globalWait.Done()
}
