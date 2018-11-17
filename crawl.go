package main

import (
	"bytes"
	"crypto/tls"
	"github.com/terorie/od-database-crawler/ds/redblackhash"
	"github.com/terorie/od-database-crawler/fasturl"
	"github.com/valyala/fasthttp"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/net/html"
	"path"
	"strconv"
	"strings"
	"time"
)

var client = fasthttp.Client {
	TLSConfig: &tls.Config{
		InsecureSkipVerify: true,
	},
}

func GetDir(j *Job, f *File) (links []fasturl.URL, err error) {
	f.IsDir = true
	f.Name = path.Base(j.Uri.Path)

	req := fasthttp.AcquireRequest()
	req.SetRequestURI(j.UriStr)

	res := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(res)

	err = client.DoTimeout(req, res, config.Timeout)
	fasthttp.ReleaseRequest(req)

	if err != nil {
		return
	}

	err = checkStatusCode(res.StatusCode())
	if err != nil {
		return
	}

	body := res.Body()
	doc := html.NewTokenizer(bytes.NewReader(body))

	var linkHref string
	for {
		tokenType := doc.Next()
		if tokenType == html.ErrorToken {
			break
		}

		switch tokenType {
		case html.StartTagToken:
			name, hasAttr := doc.TagName()
			if len(name) == 1 && name[0] == 'a' {
				for hasAttr {
					var ks, vs []byte
					ks, vs, hasAttr = doc.TagAttr()
					if bytes.Equal(ks, []byte("href")) {
						// TODO Check escape
						linkHref = string(vs)
						break
					}
				}
			}

		case html.EndTagToken:
			name, _ := doc.TagName()
			if len(name) == 1 && name[0] == 'a' {
				// Copy params
				href := linkHref

				// Reset params
				linkHref = ""

				if strings.LastIndexByte(href, '?') != -1 {
					goto nextToken
				}

				switch href {
				case "", " ", ".", "..", "/":
					goto nextToken
				}

				if strings.Contains(href, "../") {
					goto nextToken
				}

				var link fasturl.URL
				err = j.Uri.ParseRel(&link, href)
				if err != nil {
					continue
				}

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
	f.Path = strings.Trim(path.Dir(u.Path), "/")

	req := fasthttp.AcquireRequest()
	req.Header.SetMethod("HEAD")
	req.SetRequestURI(u.String())

	res := fasthttp.AcquireResponse()
	res.SkipBody = true
	defer fasthttp.ReleaseResponse(res)

	err = client.DoTimeout(req, res, config.Timeout)
	fasthttp.ReleaseRequest(req)

	if err != nil {
		return
	}

	err = checkStatusCode(res.StatusCode())
	if err != nil {
		return
	}

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
	if v == "" {
		return
	}
	size, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return
	}
	if size < 0 {
		return
	}
	f.Size = size
}

// TODO Cleanup
func (f *File) applyLastModified(v string) {
	if v == "" {
		return
	}
	var t time.Time
	var err error
	t, err = time.Parse(time.RFC1123, v)
	if err == nil {
		f.MTime = t.Unix()
		return
	}
	t, err = time.Parse(time.RFC850, v)
	if err == nil {
		f.MTime = t.Unix()
		return
	}
	// TODO Parse asctime
	t, err = time.Parse("2006-01-02", v[:10])
	if err == nil {
		f.MTime = t.Unix()
		return
	}
}

func checkStatusCode(status int) error {
	switch status {
	case fasthttp.StatusOK:
		return nil
	default:
		return &HttpError{status}
	}
}
