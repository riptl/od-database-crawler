package main

import (
	"context"
	"net/url"
)

func init() {
	prepareConfig()
}

func main() {
	readConfig()

	c := context.Background()

	remotes := make(chan *OD)
	go Schedule(c, remotes)

	u, _ := url.Parse("http://mine.terorie.com:420/")
	remote := NewRemoteDir(*u)

	globalWait.Add(1)
	remotes <- remote

	// Wait for all jobs to finish
	globalWait.Wait()
}
