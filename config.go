package main

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
	"time"
)

var config struct {
	ServerUrl  string
	Token      string
	Retries    int
	Workers    int
	Tasks      int32
	CrawlStats time.Duration
	AllocStats time.Duration
	Verbose    bool
}

const (
	ConfServerUrl  = "server.url"
	ConfToken      = "server.token"
	ConfTasks      = "crawl.tasks"
	ConfRetries    = "crawl.retries"
	ConfWorkers    = "crawl.connections"
	ConfCrawlStats = "output.crawl_stats"
	ConfAllocStats = "output.resource_stats"
	ConfVerbose    = "output.verbose"
)

func prepareConfig() {
	viper.SetDefault(ConfRetries, 5)
	viper.SetDefault(ConfWorkers, 2)
	viper.SetDefault(ConfTasks, 3)
	viper.SetDefault(ConfCrawlStats, 3 * time.Second)
	viper.SetDefault(ConfAllocStats, 0)
	viper.SetDefault(ConfVerbose, false)
}

func readConfig() {
	viper.AddConfigPath(".")
	viper.SetConfigName("config")
	err := viper.ReadInConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	config.ServerUrl = viper.GetString(ConfServerUrl)
	//if config.ServerUrl == "" {
	//	configMissing(ConfServerUrl)
	//}

	config.Token = viper.GetString(ConfToken)
	//if config.Token == "" {
	//	configMissing(ConfToken)
	//}

	config.Retries = viper.GetInt(ConfRetries)
	if config.Retries < 0 {
		config.Retries = 1 << 31
	}

	config.Workers = viper.GetInt(ConfWorkers)
	if config.Workers <= 0 {
		configOOB(ConfWorkers, config.Workers)
	}

	config.Tasks = viper.GetInt32(ConfTasks)
	if config.Tasks <= 0 {
		configOOB(ConfTasks, int(config.Tasks))
	}

	config.CrawlStats = viper.GetDuration(ConfCrawlStats)

	config.AllocStats = viper.GetDuration(ConfAllocStats)

	config.Verbose = viper.GetBool(ConfVerbose)
	if config.Verbose {
		logrus.SetLevel(logrus.DebugLevel)
	}
}

func configMissing(key string) {
	fmt.Fprintf(os.Stderr, "config: %s not set!\n", key)
	os.Exit(1)
}

func configOOB(key string, v int) {
	fmt.Fprintf(os.Stderr, "config: illegal value %d for %key!\n", v, key)
	os.Exit(1)
}
