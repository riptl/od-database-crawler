package main

import (
	"context"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"net/url"
	"os"
	"strings"
	"time"
)

var app = cli.App {
	Name:         "oddb-go",
	Usage:        "OD-Database Go crawler",
	Version:      "0.1",
	BashComplete: cli.DefaultAppComplete,
	Writer:       os.Stdout,
	Compiled:     buildDate,
	Commands:     []cli.Command{
		{
			Name:      "crawl",
			Usage:     "Crawl a list of URLs",
			ArgsUsage: "[site, site, ...]",
			Action:    cmdCrawler,
		},
	},
}

func init() {
	prepareConfig()
}

func main() {
	app.Run(os.Args)
}

func cmdCrawler(clic *cli.Context) error {
	readConfig()

	if clic.NArg() == 0 {
		cli.ShowCommandHelpAndExit(clic, "crawl", 1)
	}

	args := clic.Args()
	remotes := make([]*OD, len(args))
	for i, arg := range args {
		// https://github.com/golang/go/issues/19779
		if !strings.Contains(arg, "://") {
			arg = "http://" + arg
		}
		u, err := url.Parse(arg)
		if !strings.HasSuffix(u.Path, "/") {
			u.Path += "/"
		}
		if err != nil { return err }
		remotes[i] = &OD{ BaseUri: *u }
	}

	c := context.Background()

	inRemotes := make(chan *OD)
	go Schedule(c, inRemotes)

	for _, remote := range remotes {
		globalWait.Add(1)
		inRemotes <- remote
	}

	// Wait for all jobs to finish
	globalWait.Wait()

	logrus.Info("All dirs processed!")

	return nil
}

var buildDate = time.Date(
	2018, 10, 28,
	17, 10, 0, 0,
	time.UTC)
