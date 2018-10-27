package main

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/sirupsen/logrus"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	nConns = 100
)

var client = http.DefaultClient
var wait sync.WaitGroup

var visited int64

var in chan<- url.URL
var out <-chan url.URL

func main() {
	if len(os.Args) != 2 {
		println("Usage: ./crawl <url>")
		os.Exit(1)
	}

	in, out = makeInfinite()
	
	go func() {
		var visitedLast int64 = 0
		for range time.NewTicker(time.Second).C {
			visitedNow := atomic.LoadInt64(&visited)
			logrus.
				WithField("per_second", visitedNow - visitedLast).
				WithField("total", visitedNow).
				Info("Tick")
			visitedLast = visitedNow
		}
	}()

	base, _ := url.Parse(os.Args[1])
	in <- *base
	wait.Add(nConns)
	for i := 0; i < nConns; i++ {
		go worker()
	}
	wait.Wait()
}

func worker() {
	for u := range out {
		if strings.HasSuffix(u.Path, "/") {
			// Dir
			links := listDir(u)
			for _, sub := range links {
				subrefi, err := url.Parse(sub)
				subref := *subrefi
				if err != nil { continue }

				in <- *u.ResolveReference(&subref)
			}
		} else {
			// File
			// TODO check file
		}
	}
	wait.Done()
}

func listDir(u url.URL) (links []string) {
	//logrus.Infof("Visiting %s", u)
	atomic.AddInt64(&visited, 1)

	res, err := client.Get(u.String())
	if err != nil {
		logrus.Error(err)
		return
	}
	defer res.Body.Close()

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		logrus.Error(err)
		return
	}

	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		text := s.Text()

		if href == "." {
			return
		}

		for _, entry := range blackList {
			if strings.Contains(href, entry) {
				return
			}
		}

		for _, entry := range fileNameBlackList {
			if strings.Contains(text, entry) {
				return
			}
		}

		links = append(links, href)
	})

	return
}

func makeInfinite() (chan<- url.URL, <-chan url.URL) {
	in := make(chan url.URL)
	out := make(chan url.URL)
	// Set up in and out queues
	go func() {
		var inQueue []url.URL
		outCh := func() chan url.URL {
			if len(inQueue) == 0 {
				return nil
			}
			return out
		}
		for len(inQueue) > 0 || in != nil {
			if len(inQueue) == 0 {
				v, ok := <-in
				if !ok {
					in = nil
				} else {
					inQueue = append(inQueue, v)
				}
			} else {
				select {
				case v, ok := <-in:
					if !ok {
						in = nil
					} else {
						inQueue = append(inQueue, v)
					}
				case outCh() <- inQueue[0]:
					inQueue = inQueue[1:]
				}
			}
		}
		close(out)
	}()
	return in, out
}

var blackList = [...]string {
	"?C=N&O=D",
	"?C=M&O=A",
	"?C=S&O=A",
	"?C=D&O=A",
	"?C=N;O=D",
	"?C=M;O=A",
	"?C=M&O=D",
	"?C=S;O=A",
	"?C=S&O=D",
	"?C=D;O=A",
	"?MA",
	"?SA",
	"?DA",
	"?ND",
	"?C=N&O=A",
	"?C=N&O=A",
	"?M=A",
	"?N=D",
	"?S=A",
	"?D=A",
}

var fileNameBlackList = [...]string {
	"Parent Directory",
	" Parent Directory",
	"../",
}

