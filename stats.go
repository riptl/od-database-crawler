package main

import (
	"context"
	"github.com/sirupsen/logrus"
	"math"
	"sync/atomic"
	"time"
)

var totalStarted uint64
var totalDone uint64
var totalRetries uint64
var totalAborted uint64

type File struct {
	Name string `json:"name"`
	Size int64 `json:"size"`
	MTime time.Time `json:"mtime"`
	Path string `json:"path"`
	IsDir bool `json:"-"`
}

func Stats(c context.Context) {
	var startedLast uint64 = 0
	ticker := time.NewTicker(config.StatsInterval).C
	for {
		select {
		case <-ticker:
			startedNow := atomic.LoadUint64(&totalStarted)

			perSecond := float64(startedNow - startedLast) /
				config.StatsInterval.Seconds()

			// Round to .5
			perSecond *= 2
			perSecond = math.Round(perSecond)
			perSecond /= 2

			logrus.WithFields(logrus.Fields{
				"per_second": perSecond,
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
