package main

import (
	"bytes"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/terorie/oddb-go/ds/redblackhash"
	"github.com/terorie/oddb-go/fasturl"
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
	f.Name = path.Base(j.Uri.Path)

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
				err = j.Uri.ParseRel(&link, href)
				if err != nil { continue }

				if link.Scheme != j.Uri.Scheme ||
					link.Host != j.Uri.Host ||
					link.Path == j.Uri.Path ||
					!strings.HasPrefix(link.Path, j.Uri.Path) {
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
	u.Path = path.Clean(u.Path)
	f.Name = path.Base(u.Path)
	f.Path = strings.Trim(u.Path, "/")

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

	f.applyContentLength(string(res.Header.Peek("content-length")))
	f.applyLastModified(string(res.Header.Peek("last-modified")))

	return nil
}

func (f *File) HashDir(links []fasturl.URL) (o redblackhash.Key) {
	h, _ := blake2b.New256(nil)
	h.Write([]byte(f.Name))
	for _, link := range links {
		fileName := path.Base(link.Path)
		h.Write([]byte(fileName))
	}
	sum := h.Sum(nil)
	copy(o[:redblackhash.KeySize], sum)
	return
}

func (f *File) applyContentLength(v string) {
	if v == "" { return }
	size, err := strconv.ParseInt(v, 10, 64)
	if err != nil { return }
	if size < 0 { return }
	f.Size = size
}

func (f *File) applyLastModified(v string) {
	if v == "" { return }
	var err error
	f.MTime, err = time.Parse(time.RFC1123, v)
	if err == nil { return }
	f.MTime, err = time.Parse(time.RFC850, v)
	if err == nil { return }
	// TODO Parse asctime
	f.MTime, err = time.Parse("2006-01-02", v[:10])
	if err == nil { return }
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

