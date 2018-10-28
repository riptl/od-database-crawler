package main

import (
	"github.com/sirupsen/logrus"
	"strings"
	"sync"
	"sync/atomic"
)

var globalWait sync.WaitGroup

type WorkerContext struct {
	in chan<- Job
	out <-chan Job
}

func (w WorkerContext) Worker() {
	for job := range w.out {
		w.step(job)
	}
}

func (w WorkerContext) step(job Job) {
	defer finishJob(&job)

	var f File

	newJobs, err := DoJob(&job, &f)
	atomic.AddUint64(&totalStarted, 1)

	if err != nil {
		job.Fails++

		logrus.WithFields(logrus.Fields{
			"error": err.Error(),
			"url": job.UriStr,
		}).Warningf("Crawl error: %s", err)

		if job.Fails > config.Retries {
			atomic.AddUint64(&totalAborted, 1)
			logrus.WithField("url", job.UriStr).
				Errorf("Giving up after %d fails", job.Fails)
		} else {
			atomic.AddUint64(&totalRetries, 1)
			queueJob(w.in, job)
		}
		return
	}

	atomic.AddUint64(&totalDone, 1)
	for _, job := range newJobs {
		queueJob(w.in, job)
	}

	job.Remote.Files = append(job.Remote.Files, f)
}

func DoJob(job *Job, f *File) (newJobs []Job, err error) {
	// File
	if strings.HasSuffix(job.Uri.Path, "/") {
		// Dir
		links, err := GetDir(job.Uri, f)
		if err != nil {
			logrus.WithError(err).
				WithField("url", job.Uri.String()).
				Error("Failed getting dir")
			return nil, err
		}
		for _, link := range links {
			job.Remote.Wait.Add(1)
			newJobs = append(newJobs, Job{
				Remote: job.Remote,
				Uri:    link,
				UriStr: link.String(),
				Fails:  0,
			})
		}
	} else {
		err := GetFile(job.Uri, f)
		if err != nil {
			logrus.WithError(err).
				WithField("url", job.Uri.String()).
				Error("Failed getting file")
			return nil, err
		}
	}
	return
}

func queueJob(in chan<- Job, job Job) {
	job.Remote.Wait.Add(1)
	globalWait.Add(1)
	in <- job
}

func finishJob(job *Job) {
	job.Remote.Wait.Done()
	globalWait.Done()
}
