package main

import (
	"context"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"math"
	"runtime"
	"sync/atomic"
	"time"
)

var totalStarted uint64
var totalDone uint64
var totalRetries uint64
var totalAborted uint64

func Stats(c context.Context) {
	var startedLast uint64 = 0
	var crawlTicker <-chan time.Time
	var allocTicker <-chan time.Time

	crawlInterval := viper.GetDuration(ConfCrawlStats)
	allocInterval := viper.GetDuration(ConfAllocStats)

	if crawlInterval != 0 {
		crawlTicker = time.Tick(crawlInterval)
	}
	if allocInterval != 0 {
		allocTicker = time.Tick(allocInterval)
	}

	for {
		select {
		case <-crawlTicker:
			startedNow := atomic.LoadUint64(&totalStarted)

			perSecond := float64(startedNow - startedLast) /
				crawlInterval.Seconds()

			// Round to .5
			perSecond *= 2
			perSecond = math.Round(perSecond)
			perSecond /= 2

			if perSecond <= 0 {
				continue
			}

			logrus.WithFields(logrus.Fields{
				"per_second": perSecond,
				"done":    atomic.LoadUint64(&totalDone),
				"retries": atomic.LoadUint64(&totalRetries),
				"aborted": atomic.LoadUint64(&totalAborted),
			}).Info("Crawl Stats")

			startedLast = startedNow

		case <-allocTicker:
			var mem runtime.MemStats
			runtime.ReadMemStats(&mem)

			logrus.WithFields(logrus.Fields{
				"queue_count": atomic.LoadInt64(&totalQueued),
				"heap": FormatByteCount(mem.Alloc),
				"objects": mem.HeapObjects,
				"num_gc": mem.NumGC,
			}).Info("Resource Stats")

		case <-c.Done():
			return
		}
	}
}
