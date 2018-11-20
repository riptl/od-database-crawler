package main

import (
	"bufio"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"io"
	"os"
	"strings"
	"time"
)

var config struct {
	ServerUrl  string
	Token      string
	ServerTimeout time.Duration
	Recheck    time.Duration
	ChunkSize  int64
	Retries    int
	Workers    int
	UserAgent  string
	Tasks      int32
	CrawlStats time.Duration
	AllocStats time.Duration
	Verbose    bool
	PrintHTTP  bool
}

const (
	ConfServerUrl  = "server.url"
	ConfToken      = "server.token"
	ConfServerTimeout = "server.timeout"
	ConfRecheck    = "server.recheck"
	ConfChunkSize  = "server.upload_chunk"
	ConfTasks      = "crawl.tasks"
	ConfRetries    = "crawl.retries"
	ConfWorkers    = "crawl.connections"
	ConfUserAgent  = "crawl.user-agent"
	ConfDialTimeout = "crawl.dial_timeout"
	ConfTimeout    = "crawl.timeout"
	ConfCrawlStats = "output.crawl_stats"
	ConfAllocStats = "output.resource_stats"
	ConfVerbose    = "output.verbose"
	ConfPrintHTTP  = "output.http"
	ConfLogFile    = "output.log"
)

func prepareConfig() {
	viper.SetDefault(ConfRetries, 5)
	viper.SetDefault(ConfWorkers, 2)
	viper.SetDefault(ConfTasks, 3)
	viper.SetDefault(ConfUserAgent, "")
	viper.SetDefault(ConfDialTimeout, 10 * time.Second)
	viper.SetDefault(ConfTimeout, 30 * time.Second)
	viper.SetDefault(ConfCrawlStats, 3 * time.Second)
	viper.SetDefault(ConfAllocStats, 0)
	viper.SetDefault(ConfVerbose, false)
	viper.SetDefault(ConfPrintHTTP, false)
	viper.SetDefault(ConfLogFile, "")
	viper.SetDefault(ConfRecheck, 3 * time.Second)
	viper.SetDefault(ConfChunkSize, "1 MB")
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
	if config.ServerUrl == "" {
		configMissing(ConfServerUrl)
	}
	config.ServerUrl = strings.TrimRight(config.ServerUrl, "/")

	config.Token = viper.GetString(ConfToken)
	if config.Token == "" {
		configMissing(ConfToken)
	}

	config.ServerTimeout = viper.GetDuration(ConfServerTimeout)

	config.Recheck = viper.GetDuration(ConfRecheck)

	config.ChunkSize = int64(viper.GetSizeInBytes(ConfChunkSize))
	if config.ChunkSize < 100 {
		configOOB(ConfChunkSize, config.ChunkSize)
	}

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

	config.UserAgent = viper.GetString(ConfUserAgent)

	setDialTimeout(viper.GetDuration(ConfDialTimeout))

	setTimeout(viper.GetDuration(ConfTimeout))

	config.CrawlStats = viper.GetDuration(ConfCrawlStats)

	config.AllocStats = viper.GetDuration(ConfAllocStats)

	config.Verbose = viper.GetBool(ConfVerbose)
	if config.Verbose {
		logrus.SetLevel(logrus.DebugLevel)
	}

	if filePath := viper.GetString(ConfLogFile); filePath != "" {
		f, err := os.OpenFile(filePath, os.O_CREATE | os.O_WRONLY | os.O_APPEND, 0644)
		bufWriter := bufio.NewWriter(f)
		if err != nil { panic(err) }
		exitHooks.Add(func() {
			bufWriter.Flush()
			f.Close()
		})
		logrus.SetOutput(io.MultiWriter(os.Stdout, bufWriter))
	}

	config.PrintHTTP = viper.GetBool(ConfPrintHTTP)
}

func configMissing(key string) {
	fmt.Fprintf(os.Stderr, "config: %s not set!\n", key)
	os.Exit(1)
}

func configOOB(key string, v interface{}) {
	fmt.Fprintf(os.Stderr, "config: illegal value %v for key %s!\n", v, key)
	os.Exit(1)
}
