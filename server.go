package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

var serverClient = http.Client {
	Timeout: config.ServerTimeout,
	Transport: new(ServerTripper),
}

var serverUserAgent = "od-database-crawler/" + rootCmd.Version

func FetchTask() (t *Task, err error) {
	res, err := serverClient.PostForm(
		config.ServerUrl + "/task/get",
		url.Values{ "token": {config.Token} })
	if err != nil { return }
	defer res.Body.Close()

	switch res.StatusCode {
	case 200:
		break
	case 404, 500:
		return nil, nil
	default:
		return nil, fmt.Errorf("http %s", res.Status)
	}

	t = new(Task)
	err = json.NewDecoder(res.Body).Decode(t)
	if _, ok := err.(*json.SyntaxError); ok {
		return nil, fmt.Errorf("/task/get returned invalid JSON")
	} else if err != nil { return }

	return
}

func PushResult(result *TaskResult, f *os.File) (err error) {
	if result.WebsiteId == 0 {
		// Not a real result, don't push
		return nil
	}

	// Rewind to the beginning of the file
	_, err = f.Seek(0, 0)
	if err != nil {
		return
	}

	err = uploadChunks(result.WebsiteId, f)
	if err != nil {
		logrus.Errorf("Failed to upload file list: %s", err)
		err2 := CancelTask(result.WebsiteId)
		if err2 != nil {
			logrus.Error(err2)
		}
		return
	}

	// Upload result ignoring errors
	uploadResult(result)

	return
}

func uploadChunks(websiteId uint64, f *os.File) error {
	eof := false
	for iter := 1; !eof; iter++ {
		// TODO Stream with io.Pipe?
		var b bytes.Buffer

		multi := multipart.NewWriter(&b)

		// Set upload fields
		var err error
		err = multi.WriteField("token", config.Token)
		if err != nil { return err }
		err = multi.WriteField("website_id", fmt.Sprintf("%d", websiteId))
		if err != nil { return err }

		// Copy chunk to file_list
		formFile, err := multi.CreateFormFile("file_list", "file_list")
		var n int64
		n, err = io.CopyN(formFile, f, config.ChunkSize)
		if err != io.EOF && err != nil {
			return err
		}
		if n == 0 {
			// Don't upload, no content
			return nil
		} else if n < config.ChunkSize {
			err = nil
			// Break at end of iteration
			eof = true
		}

		multi.Close()

		for retries := 0; retries < viper.GetInt(ConfUploadRetries); retries++ {
			if retries > 0 {
				// Error occurred, retry upload
				time.Sleep(viper.GetDuration(ConfUploadRetryInterval))
			}

			req, err := http.NewRequest(
				http.MethodPost,
				config.ServerUrl + "/task/upload",
				&b)
			req.Header.Set("content-type", multi.FormDataContentType())
			if err != nil { continue }

			res, err := serverClient.Do(req)
			if err != nil { continue }
			res.Body.Close()

			if res.StatusCode != http.StatusOK {
				logrus.WithField("status", res.Status).
					WithField("part", iter).
					Errorf("Upload failed")
				continue
			}

			// Upload successful
			break
		}

		logrus.WithField("id", websiteId).
			WithField("part", iter).
			Infof("Uploaded files chunk")
	}
	return nil
}

func uploadResult(result *TaskResult) (err error) {
	req := releaseTaskRequest {
		TaskId: int64(result.WebsiteId),
		ResultCode: result.ResultCode,
		// TODO What is verification
		Verification: 0,
	}

	resultEnc, err := json.Marshal(&req)
	if err != nil { panic(err) }

	res, err := serverClient.PostForm(
		config.ServerUrl + "/task/release",
		url.Values {
			"token": {config.Token},
			"result": {string(resultEnc)},
		},
	)
	if err != nil { return }
	res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return HttpError{res.StatusCode}
	}

	return
}

func CancelTask(websiteId uint64) (err error) {
	// TODO Remove, no endpoint in task_tracker

	res, err := serverClient.PostForm(
		config.ServerUrl + "/task/cancel",
		url.Values{
			"token": {config.Token},
			"website_id": {strconv.FormatUint(websiteId, 10)},
		},
	)
	if err != nil { return }
	res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to cancel task: %s", res.Status)
	}

	return
}

type ServerTripper struct{}

func (t *ServerTripper) RoundTrip(req *http.Request) (res *http.Response, err error) {
	req.Header.Set("User-Agent", serverUserAgent)
	return http.DefaultTransport.RoundTrip(req)
}

const mimeJSON = "application/json"

// https://github.com/simon987/task_tracker/blob/master/api/models.go

type submitTaskRequest struct {
	Project           int64  `json:"project"`
	MaxRetries        int64  `json:"max_retries"`
	Recipe            string `json:"recipe"`
	Priority          int64  `json:"priority"`
	MaxAssignTime     int64  `json:"max_assign_time"`
	Hash64            int64  `json:"hash_u64"`
	UniqueString      string `json:"unique_string"`
	VerificationCount int64  `json:"verification_count"`
}

type releaseTaskRequest struct {
	TaskId       int64      `json:"task_id"`
	ResultCode   ResultCode `json:"result"`
	Verification int64      `json:"verification"`
}
