package main

import (
	"github.com/terorie/od-database-crawler/fasturl"
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
