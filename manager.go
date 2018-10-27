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
	"strings"
)

const (
	fileListChunkSize int64 = 5000000 // 5 mb
)

var serverClient = http.DefaultClient

func (c *Config) FetchTask() (t *Task, err error) {
	escToken, _ := json.Marshal(c.Token)
	payload := `{"token":` + string(escToken) + `}`

	req, err := http.NewRequest(
		http.MethodPost,
		c.ServerUrl + "/task/get",
		strings.NewReader(payload))
	if err != nil { return }

	res, err := serverClient.Do(req)
	if err != nil { return }
	defer res.Body.Close()

	if res.StatusCode != 200 {
		err = fmt.Errorf("http %s", res.Status)
		return
	}

	t = new(Task)
	err = json.NewDecoder(res.Body).Decode(t)
	if err != nil { return }

	return
}

func (c *Config) PushResult(result *TaskResult) (err error) {
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

	err = c.uploadChunks(result.WebsiteId, f)
	if err != nil {
		logrus.Errorf("Failed to upload file list: %s", err)
		err2 := c.CancelTask(result.WebsiteId)
		if err2 != nil {
			logrus.Error(err2)
		}
		return
	}

	err = c.uploadResult(result)
	if err != nil {
		logrus.Errorf("Failed to upload result: %s", err)
		err2 := c.CancelTask(result.WebsiteId)
		if err2 != nil {
			logrus.Error(err2)
		}
		return
	}

	return
}

func (c *Config) uploadChunks(websiteId uint64, f *os.File) (err error) {
	for iter := 1; iter > 0; iter++ {
		// TODO Stream with io.Pipe?
		var b bytes.Buffer

		multi := multipart.NewWriter(&b)

		// Set upload fields
		err = multi.WriteField("token", c.Token)
		if err != nil { return }
		err = multi.WriteField("website_id", fmt.Sprintf("%d", websiteId))
		if err != nil { return }

		// Copy chunk to file_list
		formFile, err := multi.CreateFormFile("file_list", "file_list")
		_, err = io.CopyN(formFile, f, fileListChunkSize)
		if err == io.EOF {
			break
		} else if err == io.ErrUnexpectedEOF {
			err = nil
			// Break at end of iteration
			iter = -420
		}

		req, err := http.NewRequest(
			http.MethodPost,
			c.ServerUrl + "/task/upload",
			&b)
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
	return
}

func (c *Config) uploadResult(result *TaskResult) (err error) {
	resultEnc, err := json.Marshal(result)
	if err != nil { panic(err) }

	payload := url.Values {
		"token": {c.Token},
		"result": {string(resultEnc)},
	}.Encode()

	req, err := http.NewRequest(
		http.MethodPost,
		c.ServerUrl + "/task/complete",
		strings.NewReader(payload))
	if err != nil { return }

	res, err := serverClient.Do(req)
	if err != nil { return }
	res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to cancel task: %s", res.Status)
	}

	return
}

func (c *Config) CancelTask(websiteId uint64) (err error) {
	form := url.Values{
		"token": {c.Token},
		"website_id": {strconv.FormatUint(websiteId, 10)},
	}
	encForm := form.Encode()

	req, err := http.NewRequest(
		http.MethodPost,
		c.ServerUrl + "/task/cancel",
		strings.NewReader(encForm))
	if err != nil { return }

	res, err := serverClient.Do(req)
	if err != nil { return }
	res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to cancel task: %s", res.Status)
	}

	return
}
