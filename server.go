package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
)

const (
	fileListChunkSize int64 = 5000000 // 5 mb
)

var serverClient = http.DefaultClient

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
	if err != nil { return }

	return
}

func PushResult(result *TaskResult) (err error) {
	if result.WebsiteId == 0 {
		// Not a real result, don't push
		return nil
	}

	filePath := filepath.Join(
		".", "crawled",
		fmt.Sprintf("%d.json", result.WebsiteId))

	defer os.Remove(filePath)

	f, err := os.Open(filePath)
	if os.IsNotExist(err) {
		err = fmt.Errorf("cannot upload result: %s does not exist", filePath)
		return
	} else if err != nil {
		return
	}
	defer f.Close()

	err = uploadChunks(result.WebsiteId, f)
	if err != nil {
		logrus.Errorf("Failed to upload file list: %s", err)
		err2 := CancelTask(result.WebsiteId)
		if err2 != nil {
			logrus.Error(err2)
		}
		return
	}

	err = uploadResult(result)
	if err != nil {
		logrus.Errorf("Failed to upload result: %s", err)
		err2 := CancelTask(result.WebsiteId)
		if err2 != nil {
			logrus.Error(err2)
		}
		return
	}

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
		n, err = io.CopyN(formFile, f, fileListChunkSize)
		if err != io.EOF {
			return err
		}
		if n < fileListChunkSize {
			err = nil
			// Break at end of iteration
			eof = true
		}

		multi.Close()

		req, err := http.NewRequest(
			http.MethodPost,
			config.ServerUrl + "/task/upload",
			&b)
		req.Header.Set("content-type", multi.FormDataContentType())
		if err != nil { return err }

		res, err := serverClient.Do(req)
		if err != nil { return err }
		res.Body.Close()

		if res.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to upload list part %d: %s",
				iter, res.Status)
		}

		logrus.Infof("Uploading file list part %d: %s",
			iter, res.Status)
	}
	return nil
}

func uploadResult(result *TaskResult) (err error) {
	resultEnc, err := json.Marshal(result)
	if err != nil { panic(err) }

	res, err := serverClient.PostForm(
		config.ServerUrl + "/task/complete",
		url.Values {
			"token": {config.Token},
			"result": {string(resultEnc)},
		},
	)
	if err != nil { return }
	res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("%s", res.Status)
	}

	return
}

func CancelTask(websiteId uint64) (err error) {
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
