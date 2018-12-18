package main

import (
	"bytes"
	"github.com/PuerkitoBio/goquery"
	"github.com/terorie/od-database-crawler/fasturl"
	"net/url"
	"strings"
	"testing"
)

func BenchmarkParseDir(b *testing.B) {
	for n := 0; n < b.N; n++ {
		var u fasturl.URL
		err := u.Parse("http://archive.ubuntu.com/ubuntu/indices/")
		if err != nil {
			b.Fatal("Failed to parse URL", err)
		}

		_, err = ParseDir([]byte(apache2Listing), &u)
		if err != nil {
			b.Fatal("Failed to extract links", err)
		}
	}
}

func BenchmarkParseDirReference(b *testing.B) {
	for n := 0; n < b.N; n++ {
		u, err := url.Parse("http://archive.ubuntu.com/ubuntu/indices/")
		if err != nil {
			b.Fatal("Failed to parse URL", err)
		}

		_, err = referenceParseDir([]byte(apache2Listing), u)
		if err != nil {
			b.Fatal("Failed to extract links", err)
		}
	}
}

func referenceParseDir(body []byte, baseUri *url.URL) (links []*url.URL, err error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil { return nil, err }

	doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		href, _ := s.Attr("href")

		sub, err := baseUri.Parse(href)
		if err != nil { return } // continue

		if !strings.HasPrefix(sub.String(), baseUri.String()) {
			return // continue
		}

		links = append(links, sub)
	})

	return
}
