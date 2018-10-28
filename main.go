package main

import (
	"context"
	"net/url"
)

func main() {
	prepareConfig()
	readConfig()

	c := context.Background()

	remotes := make(chan *RemoteDir)
	go Schedule(c, remotes)

	u, _ := url.Parse("https://the-eye.eu/public/rom/")
	remote := NewRemoteDir(*u)

	globalWait.Add(1)
	remotes <- remote

	// Wait for all jobs to finish
	globalWait.Wait()
}
