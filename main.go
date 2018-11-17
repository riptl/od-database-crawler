package main

import (
	"context"
	"github.com/sirupsen/logrus"
	"github.com/terorie/od-database-crawler/fasturl"
	"github.com/urfave/cli"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

var app = cli.App {
	Name:         "od-database-crawler",
	Usage:        "OD-Database Go crawler",
	Version:      "0.2",
	BashComplete: cli.DefaultAppComplete,
	Writer:       os.Stdout,
	Compiled:     buildDate,
	Action:       cmdBase,
	Commands:     []cli.Command{
		{
			Name:      "crawl",
			Usage:     "Crawl a list of URLs",
			ArgsUsage: "<site>",
			Action:    cmdCrawler,
		},
	},
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
			ScheduleTask(inRemotes, t, &baseUri)
		}
	}

	return nil
}

func cmdCrawler(clic *cli.Context) error {
	readConfig()

	if clic.NArg() != 1 {
		cli.ShowCommandHelpAndExit(clic, "crawl", 1)
	}

	arg := clic.Args()[0]
	// https://github.com/golang/go/issues/19779
	if !strings.Contains(arg, "://") {
		arg = "http://" + arg
	}
	var u fasturl.URL
	err := u.Parse(arg)
	if !strings.HasSuffix(u.Path, "/") {
		u.Path += "/"
	}
	if err != nil { return err }

	// TODO Graceful shutdown
	forceCtx := context.Background()

	inRemotes := make(chan *OD)
	go Schedule(forceCtx, inRemotes)

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	task := Task {
		WebsiteId: 0,
		Url: u.String(),
	}
	ScheduleTask(inRemotes, &task, &u)

	// Wait for all jobs to finish
	globalWait.Wait()

	return nil
}

var buildDate = time.Date(
	2018, 11, 15,
	23, 24, 0, 0,
	time.UTC)
