package main

import (
	"context"
	"github.com/sirupsen/logrus"
	"github.com/terorie/oddb-go/fasturl"
	"github.com/urfave/cli"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"sync/atomic"
	"time"
)

var app = cli.App {
	Name:         "oddb-go",
	Usage:        "OD-Database Go crawler",
	Version:      "0.2",
	BashComplete: cli.DefaultAppComplete,
	Writer:       os.Stdout,
	Compiled:     buildDate,
	Action:       cmdBase,
}

func init() {
	prepareConfig()
}

func main() {
	go func() {
		log.Println(http.ListenAndServe("localhost:42069", nil))
	}()
	app.Run(os.Args)
}

func cmdBase(clic *cli.Context) error {
	readConfig()

	// TODO Graceful shutdown
	appCtx := context.Background()
	forceCtx := context.Background()

	inRemotes := make(chan *OD)
	go Schedule(forceCtx, inRemotes)

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-appCtx.Done():
			return nil
		case <-ticker.C:
			t, err := FetchTask()
			if err != nil {
				logrus.WithError(err).
					Error("Failed getting new task")
				time.Sleep(30 * time.Second)
				continue
			}
			if t == nil {
				// No new task
				if atomic.LoadInt32(&activeTasks) == 0 {
					logrus.Info("Waiting …")
				}
				continue
			}

			var baseUri fasturl.URL
			err = baseUri.Parse(t.Url)
			if err != nil {
				logrus.WithError(err).
					Error("Failed getting new task")
				time.Sleep(30 * time.Second)
				continue
			}
			inRemotes <- &OD {
				Task: t,
				BaseUri: baseUri,
			}
		}
	}

	// Wait for all jobs to finish
	globalWait.Wait()

	return nil
}

var buildDate = time.Date(
	2018, 11, 15,
	23, 24, 0, 0,
	time.UTC)
