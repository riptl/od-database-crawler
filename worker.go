package main

import (
	"github.com/beeker1121/goque"
	"github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
	"math"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var globalWait sync.WaitGroup

type WorkerContext struct {
	OD *OD
	Queue *BufferedQueue
	lastRateLimit time.Time
	numRateLimits int
}

func (w *WorkerContext) Worker(results chan<- File) {
	for {
		job, err := w.Queue.Dequeue()
		switch err {
		case goque.ErrEmpty:
			time.Sleep(500 * time.Millisecond)
			continue

		case goque.ErrDBClosed:
			return

		case nil:
			w.step(results, job)

		default:
			panic(err)
		}
	}
}

func (w *WorkerContext) step(results chan<- File, job Job) {
	defer w.finishJob()

	var f File

	newJobs, err := w.DoJob(&job, &f)
	atomic.AddUint64(&totalStarted, 1)
	if err == ErrKnown {
		return
	}

	if err != nil {
		job.Fails++

		if httpErr, ok := err.(*HttpError); ok {
			switch httpErr.code {
			case fasthttp.StatusTooManyRequests:
				err = ErrRateLimit
			default:
				// Don't retry HTTP error codes
				return
			}
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

	if !f.IsDir {
		results <- f
	}
}

func (w *WorkerContext) DoJob(job *Job, f *File) (newJobs []Job, err error) {
	if len(job.Uri.Path) == 0 { return }
	if job.Uri.Path[len(job.Uri.Path)-1] == '/' {
		// Load directory
		links, err := GetDir(job, f)
		if err != nil {
			if !isErrSilent(err) {
				logrus.WithError(err).
					WithField("url", job.UriStr).
					Error("Failed to crawl dir")
			}
			return nil, err
		}

		// Hash directory
		hash := f.HashDir(links)

		// Skip symlinked dirs
		if w.OD.LoadOrStoreKey(&hash) {
			return nil, ErrKnown
		}

		// Sort by path
		sort.Slice(links, func(i, j int) bool {
			return strings.Compare(links[i].Path, links[j].Path) < 0
		})

		var newJobCount int
		var lastLink string
		for _, link := range links {
			uriStr := link.String()

			// Ignore dupes
			if uriStr == lastLink {
				continue
			}
			lastLink = uriStr

			newJobs = append(newJobs, Job{
				Uri:    link,
				UriStr: uriStr,
				Fails:  0,
			})

			newJobCount++
		}
		if config.Verbose {
			logrus.WithFields(logrus.Fields{
				"url":   job.UriStr,
				"files": newJobCount,
			}).Debug("Listed")
		}
	} else {
		// Load file
		err := GetFile(job.Uri, f)
		if err != nil {
			if !isErrSilent(err) {
				logrus.WithError(err).
					WithField("url", job.UriStr).
					Error("Failed to crawl file")
			}
			return nil, err
		}
		atomic.AddUint64(&w.OD.Result.FileCount, 1)
	}
	return
}

func (w *WorkerContext) queueJob(job Job) {
	w.OD.Wait.Add(1)

	if w.numRateLimits > 0 {
		if time.Since(w.lastRateLimit) > 5 * time.Second {
			w.numRateLimits = 0
		} else {
			time.Sleep(time.Duration(math.Sqrt(float64(50 * w.numRateLimits))) *
				100 * time.Millisecond)
		}
	}

	if err := w.Queue.Enqueue(&job); err != nil {
		panic(err)
	}
}

func (w *WorkerContext) finishJob() {
	w.OD.Wait.Done()
}

func isErrSilent(err error) bool {
	if !config.PrintHTTP {
		if _, ok := err.(*HttpError); ok {
			return true
		}
	}
	return false
}
