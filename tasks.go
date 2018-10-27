package main

import "time"

type Task struct {
	WebsiteId int `json:"website_id"`
	Url string `json:"url"`
}

type TaskResult struct {
	StatusCode int `json:"status_code"`
	FileCount uint64 `json:"file_count"`
	StartTime time.Time `json:"start_time"`
	EndTime time.Time `json:"end_time"`
	WebsiteId uint64 `json:"website_id"`
}
