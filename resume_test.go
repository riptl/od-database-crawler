package main

import (
	"bytes"
	"github.com/terorie/od-database-crawler/fasturl"
	"testing"
	"time"
)

func TestResumeTasks_Empty(t *testing.T) {
	start := time.Now().Add(-1 * time.Minute)
	od := OD {
		Task: Task {
			WebsiteId: 213,
			Url: "https://the-eye.eu/public/",
		},
		Result: TaskResult {
			StartTime: start,
			StartTimeUnix: start.Unix(),
			EndTimeUnix: time.Now().Unix(),
			WebsiteId: 213,
		},
		InProgress: 0,
		BaseUri: fasturl.URL {
			Scheme: fasturl.SchemeHTTPS,
			Host: "the-eye.eu",
			Path: "/public/",
		},
	}
	od.WCtx.OD = &od

	var b bytes.Buffer
	var err error
	err = writePauseFile(&od, &b)
	if err != nil {
		t.Fatal(err)
	}

	buf := b.Bytes()

	var od2 OD

	b2 := bytes.NewBuffer(buf)
	err = readPauseFile(&od2, b2)
	if err != nil {
		t.Fatal(err)
	}
}
