package main

import (
	"bufio"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"io"
	"os"
	"strings"
	"time"
)

var config struct {
	TrackerUrl     string
	TrackerProject string
	Token          string
	ServerTimeout  time.Duration
	Recheck        time.Duration
	ChunkSize      int64
	Retries        int
	Workers        int
	UserAgent      string
	Tasks          int32
	Verbose        bool
	PrintHTTP      bool
	JobBufferSize  int
}

var onlineMode bool

const (
	ConfTrackerUrl          = "server.url"
	ConfTrackerProject      = "server.project"
	ConfServerTimeout       = "server.timeout"
	ConfRecheck             = "server.recheck"
	ConfCooldown            = "server.cooldown"
	ConfChunkSize           = "server.upload_chunk"
	ConfUploadRetries       = "server.upload_retries"
	ConfUploadRetryInterval = "server.upload_retry_interval"

	ConfTasks         = "crawl.tasks"
	ConfRetries       = "crawl.retries"
	ConfWorkers       = "crawl.connections"
	ConfUserAgent     = "crawl.user-agent"
	ConfDialTimeout   = "crawl.dial_timeout"
	ConfTimeout       = "crawl.timeout"
	ConfJobBufferSize = "crawl.job_buffer"

	ConfCrawlStats = "output.crawl_stats"
	ConfAllocStats = "output.resource_stats"
	ConfVerbose    = "output.verbose"
	ConfPrintHTTP  = "output.http"
	ConfLogFile    = "output.log"
)

func prepareConfig() {
	pf := rootCmd.PersistentFlags()

	pf.SortFlags = false
	pf.StringVar(&configFile, "config", "", "Config file")
	configFile = os.Getenv("OD_CONFIG")

	pf.String(ConfTrackerUrl, "http://tt.the-eye.eu/api", "task_tracker api URL")

	pf.String(ConfTrackerProject, "3", "task_tracker project id")

	pf.Duration(ConfServerTimeout, 60*time.Second, "OD-DB request timeout")

	pf.Duration(ConfRecheck, 1*time.Second, "OD-DB: Poll interval for new jobs")

	pf.Duration(ConfCooldown, 30*time.Second, "OD-DB: Time to wait after a server-side error")

	pf.String(ConfChunkSize, "1 MB", "OD-DB: Result upload chunk size")

	pf.Uint(ConfUploadRetries, 10, "OD-DB: Max upload retries")

	pf.Duration(ConfUploadRetryInterval, 30*time.Second, "OD-DB: Time to wait between upload retries")

	pf.Uint(ConfTasks, 100, "Crawler: Max concurrent tasks")

	pf.Uint(ConfWorkers, 4, "Crawler: Connections per server")

	pf.Uint(ConfRetries, 5, "Crawler: Request retries")

	pf.Duration(ConfDialTimeout, 10*time.Second, "Crawler: Handshake timeout")

	pf.Duration(ConfTimeout, 30*time.Second, "Crawler: Request timeout")

	pf.String(ConfUserAgent, "Mozilla/5.0 (X11; od-database-crawler) Gecko/20100101 Firefox/52.0", "Crawler: User-Agent")

	pf.Uint(ConfJobBufferSize, 5000, "Crawler: Task queue cache size")

	pf.Duration(ConfCrawlStats, time.Second, "Log: Crawl stats interval")

	pf.Duration(ConfAllocStats, 10*time.Second, "Log: Resource stats interval")

	pf.Bool(ConfVerbose, false, "Log: Print every listed dir")

	pf.Bool(ConfPrintHTTP, false, "Log: Print HTTP client errors")

	pf.String(ConfLogFile, "crawler.log", "Log file")

	// Bind all flags to Viper
	pf.VisitAll(func(flag *pflag.Flag) {
		s := flag.Name
		s = strings.TrimLeft(s, "-")

		if err := viper.BindPFlag(s, flag); err != nil {
			panic(err)
		}
		var envKey string
		envKey = strings.Replace(s, ".", "_", -1)
		envKey = strings.ToUpper(envKey)
		envKey = "OD_" + envKey
		if err := viper.BindEnv(s, envKey); err != nil {
			panic(err)
		}
	})
}

func readConfig() {
	// If config.yml in working dir, use it
	if configFile == "" {
		_, err := os.Stat("config.yml")
		if err == nil {
			configFile = "config.yml"
		}
	}

	if configFile != "" {
		confF, err := os.Open(configFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		defer confF.Close()

		viper.SetConfigType("yml")
		err = viper.ReadConfig(confF)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	if onlineMode {
		config.TrackerUrl = viper.GetString(ConfTrackerUrl)
		if config.TrackerUrl == "" {
			configMissing(ConfTrackerUrl)
		}
		config.TrackerUrl = strings.TrimRight(config.TrackerUrl, "/")
	}
	config.TrackerProject = viper.GetString(ConfTrackerProject)

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

	config.JobBufferSize = viper.GetInt(ConfJobBufferSize)

	config.Verbose = viper.GetBool(ConfVerbose)
	if config.Verbose {
		logrus.SetLevel(logrus.DebugLevel)
	}

	if filePath := viper.GetString(ConfLogFile); filePath != "" {
		f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		bufWriter := bufio.NewWriter(f)
		if err != nil {
			panic(err)
		}
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
