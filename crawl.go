package main

import (
	"bytes"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/terorie/oddb-go/ds/redblackhash"
	"github.com/terorie/oddb-go/fasturl"
	"github.com/terorie/oddb-go/runes"
	"github.com/terorie/oddb-go/runespath"
	"github.com/valyala/fasthttp"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"path"
	"strconv"
	"strings"
	"time"
)

var client fasthttp.Client

func GetDir(j *Job, f *File) (links []fasturl.URL, err error) {
	f.IsDir = true
	f.Name = runespath.Base(j.Uri.Path)

	req := fasthttp.AcquireRequest()
	req.SetRequestURI(j.UriStr)

	res := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(res)

	err = client.Do(req, res)
	fasthttp.ReleaseRequest(req)

	if err != nil {
		logrus.Error(err)
		return
	}

	err = checkStatusCode(res.StatusCode())
	if err != nil { return }

	body := res.Body()
	doc := html.NewTokenizer(bytes.NewReader(body))

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

				var link fasturl.URL
				err = j.Uri.ParseRel(&link, []rune(href))
				if err != nil { continue }

				if !runes.Equals(link.Scheme, j.Uri.Scheme) ||
					!runes.Equals(link.Host, j.Uri.Host) ||
					 runes.Equals(link.Path, j.Uri.Path) ||
					!runes.HasPrefix(link.Path, j.Uri.Path) {
					continue
				}

				links = append(links, link)
			}
		}

		nextToken:
	}

	return
}

func GetFile(u fasturl.URL, f *File) (err error) {
	f.IsDir = false
	u.Path = []rune(path.Clean(string(u.Path)))
	f.Name = runespath.Base(u.Path)
	f.Path = runes.TrimRune(u.Path, '/')

	req := fasthttp.AcquireRequest()
	req.Header.SetMethod("HEAD")
	req.SetRequestURI(u.String())

	res := fasthttp.AcquireResponse()
	res.SkipBody = true
	defer fasthttp.ReleaseResponse(res)

	err = client.Do(req, res)
	fasthttp.ReleaseRequest(req)

	if err != nil { return }

	err = checkStatusCode(res.StatusCode())
	if err != nil { return }

	// TODO Inefficient af
	header := res.Header.Header()
	f.ParseHeader(header)

	return nil
}

func (f *File) HashDir(links []fasturl.URL) (o redblackhash.Key) {
	h, _ := blake2b.New256(nil)
	h.Write([]byte(string(f.Name)))
	for _, link := range links {
		fileName := runespath.Base(link.Path)
		h.Write([]byte(string(fileName)))
	}
	sum := h.Sum(nil)
	copy(o[:redblackhash.KeySize], sum)
	return
}

func (f *File) ParseHeader(h []byte) {
	var k1, k2 int
	var v1, v2 int

	// Simple finite state machine
	state := 0
	for i, b := range h {
		switch state {
		case 0:
			if b == byte(':') {
				state = 1
				k2 = i
			}

		case 1:
			state = 2

		case 2:
			state = 3
			v1 = i

		case 3:
			if b == byte('\r') {
				state = 4
			}

		case 4:
			state = 0
			v2 = i - 1

			key := string(h[k1:k2])
			val := string(h[v1:v2])
			k1 = i

			f.applyHeader(key, val)
		}
	}

}

func (f *File) applyHeader(k, v string) {
	switch k {
	case "content-length":
		size, err := strconv.ParseInt(v, 10, 64)
		if err != nil { break }
		if size < 0 { break }
		f.Size = size

	case "last-modified":
		var err error
		f.MTime, err = time.Parse(time.RFC1123, v)
		if err == nil { break }
		f.MTime, err = time.Parse(time.RFC850, v)
		if err == nil { break }
		// TODO Parse asctime
		f.MTime, err = time.Parse("2006-01-02", v[:10])
		if err == nil { break }
	}
}

func checkStatusCode(status int) error {
	switch status {
	case fasthttp.StatusOK:
		return nil

	case fasthttp.StatusTooManyRequests:
		return ErrRateLimit

	case fasthttp.StatusForbidden,
		fasthttp.StatusUnauthorized:
		return ErrForbidden

	default:
		return fmt.Errorf("got HTTP status %d", status)
	}
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

