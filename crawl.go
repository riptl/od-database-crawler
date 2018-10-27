package main

import (
	"bytes"
	"errors"
	"github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	nConns = 100
)

var client = fasthttp.Client{}
var wait sync.WaitGroup

var visited int64

var in chan<- url.URL
var out <-chan url.URL

type File struct {
	Name string `json:"name"`
	Size int64 `json:"size"`
	MTime time.Time `json:"mtime"`
	Path string `json:"path"`
	IsDir bool `json:"-"`
}

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
				// TODO Print errors
				if err != nil { continue }
				abs := *u.ResolveReference(&subref)
				in <- abs
			}
		} else {
			// File
			var fil File
			err := fileInfo(u, &fil)
			// TODO Print errors
			if err != nil { continue }
		}
	}
	wait.Done()
}

func listDir(u url.URL) (links []string) {
	req := fasthttp.AcquireRequest()
	req.SetRequestURI(u.String())

	res := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(res)

	err := client.Do(req, res)
	fasthttp.ReleaseRequest(req)

	if err != nil {
		logrus.Error(err)
		return
	}

	doc := html.NewTokenizer(bytes.NewReader(res.Body()))

	var linkHref string
	var linkTexts []string
	for {
		tokenType := doc.Next()
		token := doc.Token()
		if tokenType == html.ErrorToken {
			break
		}

		switch tokenType {
		case html.StartTagToken:
			if token.DataAtom == atom.A {
				for _, attr := range token.Attr {
					if attr.Key == "href" {
						linkHref = attr.Val
						break
					}
				}
			}

		case html.TextToken:
			if linkHref != "" {
				linkTexts = append(linkTexts, token.Data)
			}

		case html.EndTagToken:
			if linkHref != "" && token.DataAtom == atom.A {
				// Copy params
				href := linkHref
				linkText := strings.Join(linkTexts, " ")

				// Reset params
				linkHref = ""
				linkTexts = nil

				// TODO Optimized decision tree
				for _, entry := range urlBlackList {
					if href == entry {
						goto nextToken
					}
				}
				for _, entry := range urlPartBlackList {
					if strings.Contains(href, entry) {
						goto nextToken
					}
				}
				for _, entry := range fileNameBlackList {
					if strings.Contains(linkText, entry) {
						goto nextToken
					}
				}

				links = append(links, href)
			}
		}

		nextToken:
	}

	atomic.AddInt64(&visited, 1)

	return
}

func fileInfo(u url.URL, f *File) (err error) {
	req := fasthttp.AcquireRequest()
	req.Header.SetMethod("HEAD")
	req.SetRequestURI(u.String())

	res := fasthttp.AcquireResponse()
	res.SkipBody = true
	defer fasthttp.ReleaseResponse(res)

	err = client.Do(req, res)
	fasthttp.ReleaseRequest(req)

	if err != nil { return }

	// TODO Inefficient af
	header := res.Header.Header()

	var k []byte
	var v []byte

	// Simple finite state machine
	state := 0
	for _, b := range header {
		switch state {
		case 0:
			if b == byte(':') {
				state = 1
			} else {
				k = append(k, b)
			}

		case 1:
			if b == byte(' ') {
				state = 2
			} else {
				return errors.New("bad request")
			}

		case 2:
			if b == byte('\r') {
				state = 3
			} else {
				v = append(v, b)
			}

		case 3:
			if b == byte('\n') {
				state = 0
				key := strings.ToLower(string(k))
				val := string(v)

				switch key {
				case "content-length":
					size, err := strconv.ParseInt(val, 10, 64)
					if err != nil { break }
					if size < 0 { break }
					f.Size = size

				case "last-modified":
					var err error
					f.MTime, err = time.Parse(time.RFC1123, val)
					if err == nil { break }
					f.MTime, err = time.Parse(time.RFC850, val)
					if err == nil { break }
					// TODO Parse asctime
					f.MTime, err = time.Parse("2006-01-02", val[:10])
					if err == nil { break }
				}

				k = k[:0]
				v = v[:0]
			} else {
				return errors.New("bad request")
			}
		}
	}

	return nil
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

var urlBlackList = [...]string {
	"",
	" ",
	".",
	"..",
	"/",
}

var urlPartBlackList = [...]string {
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

