package main

import (
	"context"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/terorie/od-database-crawler/fasturl"
	"github.com/urfave/cli"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

var configFile string

var app = cli.App {
	Name:         "od-database-crawler",
	Usage:        "OD-Database Go crawler",
	Version:      "1.0.2",
	BashComplete: cli.DefaultAppComplete,
	Writer:       os.Stdout,
	Action:       cmdBase,
	Commands:     []cli.Command {
		{
			Name:      "crawl",
			Usage:     "Crawl a list of URLs",
			ArgsUsage: "<site>",
			Action:    cmdCrawler,
		},
	},
	Flags: []cli.Flag {
		cli.StringFlag {
			Name: "config",
			EnvVar: "CONFIG",
			Destination: &configFile,
		},
	},
	Before: func(i *cli.Context) error {
		if configFile != "" {
			viper.SetConfigFile(configFile)
		}
		return nil
	},
	After: func(i *cli.Context) error {
		exitHooks.Execute()
		return nil
	},
}

var exitHooks Hooks

func init() {
	prepareConfig()
}

func main() {
	if err := os.MkdirAll("crawled", 0755);
		err != nil { panic(err) }

	if err := os.MkdirAll("queue", 0755);
		err != nil { panic(err) }

	readConfig()
	app.Run(os.Args)
}

func cmdBase(_ *cli.Context) error {
	// TODO Graceful shutdown
	appCtx := context.Background()
	forceCtx := context.Background()

	inRemotes := make(chan *OD)
	go Schedule(forceCtx, inRemotes)

	ticker := time.NewTicker(config.Recheck)
	defer ticker.Stop()
	for {
		select {
		case <-appCtx.Done():
			return nil
		case <-ticker.C:
			t, err := FetchTask()
			if err != nil {
				logrus.WithError(err).
					Error("Failed to get new task")
				time.Sleep(30 * time.Second)
				continue
			}
			if t == nil {
				// No new task
				if atomic.LoadInt32(&numActiveTasks) == 0 {
					logrus.Info("Waiting …")
				}
				continue
			}

			var baseUri fasturl.URL
			err = baseUri.Parse(t.Url)
			if urlErr, ok := err.(*fasturl.Error); ok && urlErr.Err == fasturl.ErrUnknownScheme {
				// Not an error
				err = nil

				// Give back task
				//err2 := CancelTask(t.WebsiteId)
				//if err2 != nil {
				//	logrus.Error(err2)
				//}

				continue
			} else if err != nil {
				logrus.WithError(err).
					Error("Failed to get new task")
				time.Sleep(30 * time.Second)
				continue
			}
			ScheduleTask(inRemotes, t, &baseUri)
		}
	}

	return nil
}

func cmdCrawler(clic *cli.Context) error {
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
